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

type PostgresScanRepository struct {
	logger *slog.Logger
}

func (p PostgresScanRepository) ListScanAssets(ctx context.Context, tx pgx.Tx) ([]ScanAsset, error) {
	rows, err := tx.Query(ctx, `
		SELECT * 
		FROM assets
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
	row := tx.QueryRow(ctx, `
		SELECT * 
		FROM assets 
		WHERE id = $1`, id)

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

	_, err := tx.Exec(ctx, `
		INSERT INTO assets (id, endpoint) 
		VALUES(@id, @endpoint)`, args)

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

	row := tx.QueryRow(ctx, `
		UPDATE assets 
		SET endpoint = @endpoint 
		WHERE id = @id 
		RETURNING *`, args)

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

	row := tx.QueryRow(ctx, `
		DELETE FROM assets 
		WHERE id = @id 
		RETURNING *`, args)

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
		SELECT * 
		FROM scan_configs;
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

	var scans []ScanConfiguration
	for rows.Next() {
		var scan ScanConfiguration
		err = rows.Scan(&scan.ID, &scan.Name, &scan.Type, &scan.Engine)
		if err != nil {
			return nil, err
		}
		scans = append(scans, scan)
	}

	return scans, nil
}

func (p PostgresScanRepository) GetScanConfiguration(ctx context.Context, tx pgx.Tx, id string) (*ScanConfiguration, error) {
	row := tx.QueryRow(ctx, `
		SELECT * 
		FROM scan_configs 
		WHERE scan_configs.id = $1;
	`, id)

	var scan ScanConfiguration
	err := row.Scan(&scan.ID, &scan.Name, &scan.Type, &scan.Engine)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return &scan, nil
}

func (p PostgresScanRepository) CreateScanConfiguration(ctx context.Context, tx pgx.Tx, scanConfiguration ScanConfiguration) error {
	// create scan config first, then in the same transaction associate all assets
	args := pgx.NamedArgs{
		"id":     scanConfiguration.ID,
		"name":   scanConfiguration.Name,
		"type":   scanConfiguration.Type,
		"engine": scanConfiguration.Engine,
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO scan_configs (id, name, type, engine) 
		VALUES(@id, @name, @type, @engine)`, args)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
			p.logger.DebugContext(ctx, "scan config name already exists", logging.FieldError, err)
			return ErrUniqueViolation
		}
		return err
	}

	return nil
}

