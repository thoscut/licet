package services

import (
	"context"
	"fmt"
	"time"

	mail "github.com/wneessen/go-mail"

	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	"licet/internal/config"
	"licet/internal/models"
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

func (s *AlertService) CreateAlert(ctx context.Context, alert *models.Alert) error {
	query := `
		INSERT INTO alerts (server_hostname, feature_name, alert_type, message, severity, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	_, err := s.db.ExecContext(ctx, query,
		alert.ServerHostname,
		alert.FeatureName,
		alert.AlertType,
		alert.Message,
		alert.Severity,
		time.Now(),
	)

	return err
}

func (s *AlertService) GetUnsentAlerts(ctx context.Context) ([]models.Alert, error) {
	var alerts []models.Alert
	query := `SELECT * FROM alerts WHERE sent = 0 ORDER BY created_at ASC`
	err := s.db.SelectContext(ctx, &alerts, query)
	return alerts, err
}

// GetActiveAlerts returns all alerts from the last 30 days, both sent and unsent
func (s *AlertService) GetActiveAlerts(ctx context.Context) ([]models.Alert, error) {
	var alerts []models.Alert
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	query := `SELECT * FROM alerts WHERE created_at > ? ORDER BY created_at DESC`
	err := s.db.SelectContext(ctx, &alerts, query, thirtyDaysAgo)
	return alerts, err
}

func (s *AlertService) MarkAlertSent(ctx context.Context, alertID int64) error {
	query := `UPDATE alerts SET sent = 1, sent_at = ? WHERE id = ?`
	_, err := s.db.ExecContext(ctx, query, time.Now(), alertID)
	return err
}

func (s *AlertService) SendAlerts() error {
	if !s.cfg.Email.Enabled || !s.cfg.Alerts.Enabled {
		log.Debug("Email alerts are disabled")
		return nil
	}

	ctx := context.Background()
	alerts, err := s.GetUnsentAlerts(ctx)
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

		if err := s.MarkAlertSent(ctx, alert.ID); err != nil {
			log.Errorf("Failed to mark alert %d as sent: %v", alert.ID, err)
		}
	}

	return nil
}

func (s *AlertService) sendAlert(alert *models.Alert) error {
	// Create new message
	m := mail.NewMsg()

	// Set From header
	if err := m.From(s.cfg.Email.From); err != nil {
		return fmt.Errorf("failed to set From header: %w", err)
	}

	// Determine recipients based on alert severity
	recipients := s.cfg.Email.To
	if alert.Severity == "critical" {
		recipients = append(recipients, s.cfg.Email.Alerts...)
	}

	// Set To header
	if err := m.To(recipients...); err != nil {
		return fmt.Errorf("failed to set To header: %w", err)
	}

	// Set subject
	subject := fmt.Sprintf("[%s] License Alert: %s", alert.Severity, alert.AlertType)
	m.Subject(subject)

	// Set body
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

	m.SetBodyString(mail.TypeTextPlain, body)

	// Create client
	client, err := mail.NewClient(s.cfg.Email.SMTPHost,
		mail.WithPort(s.cfg.Email.SMTPPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(s.cfg.Email.Username),
		mail.WithPassword(s.cfg.Email.Password),
	)
	if err != nil {
		return fmt.Errorf("failed to create mail client: %w", err)
	}

	// Send the mail
	if err := client.DialAndSend(m); err != nil {
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
