package scheduler

import (
	"fmt"

	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/services"
)

type Scheduler struct {
	cron             *cron.Cron
	collectorService *services.CollectorService
	alertService     *services.AlertService
	cfg              *config.Config
}

func New(cfg *config.Config, collector *services.CollectorService, alert *services.AlertService) *Scheduler {
	return &Scheduler{
		cron:             cron.New(),
		collectorService: collector,
		alertService:     alert,
		cfg:              cfg,
	}
}

func (s *Scheduler) Start() {
	log.Info("Starting scheduler")

	// Collect license usage every N minutes (default: 5)
	collectionSchedule := fmt.Sprintf("*/%d * * * *", s.cfg.RRD.CollectionInterval)
	s.cron.AddFunc(collectionSchedule, func() {
		log.Debug("Running scheduled license collection")
		if err := s.collectorService.CollectAll(); err != nil {
			log.Errorf("Collection job failed: %v", err)
		}
	})

	// Check for expiring licenses daily at 2 AM
	s.cron.AddFunc("0 2 * * *", func() {
		log.Debug("Running expiration check")
		if err := s.collectorService.CheckExpirations(); err != nil {
			log.Errorf("Expiration check failed: %v", err)
		}
	})

	// Send alerts every 5 minutes
	if s.cfg.Alerts.Enabled {
		s.cron.AddFunc("*/5 * * * *", func() {
			log.Debug("Running alert sending job")
			if err := s.alertService.SendAlerts(); err != nil {
				log.Errorf("Alert sending failed: %v", err)
			}
		})
	}

	s.cron.Start()
	log.Info("Scheduler started")
}

func (s *Scheduler) Stop() {
	log.Info("Stopping scheduler")
	s.cron.Stop()
}
