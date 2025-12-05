package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"licet/internal/config"
	"licet/internal/database"
	"licet/internal/handlers"
	appmiddleware "licet/internal/middleware"
	"licet/internal/scheduler"
	"licet/internal/services"
	"licet/web"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	log "github.com/sirupsen/logrus"
)

// Version is set at build time via ldflags
var Version = "dev"

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logging
	setupLogging(cfg)

	log.WithField("version", Version).Info("Starting Licet")

	// Initialize database
	db, err := database.New(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db, cfg.Database.Type); err != nil {
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
	r := setupRouter(cfg, licenseService, alertService, Version)

	// Start HTTP/HTTPS server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Validate TLS configuration if enabled
	if cfg.Server.TLSEnabled {
		if cfg.Server.TLSCertFile == "" || cfg.Server.TLSKeyFile == "" {
			log.Fatal("TLS is enabled but certificate or key file path is not configured")
		}
		if _, err := os.Stat(cfg.Server.TLSCertFile); os.IsNotExist(err) {
			log.Fatalf("TLS certificate file not found: %s", cfg.Server.TLSCertFile)
		}
		if _, err := os.Stat(cfg.Server.TLSKeyFile); os.IsNotExist(err) {
			log.Fatalf("TLS key file not found: %s", cfg.Server.TLSKeyFile)
		}
	}

	// Start server in goroutine
	go func() {
		if cfg.Server.TLSEnabled {
			log.Infof("Server listening on https://%s (TLS enabled)", srv.Addr)
			if err := srv.ListenAndServeTLS(cfg.Server.TLSCertFile, cfg.Server.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTPS server: %v", err)
			}
		} else {
			log.Infof("Server listening on http://%s", srv.Addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to start HTTP server: %v", err)
			}
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

func setupRouter(cfg *config.Config, licenseService *services.LicenseService, alertService *services.AlertService, version string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Rate limiting middleware
	if cfg.RateLimit.Enabled {
		rateLimitConfig := appmiddleware.RateLimitConfig{
			RequestsPerMinute: cfg.RateLimit.RequestsPerMinute,
			BurstSize:         cfg.RateLimit.BurstSize,
			Enabled:           cfg.RateLimit.Enabled,
			WhitelistedIPs:    cfg.RateLimit.WhitelistedIPs,
			WhitelistedPaths:  cfg.RateLimit.WhitelistedPaths,
		}
		rateLimiter := appmiddleware.NewRateLimiter(rateLimitConfig)
		r.Use(appmiddleware.RateLimitMiddleware(rateLimiter))
		log.WithFields(log.Fields{
			"requests_per_minute": rateLimitConfig.RequestsPerMinute,
			"burst_size":          rateLimitConfig.BurstSize,
		}).Info("Rate limiting enabled")
	}

	// Setup cache for API responses
	var cache *appmiddleware.Cache
	if cfg.Cache.Enabled {
		cacheConfig := appmiddleware.CacheConfig{
			DefaultTTL: time.Duration(cfg.Cache.TTLSeconds) * time.Second,
			MaxEntries: cfg.Cache.MaxEntries,
			Enabled:    cfg.Cache.Enabled,
		}
		cache = appmiddleware.NewCache(cacheConfig)
		log.WithFields(log.Fields{
			"ttl_seconds": cfg.Cache.TTLSeconds,
			"max_entries": cfg.Cache.MaxEntries,
		}).Info("Response caching enabled")
	}

	// CORS - use configured origins or default to localhost
	corsOrigins := cfg.Server.CORSOrigins
	if len(corsOrigins) == 0 {
		corsOrigins = []string{fmt.Sprintf("http://localhost:%d", cfg.Server.Port)}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   corsOrigins,
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
	webHandler := handlers.NewWebHandler(licenseService, alertService, cfg, version)
	r.Get("/", webHandler.Index)
	r.Get("/details/{server}", webHandler.Details)
	r.Get("/expiration/{server}", webHandler.Expiration)
	r.Get("/utilization", webHandler.Utilization)
	r.Get("/utilization/trends", webHandler.UtilizationTrends)
	r.Get("/utilization/analytics", webHandler.UtilizationAnalytics)
	r.Get("/utilization/stats", webHandler.UtilizationStats)
	r.Get("/denials", webHandler.Denials)
	r.Get("/alerts", webHandler.Alerts)
	r.Get("/statistics", webHandler.Statistics)
	r.Get("/settings", webHandler.Settings)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Apply caching middleware to read-only API endpoints
		if cache != nil {
			r.Group(func(r chi.Router) {
				r.Use(appmiddleware.CacheMiddleware(cache, time.Duration(cfg.Cache.TTLSeconds)*time.Second))

				r.Get("/servers", handlers.ListServers(licenseService))
				r.Get("/servers/{server}/status", handlers.GetServerStatus(licenseService))
				r.Get("/servers/{server}/features", handlers.GetServerFeatures(licenseService))
				r.Get("/servers/{server}/users", handlers.GetServerUsers(licenseService))
				r.Get("/features/{feature}/usage", handlers.GetFeatureUsage(licenseService))
				r.Get("/alerts", handlers.GetAlerts(alertService))

				// Utilization endpoints (cached)
				r.Get("/utilization/current", handlers.GetCurrentUtilization(licenseService))
				r.Get("/utilization/history", handlers.GetUtilizationHistory(licenseService))
				r.Get("/utilization/stats", handlers.GetUtilizationStats(licenseService))
				r.Get("/utilization/heatmap", handlers.GetUtilizationHeatmap(licenseService))
				r.Get("/utilization/predictions", handlers.GetPredictiveAnalytics(licenseService))

				// Enhanced statistics endpoints (cached)
				r.Get("/statistics/enhanced", handlers.GetEnhancedStatistics(licenseService))
				r.Get("/statistics/trends", handlers.GetTrendAnalysis(licenseService))
				r.Get("/statistics/capacity", handlers.GetCapacityPlanningReport(licenseService))
			})
		} else {
			r.Get("/servers", handlers.ListServers(licenseService))
			r.Get("/servers/{server}/status", handlers.GetServerStatus(licenseService))
			r.Get("/servers/{server}/features", handlers.GetServerFeatures(licenseService))
			r.Get("/servers/{server}/users", handlers.GetServerUsers(licenseService))
			r.Get("/features/{feature}/usage", handlers.GetFeatureUsage(licenseService))
			r.Get("/alerts", handlers.GetAlerts(alertService))

			// Utilization endpoints (not cached)
			r.Get("/utilization/current", handlers.GetCurrentUtilization(licenseService))
			r.Get("/utilization/history", handlers.GetUtilizationHistory(licenseService))
			r.Get("/utilization/stats", handlers.GetUtilizationStats(licenseService))
			r.Get("/utilization/heatmap", handlers.GetUtilizationHeatmap(licenseService))
			r.Get("/utilization/predictions", handlers.GetPredictiveAnalytics(licenseService))

			// Enhanced statistics endpoints (not cached)
			r.Get("/statistics/enhanced", handlers.GetEnhancedStatistics(licenseService))
			r.Get("/statistics/trends", handlers.GetTrendAnalysis(licenseService))
			r.Get("/statistics/capacity", handlers.GetCapacityPlanningReport(licenseService))
		}

		// Non-cached endpoints (mutations and health check)
		r.Post("/servers", handlers.AddServer(cfg))
		r.Delete("/servers", handlers.DeleteServer(cfg))
		r.Post("/servers/test", handlers.TestServerConnection(cfg, licenseService))
		r.Get("/utilities/check", handlers.CheckUtilities())
		r.Post("/settings/email", handlers.UpdateEmailSettings(cfg))
		r.Post("/settings/alerts", handlers.UpdateAlertSettings(cfg))
		r.Get("/health", handlers.Health(version))

		// Export endpoints
		if cfg.Export.Enabled {
			exportHandler := handlers.NewExportHandler(licenseService)
			r.Route("/export", func(r chi.Router) {
				r.Get("/servers", exportHandler.ExportServers)
				r.Get("/features", exportHandler.ExportFeatures)
				r.Get("/utilization", exportHandler.ExportUtilization)
				r.Get("/utilization/history", exportHandler.ExportUtilizationHistory)
				r.Get("/stats", exportHandler.ExportStats)
				r.Get("/report", exportHandler.ExportReport)
			})
			log.Info("Data export endpoints enabled")
		}

		// Cache stats endpoint (for monitoring)
		if cache != nil {
			r.Get("/cache/stats", func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				stats := cache.Stats()
				w.Write([]byte(fmt.Sprintf(`{"cache": %v}`, stats)))
			})
		}
	})

	return r
}
