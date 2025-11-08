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

func (p PostgresAuthRepository) StoreToken(ctx context.Context, tx pgx.Tx, token *AuthToken) error {
	args := pgx.NamedArgs{
		"id":         token.ID,
		"user_id":    token.UserID,
		"hash":       token.Hash,
		"user_agent": token.UserAgent,
		"source_ip":  token.SourceIP,
		"revoked":    token.Revoked,
		"created_at": token.CreatedAt,
		"expires_at": token.ExpiresAt,
	}

	_, err := tx.Exec(ctx, `INSERT INTO tokens (id, user_id, hash, user_agent, source_ip, revoked, created_at, expires_at) 
								VALUES(@id, @user_id, @hash, @user_agent, @source_ip, @revoked, @created_at, @expires_at)`, args)

	return err
}

func (p PostgresAuthRepository) GetToken(ctx context.Context, tx pgx.Tx, tokenId string) (*AuthToken, error) {
	row := tx.QueryRow(ctx, "SELECT * FROM tokens WHERE id = $1", tokenId)

	var token AuthToken
	err := row.Scan(&token.ID, &token.Hash, &token.UserID, &token.CreatedAt, &token.ExpiresAt, &token.SourceIP, &token.Revoked, &token.UserAgent)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &token, nil
}

func (p PostgresAuthRepository) DeleteToken(ctx context.Context, tx pgx.Tx, tokenId string) error {
	args := pgx.NamedArgs{
		"id": tokenId,
	}

	row := tx.QueryRow(ctx, `UPDATE tokens SET revoked=true WHERE id=@id`, args)
	var token AuthToken
	err := row.Scan(&token.ID, &token.Hash, &token.UserID, &token.CreatedAt, &token.ExpiresAt, &token.SourceIP, &token.Revoked, &token.UserAgent)
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
