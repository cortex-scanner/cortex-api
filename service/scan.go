package service

import (
	"context"
	"cortex/logging"
	"cortex/repository"
	"fmt"
	"log/slog"
	"slices"

	"github.com/google/uuid"
)

type ScanService interface {
	ListScanConfigs(ctx context.Context) ([]repository.ScanConfiguration, error)
	GetScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error)
	CreateScanConfig(ctx context.Context, name string) (*repository.ScanConfiguration, error)
	UpdateScanConfig(ctx context.Context, id string, newName string) (*repository.ScanConfiguration, error)
	UpdateScanConfigAssets(ctx context.Context, id string, assetIDs []string) (*repository.ScanConfiguration, error)
	DeleteScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error)

	ListAssets(ctx context.Context) ([]repository.ScanAsset, error)
	GetAsset(ctx context.Context, id string) (*repository.ScanAsset, error)
	CreateAsset(ctx context.Context, endpoint string) (*repository.ScanAsset, error)
	DeleteAsset(ctx context.Context, id string) (*repository.ScanAsset, error)
	UpdateAsset(ctx context.Context, id string, newEndpoint string) (*repository.ScanAsset, error)
}

type scanService struct {
	repo   repository.ScanRepository
	logger *slog.Logger
}

func (s scanService) ListScanConfigs(ctx context.Context) ([]repository.ScanConfiguration, error) {
	configs, err := s.repo.ListScanConfigurations(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list scan configurations", logging.FieldError, err)
		return nil, err
	}
	return configs, nil
}

func (s scanService) GetScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error) {
	config, err := s.repo.GetScanConfiguration(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration",
			logging.FieldScanConfigID, id,
			logging.FieldError, err)
		return nil, err
	}
	return config, nil
}

func (s scanService) CreateScanConfig(ctx context.Context, name string) (*repository.ScanConfiguration, error) {
	config := repository.ScanConfiguration{
		ID:      uuid.New().String(),
		Name:    name,
		Targets: make([]repository.ScanAsset, 0),
	}

	err := s.repo.CreateScanConfiguration(ctx, config)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create scan configuration", logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan configuration created", logging.FieldScanConfigID, config.ID)

	return &config, nil
}

func (s scanService) UpdateScanConfig(ctx context.Context, id string, newName string) (*repository.ScanConfiguration, error) {
	config, err := s.repo.GetScanConfiguration(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration for update",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	config.Name = newName
	err = s.repo.UpdateScanConfiguration(ctx, *config)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update scan configuration",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan configuration updated", logging.FieldScanConfigID, id)

	return config, nil
}

func (s scanService) UpdateScanConfigAssets(ctx context.Context, id string, assetIDs []string) (*repository.ScanConfiguration, error) {
	config, err := s.repo.GetScanConfiguration(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration for asset update",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	currentAssetIDs := make([]string, 0)
	for _, asset := range config.Targets {
		currentAssetIDs = append(currentAssetIDs, asset.ID)
	}

	// find new assets
	newAssets := make([]string, 0)
	for _, assetID := range assetIDs {
		if _, err := s.repo.GetScanAsset(ctx, assetID); err == nil {
			if !slices.Contains(currentAssetIDs, assetID) {
				newAssets = append(newAssets, assetID)
			}
		} else {
			return nil, repository.ErrNotFound
		}
	}

	// find removed assets
	removedAssets := make([]string, 0)
	for _, assetID := range currentAssetIDs {
		if !slices.Contains(assetIDs, assetID) {
			removedAssets = append(removedAssets, assetID)
		}
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("adding %d assets, removing %d assets", len(newAssets), len(removedAssets)),
		logging.FieldScanConfigID, id)

	err = s.repo.AddScanConfigurationAssets(ctx, id, newAssets)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add assets to scan configuration",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}
	err = s.repo.RemoveScanConfigurationAssets(ctx, id, removedAssets)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to remove assets from scan configuration",
			logging.FieldScanConfigID, id, logging.FieldError, err)
	}

	s.logger.InfoContext(ctx, "scan configuration assets updated", logging.FieldScanConfigID, id)

	// get config again to get updated asset list
	config, err = s.repo.GetScanConfiguration(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration after asset update",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	return config, nil
}

func (s scanService) DeleteScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error) {
	config, err := s.repo.GetScanConfiguration(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration for deletion",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	err = s.repo.DeleteScanConfiguration(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete scan configuration",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan configuration deleted", logging.FieldScanConfigID, id)

	return config, nil
}

func (s scanService) ListAssets(ctx context.Context) ([]repository.ScanAsset, error) {
	assets, err := s.repo.ListScanAssets(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list scan assets", logging.FieldError, err)
		return nil, err
	}
	return assets, nil
}

func (s scanService) GetAsset(ctx context.Context, id string) (*repository.ScanAsset, error) {
	asset, err := s.repo.GetScanAsset(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}
	return asset, nil
}

func (s scanService) CreateAsset(ctx context.Context, endpoint string) (*repository.ScanAsset, error) {
	asset := repository.ScanAsset{
		ID:       uuid.New().String(),
		Endpoint: endpoint,
	}

	err := s.repo.CreateScanAsset(ctx, asset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create scan asset",
			logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan asset created", logging.FieldAssetID, asset.ID)

	return &asset, nil
}

func (s scanService) DeleteAsset(ctx context.Context, id string) (*repository.ScanAsset, error) {
	asset, err := s.repo.GetScanAsset(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan asset for deletion",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	err = s.repo.DeleteScanAsset(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan asset deleted", logging.FieldAssetID, id)

	return asset, nil
}

func (s scanService) UpdateAsset(ctx context.Context, id string, newEndpoint string) (*repository.ScanAsset, error) {
	asset, err := s.repo.GetScanAsset(ctx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get scan asset for update",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	asset.Endpoint = newEndpoint
	err = s.repo.UpdateScanAsset(ctx, *asset)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan asset updated", logging.FieldAssetID, id)

	return asset, nil
}

func NewScanService(scanRepo repository.ScanRepository) ScanService {
	return scanService{
		repo:   scanRepo,
		logger: logging.GetLogger(logging.DataAccess),
	}
}