// UpdateScanConfiguration updates an existing scan configuration in the database with the provided details.
func (p PostgresScanRepository) UpdateScanConfiguration(ctx context.Context, tx pgx.Tx, scanConfiguration ScanConfiguration) error {
	args := pgx.NamedArgs{
		"id":     scanConfiguration.ID,
		"name":   scanConfiguration.Name,
		"type":   scanConfiguration.Type,
		"engine": scanConfiguration.Engine,
	}

	row := tx.QueryRow(ctx, `
		UPDATE scan_configs 
		SET name = @name, type = @type, engine = @engine 
		WHERE id = @id 
		RETURNING *`, args)

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

	row := tx.QueryRow(ctx, `
		DELETE FROM scan_configs 
		WHERE id = @id 
		RETURNING *`, args)

	var config ScanConfiguration
	err := row.Scan(&config.ID, &config.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	return err
}

func (p PostgresScanRepository) ListScans(ctx context.Context, tx pgx.Tx) ([]ScanExecution, error) {
	rows, err := tx.Query(ctx, `
		SELECT * 
		FROM scans`)

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
		err = rows.Scan(&scan.ID, &scan.ScanConfigurationID, &scan.StartTime, &scan.EndTime, &scan.Status)
		if err != nil {
			return nil, err
		}

		// get assets associated with scan
		rows, err = tx.Query(ctx, `
			SELECT *
			FROM assets
			INNER JOIN public.scan_asset_map sam on assets.id = sam.asset_id
			WHERE sam.scan_id = $1;
		`, scan.ID)

		if err != nil {
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
		scan.Assets = assets

		scans = append(scans, scan)
	}

	return scans, nil
}

func (p PostgresScanRepository) GetScan(ctx context.Context, tx pgx.Tx, id string) (*ScanExecution, error) {
	row := tx.QueryRow(ctx, `
		SELECT * 
		FROM scans 
		WHERE id = $1`, id)

	var scan ScanExecution
	err := row.Scan(&scan.ID, &scan.ScanConfigurationID, &scan.StartTime, &scan.EndTime, &scan.Status)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	// get assets associated with scan
	var assets []ScanAsset
	row = tx.QueryRow(ctx, `
		SELECT *
		FROM assets
		INNER JOIN public.scan_asset_map sam on assets.id = sam.asset_id
		WHERE sam.scan_id = $1;
	`, scan.ID)

	var asset ScanAsset
	err = row.Scan(&asset.ID, &asset.Endpoint)
	if err != nil {
		return nil, err
	}
	assets = append(assets, asset)

	scan.Assets = assets

	return &scan, nil
}

func (p PostgresScanRepository) CreateScan(ctx context.Context, tx pgx.Tx, scanRun ScanExecution) error {
	args := pgx.NamedArgs{
		"id":              scanRun.ID,
		"scan_config_id":  scanRun.ScanConfigurationID,
		"scan_start_time": scanRun.StartTime,
		"scan_end_time":   scanRun.EndTime,
		"status":          scanRun.Status,
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO scans (id, scan_config_id, scan_start_time, scan_end_time, status) 
		VALUES(@id, @scan_config_id, @scan_start_time, @scan_end_time, @status)`, args)

	// register assets
	for _, asset := range scanRun.Assets {
		args = pgx.NamedArgs{
			"scan_id":  scanRun.ID,
			"asset_id": asset.ID,
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO scan_asset_map (scan_id, asset_id) 
			VALUES(@scan_id, @asset_id)`, args)

		if err != nil {
			return err
		}
	}

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

	row := tx.QueryRow(ctx, `
		UPDATE scans 
		SET scan_config_id = @scan_config_id, scan_start_time = @scan_start_time, scan_end_time = @scan_end_time, status = @status 
		WHERE id = @id 
		RETURNING *`, args)

	var scan ScanExecution
	err := row.Scan(&scan.ID, &scan.ScanConfigurationID, &scan.StartTime, &scan.EndTime, &scan.Status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func (p PostgresScanRepository) PutAssetDiscoveryResult(ctx context.Context, tx pgx.Tx, result ScanAssetDiscoveryResult) error {
	args := pgx.NamedArgs{
		"asset_id": result.AssetID,
		"port":     result.Port,
		"protocol": result.Protocol,
	}

	// check if already exists
	row := tx.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM asset_discovery 
		WHERE asset_id = @asset_id AND port = @port AND protocol = @protocol`, args)

	var count int
	err := row.Scan(&count)
	if err != nil {
		return err
	}

	args = pgx.NamedArgs{
		"asset_id":   result.AssetID,
		"port":       result.Port,
		"protocol":   result.Protocol,
		"first_seen": result.FirstSeen,
		"last_seen":  result.LastSeen,
	}

	if count > 0 {
		// update
		_, err = tx.Exec(ctx, `
			UPDATE asset_discovery 
			SET last_seen = @last_seen 
			WHERE asset_id = @asset_id AND port = @port AND protocol = @protocol`, args)

		if err != nil {
			return err
		}
	} else {
		// insert
		_, err = tx.Exec(ctx, `
			INSERT INTO asset_discovery (asset_id, port, protocol, first_seen, last_seen) 
			VALUES(@asset_id, @port, @protocol, @first_seen, @last_seen)`, args)

		if err != nil {
			return err
		}
	}

	return nil
}

func (p PostgresScanRepository) ListAssetDiscoveryResults(ctx context.Context, tx pgx.Tx, assetID string) ([]ScanAssetDiscoveryResult, error) {
	rows, err := tx.Query(ctx, `
		SELECT * 
		FROM asset_discovery 
		WHERE asset_id = $1`, assetID)

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
	row := tx.QueryRow(ctx, `
		SELECT COUNT(*) 
		FROM asset_discovery 
		WHERE asset_id = $1`, assetID)

	var portCount int
	err := row.Scan(&portCount)
	if err != nil {
		return nil, err
	}

	// find timestamp of last discovery scan
	row = tx.QueryRow(ctx, `
		SELECT s.scan_end_time
		FROM
			scans s
		INNER JOIN public.scan_asset_map sam on s.id = sam.scan_id
		WHERE sam.asset_id = $1
		ORDER BY s.scan_end_time DESC
		LIMIT 1;
    `, assetID)

	var lastDiscoveryTime time.Time
	err = row.Scan(&lastDiscoveryTime)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			lastDiscoveryTime = time.Time{}
		} else {
			return nil, err
		}
	}

	stats := ScanAssetStats{
		DiscoveredPortsCount: portCount,
		LastDiscovery:        lastDiscoveryTime,
	}
	return &stats, nil
}

func (p PostgresScanRepository) GetAssetHistory(ctx context.Context, tx pgx.Tx, assetID string) ([]AssetHistoryEntry, error) {
	rows, err := tx.Query(ctx, `
		SELECT * 
		FROM asset_history
		WHERE asset_id = $1;
	`, assetID)

	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			return []AssetHistoryEntry{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var entries []AssetHistoryEntry
	for rows.Next() {
		var entry AssetHistoryEntry
		err = rows.Scan(&entry.ID, &entry.AssetID, &entry.Type, &entry.UserID, &entry.Time, &entry.Data)
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

func (p PostgresScanRepository) AddAssetHistoryEntry(ctx context.Context, tx pgx.Tx, entry AssetHistoryEntry) error {
	args := pgx.NamedArgs{
		"id":         entry.ID,
		"asset_id":   entry.AssetID,
		"event_type": entry.Type,
		"user_id":    entry.UserID,
		"timestamp":  entry.Time,
		"event_data": entry.Data,
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO asset_history (id, asset_id, event_type, user_id, timestamp, event_data) 
		VALUES(@id, @asset_id, @event_type, @user_id, @timestamp, @event_data)`, args)

	return err
}

func NewPostgresScanRepository() *PostgresScanRepository {
	return &PostgresScanRepository{
		logger: logging.GetLogger(logging.DataAccess),
	}
}
