package repository

import (
	"context"
	"cortex/logging"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	PgErrorCodeUniqueViolation string = "23505"
)

var ErrUniqueViolation = errors.New("unique violation")
var ErrNotFound = errors.New("not found")

type scanConfigAssetJoin struct {
	scanConfigID   string
	scanConfigName string
	assetID        *string
	assetEndpoint  *string
}

type PostgresScanRepository struct {
	logger *slog.Logger
}

func (p PostgresScanRepository) ListScanAssets(ctx context.Context, tx pgx.Tx) ([]ScanAsset, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM assets
	`)
	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return []ScanAsset{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var assets []ScanAsset
	for rows.Next() {
		var asset ScanAsset
		err = rows.Scan(&asset.ID, &asset.Endpoint)
		if err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}

	return assets, nil
}

func (p PostgresScanRepository) GetScanAsset(ctx context.Context, tx pgx.Tx, id string) (*ScanAsset, error) {
	row := tx.QueryRow(ctx, "SELECT * FROM assets WHERE id = $1", id)

	var asset ScanAsset
	err := row.Scan(&asset.ID, &asset.Endpoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &asset, nil
}

func (p PostgresScanRepository) CreateScanAsset(ctx context.Context, tx pgx.Tx, scanAsset ScanAsset) error {
	args := pgx.NamedArgs{
		"id":       scanAsset.ID,
		"endpoint": scanAsset.Endpoint,
	}

	_, err := tx.Exec(ctx, "INSERT INTO assets (id, endpoint) VALUES(@id, @endpoint)", args)

	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
		p.logger.DebugContext(ctx, "asset endpoint already exists", logging.FieldError, err)
		return ErrUniqueViolation
	}

	return nil
}

func (p PostgresScanRepository) UpdateScanAsset(ctx context.Context, tx pgx.Tx, scanAsset ScanAsset) error {
	args := pgx.NamedArgs{
		"id":       scanAsset.ID,
		"endpoint": scanAsset.Endpoint,
	}

	row := tx.QueryRow(ctx, "UPDATE assets SET endpoint = @endpoint WHERE id = @id RETURNING *", args)
	var asset ScanAsset
	err := row.Scan(&asset.ID, &asset.Endpoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
			p.logger.DebugContext(ctx, "asset endpoint already exists", logging.FieldError, err)
			return ErrUniqueViolation
		}
		return err
	}
	return nil
}

func (p PostgresScanRepository) DeleteScanAsset(ctx context.Context, tx pgx.Tx, id string) error {
	args := pgx.NamedArgs{
		"id": id,
	}

	row := tx.QueryRow(ctx, "DELETE FROM assets WHERE id = @id RETURNING *", args)
	var asset ScanAsset
	err := row.Scan(&asset.ID, &asset.Endpoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (p PostgresScanRepository) ListScanConfigurations(ctx context.Context, tx pgx.Tx) ([]ScanConfiguration, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			scan_configs.id,
			scan_configs.name,
			assets.id AS asset_id,
			assets.endpoint AS asset_endpoint
		FROM scan_configs
		FULL OUTER JOIN scan_config_asset_map scam ON scan_configs.id = scam.scan_config_id
		LEFT JOIN public.assets ON scam.asset_id = assets.id;
	`)

	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return []ScanConfiguration{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var configsMap = make(map[string]*ScanConfiguration)

	for rows.Next() {
		var joinRes scanConfigAssetJoin
		err = rows.Scan(&joinRes.scanConfigID, &joinRes.scanConfigName, &joinRes.assetID, &joinRes.assetEndpoint)
		if err != nil {
			return nil, err
		}

		config, ok := configsMap[joinRes.scanConfigID]
		if ok {
			// config already exists
			config.Targets = append(config.Targets, ScanAsset{ID: *joinRes.assetID, Endpoint: *joinRes.assetEndpoint})
			continue
		} else {
			// new config
			var targets = make([]ScanAsset, 0)
			if joinRes.assetID != nil {
				targets = append(targets, ScanAsset{ID: *joinRes.assetID, Endpoint: *joinRes.assetEndpoint})
			}

			configsMap[joinRes.scanConfigID] = &ScanConfiguration{
				ID:      joinRes.scanConfigID,
				Name:    joinRes.scanConfigName,
				Targets: targets,
			}
		}
	}

	configs := make([]ScanConfiguration, 0, len(configsMap))
	for _, config := range configsMap {
		configs = append(configs, *config)
	}

	return configs, nil
}

