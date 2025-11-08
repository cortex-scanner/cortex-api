package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
)

type UserProvider string

const (
	UserProviderLocal UserProvider = "local"
)

type User struct {
	ID          string       `json:"id"`
	Provider    UserProvider `json:"provider"`
	Username    string       `json:"username"`
	Password    string       `json:"password"`
	Email       string       `json:"email"`
	DisplayName string       `json:"displayName"`
	CreatedAt   time.Time    `json:"createdAt"`
}

func (u User) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          string       `json:"id"`
		Provider    UserProvider `json:"provider"`
		Username    string       `json:"username"`
		Email       string       `json:"email"`
		DisplayName string       `json:"displayName"`
		CreatedAt   int64        `json:"createdAt"`
	}{
		ID:          u.ID,
		Provider:    u.Provider,
		Username:    u.Username,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt.Unix(),
	})
}

type AuthToken struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	UserID    string    `json:"userId"`
	UserAgent string    `json:"userAgent"`
	SourceIP  string    `json:"ip"`
	Revoked   bool      `json:"revoked"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s AuthToken) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string `json:"id"`
		UserID    string `json:"userId"`
		UserAgent string `json:"userAgent"`
		SourceIP  string `json:"ip"`
		Revoked   bool   `json:"revoked"`
		CreatedAt int64  `json:"createdAt"`
		ExpiresAt int64  `json:"expiresAt"`
	}{
		UserID:    s.UserID,
		UserAgent: s.UserAgent,
		SourceIP:  s.SourceIP,
		Revoked:   s.Revoked,
		CreatedAt: s.CreatedAt.Unix(),
		ExpiresAt: s.ExpiresAt.Unix(),
	})
}

type UserRepository interface {
	ListUsers(ctx context.Context, tx pgx.Tx) ([]User, error)
	GetUser(ctx context.Context, tx pgx.Tx, id string) (*User, error)
	GetUserByUsername(ctx context.Context, tx pgx.Tx, username string) (*User, error)
}

type TokenRepository interface {
	StoreToken(ctx context.Context, tx pgx.Tx, token *AuthToken) error
	GetToken(ctx context.Context, tx pgx.Tx, id string) (*AuthToken, error)
	DeleteToken(ctx context.Context, tx pgx.Tx, tokenId string) error
}

type AuthRepository interface {
	UserRepository
	TokenRepository
}
