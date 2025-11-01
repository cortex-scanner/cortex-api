package service

import (
	"context"
	"cortex/logging"
	"cortex/repository"
	"log/slog"
)

type ScanService interface {
	ListScanConfigs(ctx context.Context) ([]repository.ScanConfiguration, error)
	GetScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error)
	CreateScanConfig(ctx context.Context, name string) error
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
	//TODO implement me
	panic("implement me")
}

func (s scanService) GetScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) CreateScanConfig(ctx context.Context, name string) error {
	//TODO implement me
	panic("implement me")
}

func (s scanService) UpdateScanConfig(ctx context.Context, id string, newName string) (*repository.ScanConfiguration, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) UpdateScanConfigAssets(ctx context.Context, id string, assetIDs []string) (*repository.ScanConfiguration, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) DeleteScanConfig(ctx context.Context, id string) (*repository.ScanConfiguration, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) ListAssets(ctx context.Context) ([]repository.ScanAsset, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) GetAsset(ctx context.Context, id string) (*repository.ScanAsset, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) CreateAsset(ctx context.Context, endpoint string) (*repository.ScanAsset, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) DeleteAsset(ctx context.Context, id string) (*repository.ScanAsset, error) {
	//TODO implement me
	panic("implement me")
}

func (s scanService) UpdateAsset(ctx context.Context, id string, newEndpoint string) (*repository.ScanAsset, error) {
	//TODO implement me
	panic("implement me")
}

func NewScanService(scanRepo repository.ScanRepository) ScanService {
	return scanService{
		repo:   scanRepo,
		logger: logging.GetLogger(logging.DataAccess),
	}
}
