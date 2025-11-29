package service

import (
	"context"
	cortexContext "cortex/context"
	"cortex/logging"
	"cortex/repository"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"hash"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CreateFindingOptions struct {
	AssetID string
	Type    repository.FindingType
	Data    map[string]any
}

type FindingService interface {
	CreateFinding(ctx context.Context, opts CreateFindingOptions) (*repository.AssetFinding, error)
	GetFinding(ctx context.Context, id string) (*repository.AssetFinding, error)
}

type findingService struct {
	repo   repository.ScanRepository
	logger *slog.Logger
	pool   *pgxpool.Pool
}

func (s findingService) GetFinding(ctx context.Context, id string) (*repository.AssetFinding, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	finding, err := s.repo.GetAssetFinding(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "unable to get finding", logging.FieldError, err)
		return nil, err
	}

	return finding, nil
}

func (s findingService) CreateFinding(ctx context.Context, opts CreateFindingOptions) (*repository.AssetFinding, error) {
	findingHash, err := s.calculateFindingHash(opts.Type, opts.Data)
	if err != nil {
		s.logger.Error("unable to calculate finding hash", logging.FieldError, err)
		return nil, err
	}

	agentInfo, err := cortexContext.AgentInfo(ctx)
	if err != nil {
		s.logger.Error("unable to get agent info", logging.FieldError, err)
		return nil, err
	}

	finding := repository.AssetFinding{
		ID:          uuid.New().String(),
		AssetID:     opts.AssetID,
		CreatedAt:   time.Now(),
		Type:        opts.Type,
		Data:        opts.Data,
		FindingHash: findingHash,
		AgentID:     agentInfo.AgentID,
	}

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

	err = s.repo.PutAssetFinding(ctx, tx, finding)
	if err != nil {
		s.logger.ErrorContext(ctx, "unable to store finding in database", logging.FieldError, err)
		return nil, err
	}

	return &finding, nil
}

func (s findingService) calculateFindingHash(findingType repository.FindingType, findingData map[string]any) (string, error) {
	calculator := newFindingHashCalculator(findingData)
	switch findingType {
	case repository.FindingTypePort:
		return calculator.addField("port").addField("protocol").calculateHash()
	case repository.FindingTypeVulnerability:
		return calculator.addField("template-id").addField("port").calculateHash()

	}
	return "", errors.New("unsupported finding type")
}

func NewFindingService(repo repository.ScanRepository, pool *pgxpool.Pool) FindingService {
	return &findingService{
		repo:   repo,
		pool:   pool,
		logger: logging.GetLogger(logging.Scan),
	}
}

type findingHashCalculator struct {
	data     map[string]any
	errors   []error
	hashFunc hash.Hash
}

func (c *findingHashCalculator) addField(field string) *findingHashCalculator {
	fieldValue, err := json.Marshal(c.data[field])
	if err != nil {
		c.errors = append(c.errors, err)
		return c
	}
	c.hashFunc.Write(fieldValue)
	return c
}

func (c *findingHashCalculator) calculateHash() (string, error) {
	if len(c.errors) > 0 {
		errorString := ""
		for _, err := range c.errors {
			errorString += err.Error() + ","
		}
		return "", errors.New("unable to calculate finding hash: " + errorString)
	}

	return hex.EncodeToString(c.hashFunc.Sum(nil)), nil
}

func newFindingHashCalculator(data map[string]any) *findingHashCalculator {
	return &findingHashCalculator{
		data:     data,
		hashFunc: sha256.New(),
	}
}
