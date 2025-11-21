package service

import (
	"context"
	cortexContext "cortex/context"
	"cortex/logging"
	"cortex/repository"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScanUpdateOptions struct {
	StartTime time.Time
	EndTime   time.Time
	Status    string
}

type ScanService interface {
	ListScanConfigs(ctx context.Context) ([]repository.ScanConfiguration, error)
	GetScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error)
	CreateScanConfig(ctx context.Context, name string) (*repository.ScanConfiguration, error)
	UpdateScanConfig(ctx context.Context, id string, newName string) (*repository.ScanConfiguration, error)
	DeleteScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error)

	ListAssets(ctx context.Context) ([]repository.ScanAsset, error)
	ListAssetsWithStats(ctx context.Context) ([]repository.ScanAssetWithStats, error)
	GetAsset(ctx context.Context, id string) (*repository.ScanAsset, error)
	GetAssetWithStats(ctx context.Context, id string) (*repository.ScanAssetWithStats, error)
	CreateAsset(ctx context.Context, endpoint string) (*repository.ScanAsset, error)
	DeleteAsset(ctx context.Context, id string) (*repository.ScanAsset, error)
	UpdateAsset(ctx context.Context, id string, newEndpoint string) (*repository.ScanAsset, error)

	ListAssetFindings(ctx context.Context, assetID string) ([]repository.AssetFinding, error)
	ListAssetHistory(ctx context.Context, assetID string) ([]repository.AssetHistoryEntry, error)

	RunScan(ctx context.Context, configID string, assetIds []string) (*repository.ScanExecution, error)
	ListScans(ctx context.Context) ([]repository.ScanExecution, error)
	GetScan(ctx context.Context, id string) (*repository.ScanExecution, error)
	UpdateScan(ctx context.Context, scanID string, update ScanUpdateOptions) (*repository.ScanExecution, error)
}

type scanService struct {
	repo   repository.ScanRepository
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func (s scanService) ListScanConfigs(ctx context.Context) ([]repository.ScanConfiguration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	configs, err := s.repo.ListScanConfigurations(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list scan configurations", logging.FieldError, err)
		return nil, err
	}
	return configs, nil
}

func (s scanService) GetScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	config, err := s.repo.GetScanConfiguration(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration",
			logging.FieldScanConfigID, id,
			logging.FieldError, err)
		return nil, err
	}
	return config, nil
}

func (s scanService) CreateScanConfig(ctx context.Context, name string) (*repository.ScanConfiguration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	config := repository.ScanConfiguration{
		ID:   uuid.New().String(),
		Name: name,
	}

	err = s.repo.CreateScanConfiguration(ctx, tx, config)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create scan configuration", logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan configuration created", logging.FieldScanConfigID, config.ID)

	return &config, nil
}

func (s scanService) UpdateScanConfig(ctx context.Context, id string, newName string) (*repository.ScanConfiguration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	config, err := s.repo.GetScanConfiguration(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration for update",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	config.Name = newName
	err = s.repo.UpdateScanConfiguration(ctx, tx, *config)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update scan configuration",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan configuration updated", logging.FieldScanConfigID, id)

	return config, nil
}

func (s scanService) DeleteScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	config, err := s.repo.GetScanConfiguration(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration for deletion",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	err = s.repo.DeleteScanConfiguration(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete scan configuration",
			logging.FieldScanConfigID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan configuration deleted", logging.FieldScanConfigID, id)

	return config, nil
}

func (s scanService) listAssets(ctx context.Context, tx pgx.Tx) ([]repository.ScanAsset, error) {
	assets, err := s.repo.ListScanAssets(ctx, tx)
	if err != nil {
		return nil, err
	}
	return assets, nil
}

func (s scanService) ListAssets(ctx context.Context) ([]repository.ScanAsset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	assets, err := s.listAssets(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list scan assets", logging.FieldError, err)
		return nil, err
	}

	return assets, nil
}

func (s scanService) ListAssetsWithStats(ctx context.Context) ([]repository.ScanAssetWithStats, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	assets, err := s.listAssets(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list scan assets", logging.FieldError, err)
		return nil, err
	}

	// augment asset with stats
	var assetsWithStats []repository.ScanAssetWithStats
	for _, a := range assets {
		assetStats, err := s.repo.GetAssetStats(ctx, tx, a.ID)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get asset stats", logging.FieldError, err)
			return nil, err
		}

		stat := repository.ScanAssetWithStats{
			ID:       a.ID,
			Endpoint: a.Endpoint,
			Stats:    *assetStats,
		}

		assetsWithStats = append(assetsWithStats, stat)
	}

	return assetsWithStats, nil
}

func (s scanService) GetAsset(ctx context.Context, id string) (*repository.ScanAsset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	asset, err := s.repo.GetScanAsset(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}
	return asset, nil
}

func (s scanService) GetAssetWithStats(ctx context.Context, id string) (*repository.ScanAssetWithStats, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	asset, err := s.repo.GetScanAsset(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	assetStats, err := s.repo.GetAssetStats(ctx, tx, asset.ID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get asset stats", logging.FieldError, err)
		return nil, err
	}

	return &repository.ScanAssetWithStats{
		ID:       asset.ID,
		Endpoint: asset.Endpoint,
		Stats:    *assetStats,
	}, nil
}

func (s scanService) CreateAsset(ctx context.Context, endpoint string) (*repository.ScanAsset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	asset := repository.ScanAsset{
		ID:       uuid.New().String(),
		Endpoint: endpoint,
	}

	err = s.repo.CreateScanAsset(ctx, tx, asset)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create scan asset",
			logging.FieldError, err)
		return nil, err
	}

	// create event
	userInfo, err := cortexContext.UserInfo(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user info from context", logging.FieldError, err)
		return nil, err
	}

	event := repository.AssetHistoryEntry{
		ID:      uuid.New().String(),
		AssetID: asset.ID,
		UserID:  userInfo.UserID,
		Time:    time.Now(),
		Type:    repository.ScanAssetEventTypeCreated,
		Data:    nil,
	}
	err = s.repo.AddAssetHistoryEntry(ctx, tx, event)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add asset history entry", logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan asset created", logging.FieldAssetID, asset.ID)

	return &asset, nil
}

