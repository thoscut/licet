package services

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/models"
	log "github.com/sirupsen/logrus"
	"gopkg.in/gomail.v2"
)

type AlertService struct {
	db  *sqlx.DB
	cfg *config.Config
}

func NewAlertService(db *sqlx.DB, cfg *config.Config) *AlertService {
	return &AlertService{
		db:  db,
		cfg: cfg,
	}
}

func (s *AlertService) CreateAlert(alert *models.Alert) error {
	query := `
		INSERT INTO alerts (server_hostname, feature_name, alert_type, message, severity, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.Exec(query,
		alert.ServerHostname,
		alert.FeatureName,
		alert.AlertType,
		alert.Message,
		alert.Severity,
		time.Now(),
	)

	return err
}

func (s *AlertService) GetUnsentAlerts() ([]models.Alert, error) {
	var alerts []models.Alert
	query := `SELECT * FROM alerts WHERE sent = 0 ORDER BY created_at ASC`
	err := s.db.Select(&alerts, query)
	return alerts, err
}

func (s *AlertService) MarkAlertSent(alertID int64) error {
	query := `UPDATE alerts SET sent = 1, sent_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, time.Now(), alertID)
	return err
}

func (s *AlertService) SendAlerts() error {
	if !s.cfg.Email.Enabled || !s.cfg.Alerts.Enabled {
		log.Debug("Email alerts are disabled")
		return nil
	}

	alerts, err := s.GetUnsentAlerts()
	if err != nil {
		return fmt.Errorf("failed to get unsent alerts: %w", err)
	}

	if len(alerts) == 0 {
		log.Debug("No alerts to send")
		return nil
	}

	// Group alerts by type for better email formatting
	for _, alert := range alerts {
		if err := s.sendAlert(&alert); err != nil {
			log.Errorf("Failed to send alert %d: %v", alert.ID, err)
			continue
		}

		if err := s.MarkAlertSent(alert.ID); err != nil {
			log.Errorf("Failed to mark alert %d as sent: %v", alert.ID, err)
		}
	}

	return nil
}

func (s *AlertService) sendAlert(alert *models.Alert) error {
	m := gomail.NewMessage()
	m.SetHeader("From", s.cfg.Email.From)

	// Determine recipients based on alert severity
	recipients := s.cfg.Email.To
	if alert.Severity == "critical" {
		recipients = append(recipients, s.cfg.Email.Alerts...)
	}

	m.SetHeader("To", recipients...)

	subject := fmt.Sprintf("[%s] License Alert: %s", alert.Severity, alert.AlertType)
	m.SetHeader("Subject", subject)

	body := fmt.Sprintf(`
License Alert

Server: %s
Feature: %s
Type: %s
Severity: %s
Time: %s

Message:
%s

--
Licet (Go Edition)
`,
		alert.ServerHostname,
		alert.FeatureName,
		alert.AlertType,
		alert.Severity,
		alert.CreatedAt.Format(time.RFC3339),
		alert.Message,
	)

	m.SetBody("text/plain", body)

	d := gomail.NewDialer(
		s.cfg.Email.SMTPHost,
		s.cfg.Email.SMTPPort,
		s.cfg.Email.Username,
		s.cfg.Email.Password,
	)

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Infof("Alert sent: %s - %s", alert.AlertType, alert.ServerHostname)
	return nil
}

func (s *AlertService) CheckThrottle(hostname, alertType string) bool {
	cutoff := time.Now().Add(-time.Duration(s.cfg.Alerts.ResendIntervalMin) * time.Minute)

	var count int
	query := `
		SELECT COUNT(*) FROM alert_events
		WHERE hostname = ? AND type = ? AND datetime > ?
	`

	err := s.db.Get(&count, query, hostname, alertType, cutoff)
	if err != nil {
		log.Errorf("Failed to check throttle: %v", err)
		return false
	}

	if count > 0 {
		log.Debugf("Alert throttled for %s:%s", hostname, alertType)
		return true
	}

	// Record this alert check
	insertQuery := `INSERT INTO alert_events (datetime, type, hostname) VALUES (?, ?, ?)`
	_, err = s.db.Exec(insertQuery, time.Now(), alertType, hostname)
	if err != nil {
		log.Errorf("Failed to record alert event: %v", err)
	}

	return false
}
