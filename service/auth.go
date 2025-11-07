package service

import (
	"context"
	"cortex/crypto"
	"cortex/logging"
	"cortex/repository"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUnauthenticated = errors.New("unauthenticated")

type CreateSessionOptions struct {
	UserID    string
	UserAgent string
	SourceIP  string
}
type AuthService interface {
	ListUsers(ctx context.Context) ([]repository.User, error)
	GetUser(ctx context.Context, id string) (*repository.User, error)

	CheckUsernamePassword(ctx context.Context, username string, password string) (*repository.User, error)
	ValidateSession(ctx context.Context, token string) (*repository.User, error)
	CreateSession(ctx context.Context, opt CreateSessionOptions) (*repository.Session, error)
	DeleteSession(ctx context.Context, token string) error
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

	// TODO: check password
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

func (s authService) ValidateSession(ctx context.Context, token string) (*repository.User, error) {
	s.logger.DebugContext(ctx, fmt.Sprintf("validating session with token %s", token))

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

	session, err := s.repo.GetSession(ctx, tx, token)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("unknown session token %s", token))
			return nil, ErrUnauthenticated
		}
		return nil, err
	}

	user, err := s.repo.GetUser(ctx, tx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("unknown user %s for session", session.UserID))
			return nil, ErrUnauthenticated
		}
		return nil, err
	}

	s.logger.DebugContext(ctx, fmt.Sprintf("authentication request for token user %s (%s) using session token %s is valid", user.ID, user.Username, token))
	return user, nil
}

func (s authService) CreateSession(ctx context.Context, opt CreateSessionOptions) (*repository.Session, error) {
	s.logger.DebugContext(ctx, fmt.Sprintf("creating session for user %s", opt.UserID))

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

	// check if user exists first
	_, err = s.repo.GetUser(ctx, tx, opt.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			s.logger.WarnContext(ctx, fmt.Sprintf("requested to create session for unknown user id %s", opt.UserID))
		}
		return nil, err
	}

	// TODO: make session expiration configurable
	expiration := time.Now().Add(time.Hour * 24 * 7)

	session := repository.Session{
		UserID:    opt.UserID,
		Token:     s.generateSessionID(),
		UserAgent: opt.UserAgent,
		SourceIP:  opt.SourceIP,
		Revoked:   false,
		CreatedAt: time.Now(),
		ExpiresAt: expiration,
	}

	err = s.repo.CreateSession(ctx, tx, &session)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to create session", logging.FieldError, err)
		return nil, err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("created session for user %s with token %s", opt.UserID, session.Token))
	return &session, nil
}

func (s authService) DeleteSession(ctx context.Context, token string) error {
	s.logger.DebugContext(ctx, fmt.Sprintf("invalidating session with token %s", token))

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

	err = s.repo.DeleteSession(ctx, tx, token)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to delete session", logging.FieldError, err)
		return err
	}

	s.logger.InfoContext(ctx, fmt.Sprintf("deleted session with token %s", token))
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

func (s authService) generateSessionID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}

func NewAuthService(authRepo repository.AuthRepository, pool *pgxpool.Pool) AuthService {
	return authService{
		repo:   authRepo,
		logger: logging.GetLogger(logging.Auth),
		pool:   pool,
	}
}
