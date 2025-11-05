package scanner

import (
	"context"
	"cortex/logging"
	"cortex/repository"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Scanner interface {
	Scan(ctx context.Context, scan repository.ScanExecution, config repository.ScanConfiguration) error
}

type ScanRunner struct {
	logger *slog.Logger
	repo   repository.ScanRepository
	pool   *pgxpool.Pool
}

func (s ScanRunner) Scan(ctx context.Context, scan repository.ScanExecution, config repository.ScanConfiguration) error {
	var scanner Scanner
	switch scan.Type {
	case "discovery":
		scanner = NewDiscoveryScanner(s.repo, s.pool)
	default:
		return errors.New("unsupported scan type")
	}

	// just start scan for now
	// TODO: run scan in goroutine
	return scanner.Scan(ctx, scan, config)
}

func NewScanRunner(repo repository.ScanRepository, pool *pgxpool.Pool) *ScanRunner {
	return &ScanRunner{
		logger: logging.GetLogger(logging.Scan),
		repo:   repo,
		pool:   pool,
	}
}
