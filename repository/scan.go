package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// ScanAsset defines a target endpoint for a scan
type ScanAsset struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

type ScanProtocol string

const (
	ScanProtocolTCP ScanProtocol = "tcp"
	ScanProtocolUDP ScanProtocol = "udp"
)

// ScanAssetDiscoveryResult represents the result of discovering an asset during a scan.
// It includes information about the asset, port, protocol, and discovery timestamps.
type ScanAssetDiscoveryResult struct {
	AssetID   string       `json:"assetId"`
	Port      int          `json:"port"`
	Protocol  ScanProtocol `json:"protocol"`
	FirstSeen time.Time    `json:"firstSeen"`
	LastSeen  time.Time    `json:"lastSeen"`
}

// ScanConfiguration defines a scan configuration applied to a scan
type ScanConfiguration struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Targets []ScanAsset `json:"targets"`
}

type ScanStatus string

const (
	ScanStatusQueued    ScanStatus = "queued"
	ScanStatusRunning   ScanStatus = "running"
	ScanStatusComplete  ScanStatus = "complete"
	ScanStatusFailed    ScanStatus = "failed"
	ScanStatusCancelled ScanStatus = "cancelled"
)

type ScanType string

const (
	ScanTypeDiscovery     ScanType = "discovery"
	ScanTypeVulnerability ScanType = "vuln"
	ScanTypeCombined      ScanType = "discovery+vuln"
)

// ScanExecution represents metadata and status details for a single scan execution.
type ScanExecution struct {
	ID                  string     `json:"id"`
	ScanConfigurationID string     `json:"scanConfigurationId"`
	Type                ScanType   `json:"type"`
	Status              ScanStatus `json:"status"`
	StartTime           *time.Time `json:"startTime"`
	EndTime             *time.Time `json:"endTime"`
}

// ScanAssetRepository defines an interface for managing and interacting with scan asset data in a repository.
type ScanAssetRepository interface {
	// ListScanAssets retrieves all scan assets from the repository.
	ListScanAssets(ctx context.Context, tx pgx.Tx) ([]ScanAsset, error)
	// GetScanAsset fetches a specific scan asset given its unique identifier.
	GetScanAsset(ctx context.Context, tx pgx.Tx, id string) (*ScanAsset, error)
	// CreateScanAsset adds a new scan asset to the repository.
	CreateScanAsset(ctx context.Context, tx pgx.Tx, scanAsset ScanAsset) error
	// UpdateScanAsset modifies an existing scan asset in the repository.
	UpdateScanAsset(ctx context.Context, tx pgx.Tx, scanAsset ScanAsset) error
	// DeleteScanAsset removes a scan asset from the repository using its unique identifier.
	DeleteScanAsset(ctx context.Context, tx pgx.Tx, id string) error

	PutAssetDiscoveryResult(ctx context.Context, tx pgx.Tx, result ScanAssetDiscoveryResult) error
	ListAssetDiscoveryResults(ctx context.Context, tx pgx.Tx, assetID string) ([]ScanAssetDiscoveryResult, error)
}

// ScanConfigurationRepository defines methods to manage scan configurations in a repository.
type ScanConfigurationRepository interface {
	// ListScanConfigurations retrieves all scan configurations.
	ListScanConfigurations(ctx context.Context, tx pgx.Tx) ([]ScanConfiguration, error)
	// GetScanConfiguration fetches a scan configuration by its unique identifier.
	GetScanConfiguration(ctx context.Context, tx pgx.Tx, id string) (*ScanConfiguration, error)
	// CreateScanConfiguration adds a new scan configuration to the repository.
	CreateScanConfiguration(ctx context.Context, tx pgx.Tx, scanConfiguration ScanConfiguration) error
	// UpdateScanConfiguration updates an existing scan configuration. Does not update the assets associated with the scan configuration.
	UpdateScanConfiguration(ctx context.Context, tx pgx.Tx, scanConfiguration ScanConfiguration) error
	// DeleteScanConfiguration removes a scan configuration using its unique identifier.
	DeleteScanConfiguration(ctx context.Context, tx pgx.Tx, id string) error
	// RemoveScanConfigurationAssets removes specified assets from a scan configuration identified by its unique ID.
	RemoveScanConfigurationAssets(ctx context.Context, tx pgx.Tx, scanConfigID string, assetIDs []string) error
	// AddScanConfigurationAssets associates a list of asset IDs with the specified scan configuration in the repository.
	AddScanConfigurationAssets(ctx context.Context, tx pgx.Tx, scanConfigID string, assetIDs []string) error
}

// ScanExecutionRepository defines methods for managing scan executions and their metadata in a repository.
type ScanExecutionRepository interface {
	// ListScans retrieves all scan executions from the repository.
	ListScans(ctx context.Context, tx pgx.Tx) ([]ScanExecution, error)
	// GetScan fetches a specific scan execution given its unique identifier.
	GetScan(ctx context.Context, tx pgx.Tx, id string) (*ScanExecution, error)
	// CreateScan adds a new scan execution to the repository.
	CreateScan(ctx context.Context, tx pgx.Tx, scanRun ScanExecution) error
	// UpdateScan modifies an existing scan execution in the repository.
	UpdateScan(ctx context.Context, tx pgx.Tx, scanRun ScanExecution) error
}

// ScanRepository combines functionality for managing scan asset data and scan configurations in a repository.
type ScanRepository interface {
	ScanAssetRepository
	ScanConfigurationRepository
	ScanExecutionRepository
}
