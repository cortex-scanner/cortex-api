package repository

import (
	"context"
	"cortex/logging"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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
	pool   *pgxpool.Pool
}

func (p PostgresScanRepository) ListScanAssets(ctx context.Context) ([]ScanAsset, error) {
	tx, err := p.pool.Begin(ctx)
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

	rows, err := tx.Query(ctx, `
		SELECT * FROM assets
	`)
	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			// reset error to not trigger rollback
			err = nil
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

func (p PostgresScanRepository) GetScanAsset(ctx context.Context, id string) (*ScanAsset, error) {
	tx, err := p.pool.Begin(ctx)
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

	row := tx.QueryRow(ctx, "SELECT * FROM assets WHERE id = $1", id)

	var asset ScanAsset
	err = row.Scan(&asset.ID, &asset.Endpoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &asset, nil
}

func (p PostgresScanRepository) CreateScanAsset(ctx context.Context, scanAsset ScanAsset) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	args := pgx.NamedArgs{
		"id":       scanAsset.ID,
		"endpoint": scanAsset.Endpoint,
	}

	_, err = tx.Exec(ctx, "INSERT INTO assets (id, endpoint) VALUES(@id, @endpoint)", args)

	var pgErr *pgconn.PgError
	if err != nil && errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
		p.logger.DebugContext(ctx, "asset endpoint already exists", logging.FieldError, err)
		return ErrUniqueViolation
	}

	return nil
}

func (p PostgresScanRepository) UpdateScanAsset(ctx context.Context, scanAsset ScanAsset) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	args := pgx.NamedArgs{
		"id":       scanAsset.ID,
		"endpoint": scanAsset.Endpoint,
	}

	row := tx.QueryRow(ctx, "UPDATE assets SET endpoint = @endpoint WHERE id = @id RETURNING *", args)
	var asset ScanAsset
	err = row.Scan(&asset.ID, &asset.Endpoint)
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

func (p PostgresScanRepository) DeleteScanAsset(ctx context.Context, id string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	args := pgx.NamedArgs{
		"id": id,
	}

	row := tx.QueryRow(ctx, "DELETE FROM assets WHERE id = @id RETURNING *", args)
	var asset ScanAsset
	err = row.Scan(&asset.ID, &asset.Endpoint)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (p PostgresScanRepository) ListScanConfigurations(ctx context.Context) ([]ScanConfiguration, error) {
	tx, err := p.pool.Begin(ctx)
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
			err = nil
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

func (p PostgresScanRepository) GetScanConfiguration(ctx context.Context, id string) (*ScanConfiguration, error) {
	tx, err := p.pool.Begin(ctx)
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
			err = nil
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

func (p PostgresScanRepository) CreateScanConfiguration(ctx context.Context, scanConfiguration ScanConfiguration) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	// create scan config first, then in the same transaction associate all assets
	args := pgx.NamedArgs{
		"id":   scanConfiguration.ID,
		"name": scanConfiguration.Name,
	}

	_, err = tx.Exec(ctx, "INSERT INTO scan_configs (id, name) VALUES(@id, @name)", args)

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
func (p PostgresScanRepository) UpdateScanConfiguration(ctx context.Context, scanConfiguration ScanConfiguration) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	args := pgx.NamedArgs{
		"id":   scanConfiguration.ID,
		"name": scanConfiguration.Name,
	}

	row := tx.QueryRow(ctx, "UPDATE scan_configs SET name = @name WHERE id = @id RETURNING *", args)
	var config ScanConfiguration
	err = row.Scan(&config.ID, &config.Name)
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

func (p PostgresScanRepository) DeleteScanConfiguration(ctx context.Context, id string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	args := pgx.NamedArgs{
		"id": id,
	}

	row := tx.QueryRow(ctx, "DELETE FROM scan_configs WHERE id = @id RETURNING *", args)
	var config ScanConfiguration
	err = row.Scan(&config.ID, &config.Name)
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
			err = nil
			return nil
		}
	}

	return err
}

func NewPostgresScanRepository(pool *pgxpool.Pool) *PostgresScanRepository {
	return &PostgresScanRepository{
		logger: logging.GetLogger(logging.DataAccess),
		pool:   pool,
	}
}
