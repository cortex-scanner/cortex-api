package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ScanAsset defines a target endpoint for a scan
type ScanAsset struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint"`
}

type ScanAssetStats struct {
	DiscoveredPortsCount int       `json:"discoveredPortsCount"`
	LastDiscovery        time.Time `json:"lastDiscovery"`
}

func (s ScanAssetStats) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		DiscoveredPortsCount int   `json:"discoveredPortsCount"`
		LastDiscovery        int64 `json:"lastDiscovery"`
	}{
		DiscoveredPortsCount: s.DiscoveredPortsCount,
		LastDiscovery:        s.LastDiscovery.Unix(),
	})
}

type ScanAssetWithStats struct {
	ID       string         `json:"id"`
	Endpoint string         `json:"endpoint"`
	Stats    ScanAssetStats `json:"stats"`
}

type ScanAssetEventType string

const (
	ScanAssetEventTypeCreated   ScanAssetEventType = "created"
	ScanAssetEventTypeUpdated   ScanAssetEventType = "updated"
	ScanAssetEventTypeScanEnded ScanAssetEventType = "scan_finished"
)

type AssetHistoryEntry struct {
	ID      string             `json:"id"`
	AssetID string             `json:"assetId"`
	UserID  string             `json:"userId"`
	Time    time.Time          `json:"timestamp"`
	Type    ScanAssetEventType `json:"eventType"`
	Data    map[string]any     `json:"eventData"`
}

func (a AssetHistoryEntry) MarshalJSON() ([]byte, error) {
	data := struct {
		ID      string             `json:"id"`
		AssetID string             `json:"assetId"`
		UserID  string             `json:"userId"`
		Time    int64              `json:"timestamp"`
		Type    ScanAssetEventType `json:"eventType"`
		Data    map[string]any     `json:"eventData"`
	}{
		ID:      a.ID,
		AssetID: a.AssetID,
		UserID:  a.UserID,
		Time:    a.Time.Unix(),
		Type:    a.Type,
		Data:    a.Data,
	}

	return json.Marshal(data)
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

func (d ScanAssetDiscoveryResult) MarshalJSON() ([]byte, error) {
	// marshal with time.Time to unix
	data := struct {
		AssetID   string       `json:"assetId"`
		Port      int          `json:"port"`
		Protocol  ScanProtocol `json:"protocol"`
		FirstSeen int64        `json:"firstSeen"`
		LastSeen  int64        `json:"lastSeen"`
	}{
		AssetID:   d.AssetID,
		Port:      d.Port,
		Protocol:  d.Protocol,
		FirstSeen: d.FirstSeen.Unix(),
		LastSeen:  d.LastSeen.Unix(),
	}

	return json.Marshal(data)
}

// ScanConfiguration defines a scan configuration applied to a scan
type ScanConfiguration struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Type   ScanType `json:"type"`
	Engine string   `json:"engine"`
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
	ID                  string           `json:"id"`
	ScanConfigurationID string           `json:"scanConfigurationId"`
	Status              ScanStatus       `json:"status"`
	StartTime           pgtype.Timestamp `json:"startTime"`
	EndTime             pgtype.Timestamp `json:"endTime"`
	Assets              []ScanAsset      `json:"assets"`
}

func (s ScanExecution) MarshalJSON() ([]byte, error) {
	startTime := int64(0)
	if s.StartTime.Valid {
		startTime = s.StartTime.Time.Unix()
	}

	endTime := int64(0)
	if s.EndTime.Valid {
		endTime = s.EndTime.Time.Unix()
	}

	data := struct {
		ID                  string      `json:"id"`
		ScanConfigurationID string      `json:"scanConfigurationId"`
		Type                ScanType    `json:"type"`
		Status              ScanStatus  `json:"status"`
		StartTime           int64       `json:"startTime"`
		EndTime             int64       `json:"endTime"`
		Assets              []ScanAsset `json:"assets"`
	}{
		ID:                  s.ID,
		ScanConfigurationID: s.ScanConfigurationID,
		Status:              s.Status,
		StartTime:           startTime,
		EndTime:             endTime,
		Assets:              s.Assets,
	}

	return json.Marshal(data)
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

	GetAssetStats(ctx context.Context, tx pgx.Tx, assetID string) (*ScanAssetStats, error)

	GetAssetHistory(ctx context.Context, tx pgx.Tx, assetID string) ([]AssetHistoryEntry, error)
	AddAssetHistoryEntry(ctx context.Context, tx pgx.Tx, entry AssetHistoryEntry) error
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
