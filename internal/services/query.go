package services

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/thoscut/licet/internal/config"
	"github.com/thoscut/licet/internal/models"
	"github.com/thoscut/licet/internal/parsers"
)

// QueryService handles license server query operations
type QueryService struct {
	cfg           *config.Config
	parserFactory *parsers.ParserFactory
	storage       *StorageService
}

// NewQueryService creates a new query service
func NewQueryService(cfg *config.Config, storage *StorageService) *QueryService {
	binPaths := GetDefaultBinaryPaths()
	return &QueryService{
		cfg:           cfg,
		parserFactory: parsers.NewParserFactory(binPaths),
		storage:       storage,
	}
}

// GetAllServers returns all configured license servers
func (s *QueryService) GetAllServers() ([]models.LicenseServer, error) {
	var servers []models.LicenseServer

	for _, srv := range s.cfg.Servers {
		servers = append(servers, models.LicenseServer{
			Hostname:    srv.Hostname,
			Description: srv.Description,
			Type:        srv.Type,
			CactiID:     srv.CactiID,
			WebUI:       srv.WebUI,
		})
	}

	return servers, nil
}

// QueryServer queries a license server and optionally stores results
func (s *QueryService) QueryServer(hostname, serverType string) (models.ServerQueryResult, error) {
	parser, err := s.parserFactory.GetParser(serverType)
	if err != nil {
		return models.ServerQueryResult{}, err
	}

	log.Infof("Querying %s server: %s", serverType, hostname)
	result := parser.Query(hostname)

	// Log query results at debug level
	if result.Error != nil {
		log.Debugf("Query error for %s: %v", hostname, result.Error)
	} else {
		log.Debugf("Query successful for %s: service=%s, features=%d, users=%d",
			hostname, result.Status.Service, len(result.Features), len(result.Users))
	}

	// Store results in database if storage service is available
	if s.storage != nil && result.Error == nil && len(result.Features) > 0 {
		// Use context with timeout for database operations
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		log.Debugf("Storing %d features from %s to database", len(result.Features), hostname)
		if err := s.storage.StoreFeatures(ctx, result.Features); err != nil {
			log.Errorf("Failed to store features: %v", err)
		} else {
			log.Debugf("Successfully stored features from %s", hostname)
		}

		log.Debugf("Recording usage data for %d features from %s", len(result.Features), hostname)
		if err := s.storage.RecordUsage(ctx, result.Features); err != nil {
			log.Errorf("Failed to record usage: %v", err)
		} else {
			log.Debugf("Successfully recorded usage from %s", hostname)
		}
	}

	return result, result.Error
}
