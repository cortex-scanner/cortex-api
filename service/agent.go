package service

import (
	"context"
	"cortex/crypto"
	"cortex/logging"
	"cortex/repository"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentService interface {
	ListAgents(ctx context.Context) ([]repository.Agent, error)
	GetAgent(ctx context.Context, id string) (*repository.Agent, error)
	CreateAgent(ctx context.Context, name string) (*repository.Agent, string, error)
	CreateAgentWithToken(ctx context.Context, tokenPlain string, name string) (*repository.Agent, error)
	UpdateAgent(ctx context.Context, id string, name string) (*repository.Agent, error)
	DeleteAgent(ctx context.Context, id string) (*repository.Agent, error)
}

type agentService struct {
	logger *slog.Logger
	repo   repository.AgentRepository
	pool   *pgxpool.Pool
}

func (s agentService) CreateAgentWithToken(ctx context.Context, tokenPlain string, name string) (*repository.Agent, error) {
	// Parse the token to extract the secret part for hashing
	tokenComponents, err := parseTokenString(tokenPlain)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to parse token string", logging.FieldError, err)
		return nil, fmt.Errorf("invalid token format: %w", err)
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("creating agent with token id %s", tokenComponents.id))

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

	// Check if agent with this token ID already exists
	existingAgent, err := s.repo.GetAgent(ctx, tx, tokenComponents.id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		s.logger.ErrorContext(ctx, "failed to check for existing agent", logging.FieldError, err)
		return nil, err
	}

	// If agent exists, return it
	if existingAgent != nil {
		s.logger.DebugContext(ctx, fmt.Sprintf("agent with id %s already exists, returning existing agent", tokenComponents.id))
		return existingAgent, nil
	}

	// Hash the token secret
	hash, err := crypto.CalculateArgonHash(tokenComponents.secret)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to calculate token hash", logging.FieldError, err)
		return nil, err
	}

	// Create new agent
	agent := repository.Agent{
		ID:        tokenComponents.id,
		Name:      name,
		TokenHash: hash,
		CreatedAt: time.Now(),
	}

	err = s.repo.CreateAgent(ctx, tx, agent)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create agent", logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("created agent %s with id %s", name, agent.ID))
	return &agent, nil
}

func (s agentService) ListAgents(ctx context.Context) ([]repository.Agent, error) {
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

	agents, err := s.repo.ListAgents(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list agents", logging.FieldError, err)
		return nil, err
	}
	return agents, nil
}

func (s agentService) GetAgent(ctx context.Context, id string) (*repository.Agent, error) {
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

	agent, err := s.repo.GetAgent(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get agent",
			logging.FieldError, err)
		return nil, err
	}
	return agent, nil
}

func (s agentService) CreateAgent(ctx context.Context, name string) (*repository.Agent, string, error) {
	s.logger.DebugContext(ctx, fmt.Sprintf("creating agent with name %s", name))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, "", err
	}
	defer func() {
		switch err {
		case nil:
			err = tx.Commit(ctx)
		default:
			_ = tx.Rollback(ctx)
		}
	}()

	// Generate token components for agent
	tokenComponents := newToken()

	// Hash the token secret
	hash, err := crypto.CalculateArgonHash(tokenComponents.secret)
	if err != nil {
		return nil, "", err
	}

	agent := repository.Agent{
		ID:        tokenComponents.id,
		Name:      name,
		TokenHash: hash,
		CreatedAt: time.Now(),
	}

	err = s.repo.CreateAgent(ctx, tx, agent)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create agent", logging.FieldError, err)
		return nil, "", err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("created agent %s with id %s", name, agent.ID))
	return &agent, tokenComponents.ToTokenString(), nil
}

func (s agentService) UpdateAgent(ctx context.Context, id string, name string) (*repository.Agent, error) {
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

	agent, err := s.repo.GetAgent(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get agent for update",
			logging.FieldError, err)
		return nil, err
	}

	agent.Name = name
	err = s.repo.UpdateAgent(ctx, tx, *agent)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update agent",
			logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("updated agent %s", id))

	return agent, nil
}

func (s agentService) DeleteAgent(ctx context.Context, id string) (*repository.Agent, error) {
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

	agent, err := s.repo.GetAgent(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get agent for deletion",
			logging.FieldError, err)
		return nil, err
	}

	err = s.repo.DeleteAgent(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete agent",
			logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("deleted agent %s", id))

	return agent, nil
}

func NewAgentService(agentRepo repository.AgentRepository, pool *pgxpool.Pool) AgentService {
	return &agentService{
		repo:   agentRepo,
		logger: logging.GetLogger(logging.Agent),
		pool:   pool,
	}
}