func (s scanService) DeleteAsset(ctx context.Context, id string) (*repository.ScanAsset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	asset, err := s.repo.GetScanAsset(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan asset for deletion",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	err = s.repo.DeleteScanAsset(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan asset deleted", logging.FieldAssetID, id)

	return asset, nil
}

func (s scanService) UpdateAsset(ctx context.Context, id string, newEndpoint string) (*repository.ScanAsset, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	asset, err := s.repo.GetScanAsset(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to get scan asset for update",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	asset.Endpoint = newEndpoint
	err = s.repo.UpdateScanAsset(ctx, tx, *asset)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to update scan asset",
			logging.FieldAssetID, id, logging.FieldError, err)
		return nil, err
	}

	// create event
	userInfo, err := cortexContext.UserInfo(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user info from context", logging.FieldError, err)
		return nil, err
	}

	event := repository.AssetHistoryEntry{
		ID:      uuid.New().String(),
		AssetID: asset.ID,
		UserID:  userInfo.UserID,
		Time:    time.Now(),
		Type:    repository.ScanAssetEventTypeUpdated,
		// TODO: get changed attributes
		Data: nil,
	}

	err = s.repo.AddAssetHistoryEntry(ctx, tx, event)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to add asset history entry", logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "scan asset updated", logging.FieldAssetID, id)

	return asset, nil
}

func (s scanService) RunScan(ctx context.Context, configID string, assetIds []string) (*repository.ScanExecution, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	// check if scan config exists
	config, err := s.repo.GetScanConfiguration(ctx, tx, configID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan configuration",
			logging.FieldError, err)
		return nil, err
	}

	now := time.Now()
	scan := repository.ScanExecution{
		ID:                  uuid.New().String(),
		ScanConfigurationID: config.ID,
		Status:              repository.ScanStatusQueued,
		StartTime:           pgtype.Timestamp{Time: now},
	}

	// add assets to scan
	for _, assetId := range assetIds {
		// check if the asset exists
		asset, err := s.repo.GetScanAsset(ctx, tx, assetId)
		if err != nil {
			s.logger.ErrorContext(ctx, "failed to get scan asset",
				logging.FieldAssetID, assetId, logging.FieldError, err)
			return nil, err
		}

		scan.Assets = append(scan.Assets, *asset)
	}

	err = s.repo.CreateScan(ctx, tx, scan)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create scan",
			logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "queued scan execution",
		logging.FieldScanConfigID, config.ID, logging.FieldScanID, scan.ID)

	// commit before running scan so scanner can access the scan
	err = tx.Commit(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to commit transaction when creating scan", logging.FieldError, err)
		return nil, err
	}

	return &scan, nil
}

func (s scanService) ListScans(ctx context.Context) ([]repository.ScanExecution, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	scans, err := s.repo.ListScans(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list scans", logging.FieldError, err)
		return nil, err
	}
	return scans, nil
}

func (s scanService) GetScan(ctx context.Context, id string) (*repository.ScanExecution, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	scan, err := s.repo.GetScan(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan", logging.FieldError, err)
		return nil, err
	}
	return scan, nil
}

func (s scanService) UpdateScan(ctx context.Context, scanID string, update ScanUpdateOptions) (*repository.ScanExecution, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	// check if scan exists
	scan, err := s.repo.GetScan(ctx, tx, scanID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get scan",
			logging.FieldError, err)
		return nil, err
	}

	// apply updates
	if !update.StartTime.Before(time.Date(1970, 1, 1, 2, 0, 0, 0, time.UTC)) {
		scan.StartTime.Time = update.StartTime
	}
	if !update.EndTime.Before(time.Date(1970, 1, 1, 2, 0, 0, 0, time.UTC)) {
		scan.EndTime.Time = update.EndTime
	}
	if update.Status != "" {
		scan.Status = repository.ScanStatus(update.Status)
	}

	err = s.repo.UpdateScan(ctx, tx, *scan)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update scan",
			logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, "updated scan", logging.FieldScanID, scan.ID)

	return scan, nil
}

func (s scanService) ListAssetFindings(ctx context.Context, assetID string) ([]repository.AssetFinding, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	results, err := s.repo.ListAssetFindings(ctx, tx, assetID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list asset discovery results",
			logging.FieldAssetID, assetID, logging.FieldError, err)
		return nil, err
	}
	return results, nil
}

func (s scanService) ListAssetHistory(ctx context.Context, assetID string) ([]repository.AssetHistoryEntry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	history, err := s.repo.GetAssetHistory(ctx, tx, assetID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get asset history", logging.FieldAssetID, assetID, logging.FieldError, err)
		return nil, err
	}

	return history, nil
}

func NewScanService(scanRepo repository.ScanRepository, pool *pgxpool.Pool) ScanService {
	return scanService{
		repo:   scanRepo,
		logger: logging.GetLogger(logging.DataAccess),
		pool:   pool,
	}
}