func (p PostgresScanRepository) GetScanConfiguration(ctx context.Context, tx pgx.Tx, id string) (*ScanConfiguration, error) {
	rows, err := tx.Query(ctx, `
		SELECT
			scan_configs.id,
			scan_configs.name,
			assets.id AS asset_id,
			assets.endpoint AS asset_endpoint
		FROM scan_configs
		FULL OUTER JOIN scan_config_asset_map scam ON scan_configs.id = scam.scan_config_id
		LEFT JOIN public.assets ON scam.asset_id = assets.id
		WHERE scan_configs.id = $1;
	`, id)

	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return nil, ErrNotFound
		}
		return nil, err
	}
	defer rows.Close()

	var config ScanConfiguration
	config.Targets = make([]ScanAsset, 0)
	for rows.Next() {
		var joinRes scanConfigAssetJoin
		err = rows.Scan(&joinRes.scanConfigID, &joinRes.scanConfigName, &joinRes.assetID, &joinRes.assetEndpoint)
		if err != nil {
			return nil, err
		}

		config.ID = joinRes.scanConfigID
		config.Name = joinRes.scanConfigName

		if joinRes.assetID == nil {
			continue
		}
		config.Targets = append(config.Targets, ScanAsset{ID: *joinRes.assetID, Endpoint: *joinRes.assetEndpoint})
	}

	return &config, nil
}

func (p PostgresScanRepository) CreateScanConfiguration(ctx context.Context, tx pgx.Tx, scanConfiguration ScanConfiguration) error {
	// create scan config first, then in the same transaction associate all assets
	args := pgx.NamedArgs{
		"id":   scanConfiguration.ID,
		"name": scanConfiguration.Name,
	}

	_, err := tx.Exec(ctx, "INSERT INTO scan_configs (id, name) VALUES(@id, @name)", args)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
			p.logger.DebugContext(ctx, "scan config name already exists", logging.FieldError, err)
			return ErrUniqueViolation
		}
		return err
	}

	for _, asset := range scanConfiguration.Targets {
		args = pgx.NamedArgs{
			"scan_config_id": scanConfiguration.ID,
			"asset_id":       asset.ID,
		}

		_, err = tx.Exec(ctx, "INSERT INTO scan_config_asset_map (scan_config_id, asset_id) VALUES(@scan_config_id, @asset_id)", args)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateScanConfiguration updates an existing scan configuration in the database with the provided details. Does not update the assets associated with the scan configuration.
func (p PostgresScanRepository) UpdateScanConfiguration(ctx context.Context, tx pgx.Tx, scanConfiguration ScanConfiguration) error {
	args := pgx.NamedArgs{
		"id":   scanConfiguration.ID,
		"name": scanConfiguration.Name,
	}

	row := tx.QueryRow(ctx, "UPDATE scan_configs SET name = @name WHERE id = @id RETURNING *", args)
	var config ScanConfiguration
	err := row.Scan(&config.ID, &config.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
			p.logger.DebugContext(ctx, "scan config name already exists", logging.FieldError, err)
			return ErrUniqueViolation
		}
	}
	return nil
}

func (p PostgresScanRepository) DeleteScanConfiguration(ctx context.Context, tx pgx.Tx, id string) error {
	args := pgx.NamedArgs{
		"id": id,
	}

	row := tx.QueryRow(ctx, "DELETE FROM scan_configs WHERE id = @id RETURNING *", args)
	var config ScanConfiguration
	err := row.Scan(&config.ID, &config.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	// delete all assets associated with this scan config

	args = pgx.NamedArgs{
		"scan_config_id": id,
	}

	row = tx.QueryRow(ctx, "DELETE FROM scan_config_asset_map WHERE scan_config_id = @scan_config_id RETURNING *", args)
	var tmpConfig ScanConfiguration
	err = row.Scan(&tmpConfig.ID, &tmpConfig.Name)
	if err != nil {
		// don't care if there were no rows in the config
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return nil
		}
	}

	return err
}

func (p PostgresScanRepository) RemoveScanConfigurationAssets(ctx context.Context, tx pgx.Tx, scanConfigID string, assetIDs []string) error {
	for _, assetID := range assetIDs {
		args := pgx.NamedArgs{
			"scan_config_id": scanConfigID,
			"asset_id":       assetID,
		}

		_, err := tx.Exec(ctx, "DELETE FROM scan_config_asset_map WHERE scan_config_id = @scan_config_id AND asset_id = @asset_id", args)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return err
		}
	}

	return nil
}

