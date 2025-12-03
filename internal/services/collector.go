package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"licet/internal/config"
	"licet/internal/models"
)

type CollectorService struct {
	db             *sqlx.DB
	cfg            *config.Config
	licenseService *LicenseService
}

func NewCollectorService(db *sqlx.DB, cfg *config.Config, licenseService *LicenseService) *CollectorService {
	return &CollectorService{
		db:             db,
		cfg:            cfg,
		licenseService: licenseService,
	}
}

func (s *CollectorService) CollectAll() error {
	log.Info("Starting license data collection")

	servers, err := s.licenseService.GetAllServers()
	if err != nil {
		return err
	}

	// Use parallel collection with worker pool
	// Limit concurrent queries to avoid overwhelming license servers
	maxWorkers := 5
	if len(servers) < maxWorkers {
		maxWorkers = len(servers)
	}

	var wg sync.WaitGroup
	serverChan := make(chan models.LicenseServer, len(servers))
	errorChan := make(chan error, len(servers))

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for server := range serverChan {
				if err := s.CollectServer(server); err != nil {
					log.Errorf("Failed to collect data for %s: %v", server.Hostname, err)
					errorChan <- err
				}
			}
		}()
	}

	// Send servers to workers
	for _, server := range servers {
		serverChan <- server
	}
	close(serverChan)

	// Wait for all workers to complete
	wg.Wait()
	close(errorChan)

	// Check if any errors occurred
	errorCount := len(errorChan)
	if errorCount > 0 {
		log.Warnf("Collection completed with %d errors", errorCount)
	}

	log.Info("License data collection completed")
	return nil
}

func (s *CollectorService) CollectServer(server models.LicenseServer) error {
	log.Debugf("Collecting data for %s (%s)", server.Hostname, server.Type)

	result, err := s.licenseService.QueryServer(server.Hostname, server.Type)
	if err != nil {
		log.Errorf("Query failed for %s: %v", server.Hostname, err)
		return err
	}

	log.Infof("Collected %d features and %d users from %s",
		len(result.Features), len(result.Users), server.Hostname)

	return nil
}

func (s *CollectorService) CheckExpirations() error {
	log.Info("Checking for expiring licenses")

	ctx := context.Background()
	features, err := s.licenseService.GetExpiringFeatures(ctx, s.cfg.Alerts.LeadTimeDays)
	if err != nil {
		return err
	}

	if len(features) == 0 {
		log.Debug("No expiring licenses found")
		return nil
	}

	log.Infof("Found %d expiring features", len(features))

	// Create alerts for expiring licenses
	alertService := NewAlertService(s.db, s.cfg)

	for _, feature := range features {
		daysToExpire := int(time.Until(feature.ExpirationDate).Hours() / 24)

		if daysToExpire < 0 {
			continue // Already expired
		}

		severity := "info"
		if daysToExpire <= 3 {
			severity = "critical"
		} else if daysToExpire <= 7 {
			severity = "warning"
		}

		alert := &models.Alert{
			ServerHostname: feature.ServerHostname,
			FeatureName:    feature.Name,
			AlertType:      "expiration",
			Message: fmt.Sprintf("License '%s' on %s expires in %d days (%s)",
				feature.Name, feature.ServerHostname, daysToExpire, feature.ExpirationDate.Format("2006-01-02")),
			Severity: severity,
		}

		// Check throttle before creating alert
		if !alertService.CheckThrottle(feature.ServerHostname, "expiration") {
			if err := alertService.CreateAlert(ctx, alert); err != nil {
				log.Errorf("Failed to create alert: %v", err)
			}
		}
	}

	return nil
}
