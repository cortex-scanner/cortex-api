package repository

import (
	"context"
	"cortex/logging"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Agent struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TokenHash string    `json:"-"`
	CreatedAt time.Time `json:"createdAt"`
}

func (a Agent) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		CreatedAt int64  `json:"createdAt"`
	}{
		ID:        a.ID,
		Name:      a.Name,
		CreatedAt: a.CreatedAt.Unix(),
	})
}

type AgentRepository interface {
	ListAgents(ctx context.Context, tx pgx.Tx) ([]Agent, error)
	GetAgent(ctx context.Context, tx pgx.Tx, id string) (*Agent, error)
	CreateAgent(ctx context.Context, tx pgx.Tx, agent Agent) error
	UpdateAgent(ctx context.Context, tx pgx.Tx, agent Agent) error
	DeleteAgent(ctx context.Context, tx pgx.Tx, id string) error
}

type PostgresAgentRepository struct {
	logger *slog.Logger
}

func (r PostgresAgentRepository) ListAgents(ctx context.Context, tx pgx.Tx) ([]Agent, error) {
	rows, err := tx.Query(ctx, `
		SELECT * 
		FROM agents`)

	if err != nil {
		// return empty list if no agents are found
		if errors.Is(err, pgx.ErrNoRows) {
			return []Agent{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var agents []Agent
	for rows.Next() {
		var agent Agent
		err = rows.Scan(&agent.ID, &agent.Name, &agent.TokenHash, &agent.CreatedAt)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}

	return agents, nil
}

func (r PostgresAgentRepository) GetAgent(ctx context.Context, tx pgx.Tx, id string) (*Agent, error) {
	row := tx.QueryRow(ctx, `
		SELECT * 
		FROM agents 
		WHERE id = $1`, id)

	var agent Agent
	err := row.Scan(&agent.ID, &agent.Name, &agent.TokenHash, &agent.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &agent, nil
}

func (r PostgresAgentRepository) CreateAgent(ctx context.Context, tx pgx.Tx, agent Agent) error {
	args := pgx.NamedArgs{
		"id":              agent.ID,
		"name":            agent.Name,
		"auth_token_hash": agent.TokenHash,
		"created_at":      agent.CreatedAt,
	}

	_, err := tx.Exec(ctx, `
		INSERT INTO agents (id, name, auth_token_hash, created_at) 
		VALUES(@id, @name, @auth_token_hash, @created_at)`, args)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
			r.logger.DebugContext(ctx, "agent name already exists", logging.FieldError, err)
			return ErrUniqueViolation
		}
		return err
	}

	return nil
}

func (r PostgresAgentRepository) UpdateAgent(ctx context.Context, tx pgx.Tx, agent Agent) error {
	args := pgx.NamedArgs{
		"id":   agent.ID,
		"name": agent.Name,
	}

	row := tx.QueryRow(ctx, `
		UPDATE agents 
		SET name = @name
		WHERE id = @id`, args)

	var updatedAgent Agent
	err := row.Scan(&updatedAgent.ID, &updatedAgent.Name, &updatedAgent.TokenHash, &updatedAgent.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == PgErrorCodeUniqueViolation {
			r.logger.DebugContext(ctx, "agent name already exists", logging.FieldError, err)
			return ErrUniqueViolation
		}
		return err
	}
	return nil
}

func (r PostgresAgentRepository) DeleteAgent(ctx context.Context, tx pgx.Tx, id string) error {
	args := pgx.NamedArgs{
		"id": id,
	}

	row := tx.QueryRow(ctx, `
		DELETE FROM agents 
		WHERE id = @id 
		RETURNING id, name, auth_token_hash, created_at`, args)

	var agent Agent
	err := row.Scan(&agent.ID, &agent.Name, &agent.TokenHash, &agent.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func NewPostgresAgentRepository() *PostgresAgentRepository {
	return &PostgresAgentRepository{
		logger: logging.GetLogger(logging.DataAccess),
	}
}
