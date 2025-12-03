package services

import (
	"context"

	"github.com/jmoiron/sqlx"
	"licet/internal/config"
	"licet/internal/models"
)

// LicenseService is a facade that combines Query, Storage, and Analytics services
// Provides backward compatibility with existing code
type LicenseService struct {
	Query     *QueryService
	Storage   *StorageService
	Analytics *AnalyticsService
}

// NewLicenseService creates a new license service facade
func NewLicenseService(db *sqlx.DB, cfg *config.Config) *LicenseService {
	dbType := cfg.Database.Type
	storage := NewStorageService(db, dbType)
	query := NewQueryService(cfg, storage)
	analytics := NewAnalyticsService(db, storage, dbType)

	return &LicenseService{
		Query:     query,
		Storage:   storage,
		Analytics: analytics,
	}
}

// GetAllServers delegates to QueryService
func (s *LicenseService) GetAllServers() ([]models.LicenseServer, error) {
	return s.Query.GetAllServers()
}

// QueryServer delegates to QueryService
func (s *LicenseService) QueryServer(hostname, serverType string) (models.ServerQueryResult, error) {
	return s.Query.QueryServer(hostname, serverType)
}

// GetFeatures delegates to StorageService
func (s *LicenseService) GetFeatures(ctx context.Context, hostname string) ([]models.Feature, error) {
	return s.Storage.GetFeatures(ctx, hostname)
}

// GetFeaturesWithExpiration delegates to StorageService
func (s *LicenseService) GetFeaturesWithExpiration(ctx context.Context, hostname string) ([]models.Feature, error) {
	return s.Storage.GetFeaturesWithExpiration(ctx, hostname)
}

// GetExpiringFeatures delegates to StorageService
func (s *LicenseService) GetExpiringFeatures(ctx context.Context, days int) ([]models.Feature, error) {
	return s.Storage.GetExpiringFeatures(ctx, days)
}

// GetFeatureUsageHistory delegates to StorageService
func (s *LicenseService) GetFeatureUsageHistory(ctx context.Context, hostname, featureName string, days int) ([]models.FeatureUsage, error) {
	return s.Storage.GetFeatureUsageHistory(ctx, hostname, featureName, days)
}

// GetCurrentUtilization delegates to AnalyticsService
func (s *LicenseService) GetCurrentUtilization(ctx context.Context, serverFilter string) ([]models.UtilizationData, error) {
	return s.Analytics.GetCurrentUtilization(ctx, serverFilter)
}

// GetUtilizationHistory delegates to AnalyticsService
func (s *LicenseService) GetUtilizationHistory(ctx context.Context, server, feature string, days int) ([]models.UtilizationHistoryPoint, error) {
	return s.Analytics.GetUtilizationHistory(ctx, server, feature, days)
}

// GetUtilizationStats delegates to AnalyticsService
func (s *LicenseService) GetUtilizationStats(ctx context.Context, server string, days int) ([]models.UtilizationStats, error) {
	return s.Analytics.GetUtilizationStats(ctx, server, days)
}

// GetHeatmapData delegates to AnalyticsService
func (s *LicenseService) GetHeatmapData(ctx context.Context, server string, days int) ([]models.HeatmapData, error) {
	return s.Analytics.GetHeatmapData(ctx, server, days)
}

// GetPredictiveAnalytics delegates to AnalyticsService
func (s *LicenseService) GetPredictiveAnalytics(ctx context.Context, server, feature string, days int) (*models.PredictiveAnalytics, error) {
	return s.Analytics.GetPredictiveAnalytics(ctx, server, feature, days)
}
