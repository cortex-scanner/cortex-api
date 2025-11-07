package context

import (
	"context"
	"errors"
)

type Key string

const (
	KeyRequestID Key = "request-id"
	KeyUserInfo  Key = "user"
)

type UserInfoData struct {
	UserID       string
	Username     string
	SessionToken string
}

var ErrNoUserInfo = errors.New("no user info in context")

func RequestID(ctx context.Context) string {
	if val, ok := ctx.Value(KeyRequestID).(string); ok {
		return val
	}

	return ""
}

func UserInfo(ctx context.Context) (*UserInfoData, error) {
	if val, ok := ctx.Value(KeyUserInfo).(UserInfoData); ok {
		return &val, nil
	} else {
		return nil, ErrNoUserInfo
	}
}
