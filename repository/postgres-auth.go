package repository

import (
	"context"
	"cortex/logging"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
)

type PostgresAuthRepository struct {
	logger *slog.Logger
}

func (p PostgresAuthRepository) CreateSession(ctx context.Context, tx pgx.Tx, session *Session) error {
	args := pgx.NamedArgs{
		"user_id":    session.UserID,
		"token":      session.Token,
		"user_agent": session.UserAgent,
		"source_ip":  session.SourceIP,
		"revoked":    session.Revoked,
		"created_at": session.CreatedAt,
		"expires_at": session.ExpiresAt,
	}

	_, err := tx.Exec(ctx, `INSERT INTO sessions (user_id, token, user_agent, source_ip, revoked, created_at, expires_at) 
								VALUES(@user_id, @token, @user_agent, @source_ip, @revoked, @created_at, @expires_at)`, args)

	return err
}

func (p PostgresAuthRepository) GetSession(ctx context.Context, tx pgx.Tx, token string) (*Session, error) {
	row := tx.QueryRow(ctx, "SELECT * FROM sessions WHERE token = $1", token)

	var session Session
	err := row.Scan(&session.Token, &session.UserID, &session.CreatedAt, &session.ExpiresAt, &session.SourceIP, &session.Revoked, &session.UserAgent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &session, nil
}

func (p PostgresAuthRepository) DeleteSession(ctx context.Context, tx pgx.Tx, token string) error {
	args := pgx.NamedArgs{
		"token": token,
	}

	row := tx.QueryRow(ctx, `UPDATE sessions SET revoked=true WHERE token=@token`, args)
	var session Session
	err := row.Scan(&session.UserID, &session.Token, &session.UserAgent, &session.SourceIP, &session.Revoked, &session.CreatedAt, &session.ExpiresAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (p PostgresAuthRepository) ListUsers(ctx context.Context, tx pgx.Tx) ([]User, error) {
	rows, err := tx.Query(ctx, `
		SELECT * FROM users
	`)
	if err != nil {
		// return empty list if no identities are found
		if errors.Is(err, pgx.ErrNoRows) {
			return []User{}, nil
		}
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err = rows.Scan(&user.ID, &user.Provider, &user.Username, &user.Email, &user.DisplayName, &user.Password, &user.CreatedAt)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (p PostgresAuthRepository) GetUser(ctx context.Context, tx pgx.Tx, id string) (*User, error) {
	row := tx.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", id)

	var user User
	err := row.Scan(&user.ID, &user.Provider, &user.Username, &user.Email, &user.DisplayName, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (p PostgresAuthRepository) GetUserByUsername(ctx context.Context, tx pgx.Tx, username string) (*User, error) {
	row := tx.QueryRow(ctx, "SELECT * FROM users WHERE username = $1", username)

	var user User
	err := row.Scan(&user.ID, &user.Provider, &user.Username, &user.Email, &user.DisplayName, &user.Password, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func NewPostgresAuthRepository() *PostgresAuthRepository {
	return &PostgresAuthRepository{
		logger: logging.GetLogger(logging.DataAccess),
	}
}
