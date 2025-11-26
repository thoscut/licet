package services

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/models"
	log "github.com/sirupsen/logrus"
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

	for _, server := range servers {
		if err := s.CollectServer(server); err != nil {
			log.Errorf("Failed to collect data for %s: %v", server.Hostname, err)
		}
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

	features, err := s.licenseService.GetExpiringFeatures(s.cfg.Alerts.LeadTimeDays)
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
			Message:        fmt.Sprintf("License '%s' on %s expires in %d days (%s)",
				feature.Name, feature.ServerHostname, daysToExpire, feature.ExpirationDate.Format("2006-01-02")),
			Severity:       severity,
		}

		// Check throttle before creating alert
		if !alertService.CheckThrottle(feature.ServerHostname, "expiration") {
			if err := alertService.CreateAlert(alert); err != nil {
				log.Errorf("Failed to create alert: %v", err)
			}
		}
	}

	return nil
}
