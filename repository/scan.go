package repository

import "context"

// ScanAsset defines a target endpoint for a scan
type ScanAsset struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

// ScanConfiguration defines a scan configuration applied to a scan
type ScanConfiguration struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Targets []ScanAsset `json:"targets"`
}

// ScanAssetRepository defines an interface for managing and interacting with scan asset data in a repository.
type ScanAssetRepository interface {
	// ListScanAssets retrieves all scan assets from the repository.
	ListScanAssets(ctx context.Context) ([]ScanAsset, error)
	// GetScanAsset fetches a specific scan asset given its unique identifier.
	GetScanAsset(ctx context.Context, id string) (*ScanAsset, error)
	// CreateScanAsset adds a new scan asset to the repository.
	CreateScanAsset(ctx context.Context, scanAsset ScanAsset) error
	// UpdateScanAsset modifies an existing scan asset in the repository.
	UpdateScanAsset(ctx context.Context, scanAsset ScanAsset) error
	// DeleteScanAsset removes a scan asset from the repository using its unique identifier.
	DeleteScanAsset(ctx context.Context, id string) error
}

// TODO: Add support for updating scan configuration assets

// ScanConfigurationRepository defines methods to manage scan configurations in a repository.
type ScanConfigurationRepository interface {
	// ListScanConfigurations retrieves all scan configurations.
	ListScanConfigurations(ctx context.Context) ([]ScanConfiguration, error)
	// GetScanConfiguration fetches a scan configuration by its unique identifier.
	GetScanConfiguration(ctx context.Context, id string) (*ScanConfiguration, error)
	// CreateScanConfiguration adds a new scan configuration to the repository.
	CreateScanConfiguration(ctx context.Context, scanConfiguration ScanConfiguration) error
	// UpdateScanConfiguration updates an existing scan configuration. Does not update the assets associated with the scan configuration.
	UpdateScanConfiguration(ctx context.Context, scanConfiguration ScanConfiguration) error
	// DeleteScanConfiguration removes a scan configuration using its unique identifier.
	DeleteScanConfiguration(ctx context.Context, id string) error
}

// ScanRepository combines functionality for managing scan asset data and scan configurations in a repository.
type ScanRepository interface {
	ScanAssetRepository
	ScanConfigurationRepository
}
