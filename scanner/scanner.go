package scanner

import (
	"context"
	"cortex/logging"
	"cortex/repository"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ScanEngineNaabu string = "naabu"
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
	switch config.Engine {
	case ScanEngineNaabu:
		scanner = NewNaabuScanner(s.repo, s.pool)
	default:
		return errors.New("unsupported scan engine")
	}

	go func() {
		err := scanner.Scan(context.WithoutCancel(ctx), scan, config)
		if err != nil {
			s.logger.ErrorContext(ctx, "scan failed", logging.FieldScanID, scan.ID, logging.FieldError, err)
		}
	}()

	return nil
}

func NewScanRunner(repo repository.ScanRepository, pool *pgxpool.Pool) *ScanRunner {
	return &ScanRunner{
		logger: logging.GetLogger(logging.Scan),
		repo:   repo,
		pool:   pool,
	}
}
