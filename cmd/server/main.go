package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/database"
	"github.com/thoscut/licet/internal/handlers"
	"github.com/thoscut/licet/internal/scheduler"
	"github.com/thoscut/licet/internal/services"
	"github.com/thoscut/licet/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	log "github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg)

	log.Info("Starting Licet (Go Edition)")

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize services
	licenseService := services.NewLicenseService(db, cfg)
	alertService := services.NewAlertService(db, cfg)
	collectorService := services.NewCollectorService(db, cfg, licenseService)

	// Initialize scheduler for background tasks
	sched := scheduler.New(cfg, collectorService, alertService)
	sched.Start()
	defer sched.Stop()

	// Setup HTTP router
	r := setupRouter(cfg, licenseService, alertService)

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Infof("Server listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Errorf("Server forced to shutdown: %v", err)
	}

	log.Info("Server exited")
}

func setupLogging(cfg *config.Config) {
	level, err := log.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = log.InfoLevel
	}
	log.SetLevel(level)

	if cfg.Logging.Format == "json" {
		log.SetFormatter(&log.JSONFormatter{})
	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}
}

func setupRouter(cfg *config.Config, licenseService *services.LicenseService, alertService *services.AlertService) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// Static files from embedded filesystem
	staticFS := web.GetStaticFS()
	fileServer := http.FileServer(http.FS(staticFS))
	r.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Web handlers
	webHandler := handlers.NewWebHandler(licenseService, alertService, cfg)
	r.Get("/", webHandler.Index)
	r.Get("/details/{server}", webHandler.Details)
	r.Get("/expiration/{server}", webHandler.Expiration)
	r.Get("/utilization", webHandler.Utilization)
	r.Get("/denials", webHandler.Denials)
	r.Get("/alerts", webHandler.Alerts)
	r.Get("/statistics", webHandler.Statistics)
	r.Get("/settings", webHandler.Settings)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/servers", handlers.ListServers(licenseService))
		r.Post("/servers", handlers.AddServer(cfg))
		r.Delete("/servers", handlers.DeleteServer(cfg))
		r.Post("/servers/test", handlers.TestServerConnection(cfg, licenseService))
		r.Get("/servers/{server}/status", handlers.GetServerStatus(licenseService))
		r.Get("/servers/{server}/features", handlers.GetServerFeatures(licenseService))
		r.Get("/servers/{server}/users", handlers.GetServerUsers(licenseService))
		r.Get("/features/{feature}/usage", handlers.GetFeatureUsage(licenseService))
		r.Get("/alerts", handlers.GetAlerts(alertService))
		r.Get("/utilities/check", handlers.CheckUtilities())
		r.Post("/settings/email", handlers.UpdateEmailSettings(cfg))
		r.Post("/settings/alerts", handlers.UpdateAlertSettings(cfg))
		r.Get("/health", handlers.Health())
	})

	return r
}
