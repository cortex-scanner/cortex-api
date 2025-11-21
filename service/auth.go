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

var ErrUnauthenticated = errors.New("unauthenticated")

type CreateTokenOptions struct {
	UserID    string
	UserAgent string
	SourceIP  string
}

type AuthService interface {
	ListUsers(ctx context.Context) ([]repository.User, error)
	GetUser(ctx context.Context, id string) (*repository.User, error)

	CheckUsernamePassword(ctx context.Context, username string, password string) (*repository.User, error)
	ValidateToken(ctx context.Context, tokenString string) (*repository.User, string, error)
	CreateSessionToken(ctx context.Context, opt CreateTokenOptions) (*repository.AuthToken, string, error)
	RevokeToken(ctx context.Context, tokenString string) error
}

type authService struct {
	logger *slog.Logger
	repo   repository.AuthRepository
	pool   *pgxpool.Pool
}

func (s authService) CheckUsernamePassword(ctx context.Context, username string, password string) (*repository.User, error) {
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

	// get user
	user, err := s.repo.GetUserByUsername(ctx, tx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("authentication request for unknown user %s", username))
			return nil, ErrUnauthenticated
		}
		return nil, err
	}

	match, err := crypto.ValidatePasswordWithArgonHash(password, user.Password)
	if err != nil {
		return nil, err
	}
	if !match {
		s.logger.InfoContext(ctx, fmt.Sprintf("authentication request for user %s failed: password does not match", username))
		return nil, ErrUnauthenticated
	}

	return user, nil
}

func (s authService) ValidateToken(ctx context.Context, tokenString string) (*repository.User, string, error) {
	components, err := parseTokenString(tokenString)
	if err != nil {
		return nil, "", err
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("validating token %s", components.id))

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

	authToken, err := s.repo.GetToken(ctx, tx, components.id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("unknown token %s", components.id))
			return nil, "", ErrUnauthenticated
		}
		return nil, "", err
	}

	// check if authToken is expired
	if authToken.ExpiresAt.Before(time.Now()) {
		s.logger.DebugContext(ctx, fmt.Sprintf("token %s expired", authToken.ID))
		return nil, "", ErrUnauthenticated
	}

	// validate hash
	match, err := crypto.ValidatePasswordWithArgonHash(components.secret, authToken.Hash)
	if err != nil {
		s.logger.DebugContext(ctx, "failed to validate token", logging.FieldError, err)
		return nil, "", ErrUnauthenticated
	}
	if !match {
		s.logger.DebugContext(ctx, fmt.Sprintf("token %s failed validation", authToken.ID))
		return nil, "", ErrUnauthenticated
	}

	user, err := s.repo.GetUser(ctx, tx, authToken.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("unknown user %s for token", authToken.UserID))
			return nil, "", ErrUnauthenticated
		}
		return nil, "", err
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("authentication request for user %s (%s) using id %s is valid",
		user.ID, user.Username, authToken.ID))
	return user, components.id, nil
}

func (s authService) CreateSessionToken(ctx context.Context, opt CreateTokenOptions) (*repository.AuthToken, string, error) {
	s.logger.DebugContext(ctx, fmt.Sprintf("creating session token for user %s", opt.UserID))

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

	// check if user exists first
	_, err = s.repo.GetUser(ctx, tx, opt.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("requested to create token for unknown user id %s", opt.UserID))
		}
		return nil, "", err
	}

	// TODO: make token expiration configurable
	expiration := time.Now().Add(time.Hour * 24 * 7)

	tokenComponents := newToken()

	hash, err := crypto.CalculateArgonHash(tokenComponents.secret)
	if err != nil {
		return nil, "", err
	}

	authToken := repository.AuthToken{
		ID:        tokenComponents.id,
		UserID:    opt.UserID,
		Hash:      hash,
		UserAgent: opt.UserAgent,
		SourceIP:  opt.SourceIP,
		Revoked:   false,
		CreatedAt: time.Now(),
		ExpiresAt: expiration,
	}

	err = s.repo.StoreToken(ctx, tx, &authToken)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create token", logging.FieldError, err)
		return nil, "", err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("created token for user %s with id %s", opt.UserID, authToken.ID))
	return &authToken, tokenComponents.ToTokenString(), nil
}

func (s authService) RevokeToken(ctx context.Context, tokenString string) error {
	components, err := parseTokenString(tokenString)
	if err != nil {
		return err
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("invalidating token with token %s", components.id))

	tx, err := s.pool.Begin(ctx)
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

	err = s.repo.DeleteToken(ctx, tx, components.id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete token", logging.FieldError, err)
		return err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("deleted token %s", components.id))
	return nil
}

func (s authService) ListUsers(ctx context.Context) ([]repository.User, error) {
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

	users, err := s.repo.ListUsers(ctx, tx)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to list users", logging.FieldError, err)
		return nil, err
	}
	return users, nil
}

func (s authService) GetUser(ctx context.Context, id string) (*repository.User, error) {
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

	user, err := s.repo.GetUser(ctx, tx, id)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get user",
			logging.FieldUserID, id,
			logging.FieldError, err)
		return nil, err
	}
	return user, nil
}

func NewAuthService(authRepo repository.AuthRepository, pool *pgxpool.Pool) AuthService {
	return authService{
		repo:   authRepo,
		logger: logging.GetLogger(logging.Auth),
		pool:   pool,
	}
}