func (p PostgresScanRepository) AddScanConfigurationAssets(ctx context.Context, tx pgx.Tx, scanConfigID string, assetIDs []string) error {
	for _, assetID := range assetIDs {
		args := pgx.NamedArgs{
			"scan_config_id": scanConfigID,
			"asset_id":       assetID,
		}

		_, err := tx.Exec(ctx, "INSERT INTO scan_config_asset_map (scan_config_id, asset_id) VALUES(@scan_config_id, @asset_id)", args)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p PostgresScanRepository) ListScans(ctx context.Context, tx pgx.Tx) ([]ScanExecution, error) {
	rows, err := tx.Query(ctx, `SELECT * FROM scans`)
	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return []ScanExecution{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var scans []ScanExecution
	for rows.Next() {
		var scan ScanExecution
		err = rows.Scan(&scan.ID, &scan.ScanConfigurationID, &scan.StartTime, &scan.EndTime, &scan.Status, &scan.Type)
		if err != nil {
			return nil, err
		}
		scans = append(scans, scan)
	}

	return scans, nil
}

func (p PostgresScanRepository) GetScan(ctx context.Context, tx pgx.Tx, id string) (*ScanExecution, error) {
	row := tx.QueryRow(ctx, "SELECT * FROM scans WHERE id = $1", id)

	var scan ScanExecution
	err := row.Scan(&scan.ID, &scan.ScanConfigurationID, &scan.StartTime, &scan.EndTime, &scan.Status, &scan.Type)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &scan, nil
}

func (p PostgresScanRepository) CreateScan(ctx context.Context, tx pgx.Tx, scanRun ScanExecution) error {
	args := pgx.NamedArgs{
		"id":              scanRun.ID,
		"type":            scanRun.Type,
		"scan_config_id":  scanRun.ScanConfigurationID,
		"scan_start_time": scanRun.StartTime,
		"scan_end_time":   scanRun.EndTime,
		"status":          scanRun.Status,
	}

	_, err := tx.Exec(ctx, "INSERT INTO scans (id, scan_config_id, scan_start_time, scan_end_time, status, type) VALUES(@id, @scan_config_id, @scan_start_time, @scan_end_time, @status, @type)", args)
	return err
}

func (p PostgresScanRepository) UpdateScan(ctx context.Context, tx pgx.Tx, scanRun ScanExecution) error {
	args := pgx.NamedArgs{
		"id":              scanRun.ID,
		"scan_config_id":  scanRun.ScanConfigurationID,
		"scan_start_time": scanRun.StartTime,
		"scan_end_time":   scanRun.EndTime,
		"status":          scanRun.Status,
	}

	row := tx.QueryRow(ctx, "UPDATE scans SET scan_config_id = @scan_config_id, scan_start_time = @scan_start_time, scan_end_time = @scan_end_time, status = @status WHERE id = @id RETURNING *", args)
	var scan ScanExecution
	err := row.Scan(&scan.ID, &scan.ScanConfigurationID, &scan.StartTime, &scan.EndTime, &scan.Status, &scan.Type)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (p PostgresScanRepository) PutAssetDiscoveryResult(ctx context.Context, tx pgx.Tx, result ScanAssetDiscoveryResult) error {
	// check if already exists
	row := tx.QueryRow(ctx, "SELECT COUNT(*) FROM asset_discovery WHERE asset_id = $1 AND port = $2 AND protocol = $3",
		result.AssetID, result.Port, result.Protocol)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return err
	}

	args := pgx.NamedArgs{
		"asset_id":   result.AssetID,
		"port":       result.Port,
		"protocol":   result.Protocol,
		"first_seen": result.FirstSeen,
		"last_seen":  result.LastSeen,
	}

	if count > 0 {
		// update
		_, err = tx.Exec(ctx, `UPDATE asset_discovery SET last_seen = @last_seen WHERE asset_id = @asset_id AND port = @port AND protocol = @protocol`, args)
		if err != nil {
			return err
		}
	} else {
		// insert
		_, err = tx.Exec(ctx, `INSERT INTO asset_discovery (asset_id, port, protocol, first_seen, last_seen)
								    VALUES(@asset_id, @port, @protocol, @first_seen, @last_seen)`, args)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p PostgresScanRepository) ListAssetDiscoveryResults(ctx context.Context, tx pgx.Tx, assetID string) ([]ScanAssetDiscoveryResult, error) {
	rows, err := tx.Query(ctx, `SELECT * FROM asset_discovery WHERE asset_id = $1`, assetID)
	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return []ScanAssetDiscoveryResult{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var discoveryResults []ScanAssetDiscoveryResult
	for rows.Next() {
		var discoveryResult ScanAssetDiscoveryResult
		err = rows.Scan(&discoveryResult.AssetID, &discoveryResult.Port,
			&discoveryResult.Protocol, &discoveryResult.FirstSeen, &discoveryResult.LastSeen)
		if err != nil {
			return nil, err
		}
		discoveryResults = append(discoveryResults, discoveryResult)
	}

	return discoveryResults, nil
}

func (p PostgresScanRepository) GetAssetStats(ctx context.Context, tx pgx.Tx, assetID string) (*ScanAssetStats, error) {
	// get number of discovered ports
	row := tx.QueryRow(ctx, "SELECT COUNT(*) FROM asset_discovery WHERE asset_id = $1", assetID)
	var portCount int
	err := row.Scan(&portCount)
	if err != nil {
		return nil, err
	}

	// find timestamp of last discovery scan
	row = tx.QueryRow(ctx, `
		SELECT s.scan_end_time
		FROM scans s
				 INNER JOIN scan_config_asset_map scam ON s.scan_config_id = scam.scan_config_id
		WHERE scam.asset_id = $1
		  AND s.type = 'discovery'
		ORDER BY s.scan_start_time DESC
		LIMIT 1;
    `, assetID)
	var lastDiscoveryTime time.Time
	err = row.Scan(&lastDiscoveryTime)

	stats := ScanAssetStats{
		DiscoveredPortsCount: portCount,
		LastDiscovery:        lastDiscoveryTime,
	}
	return &stats, nil
}

func NewPostgresScanRepository() *PostgresScanRepository {
	return &PostgresScanRepository{
		logger: logging.GetLogger(logging.DataAccess),
	}
}
