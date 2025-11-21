package context

import (
	"context"
	"errors"
)

type Key string

const (
	KeyRequestID Key = "request-id"
	KeyUserInfo  Key = "user"
	KeyAgentInfo Key = "agent"
)

type UserInfoData struct {
	UserID   string
	Username string
	TokenID  string
}

type AgentInfoData struct {
	AgentID string
}

var ErrNoUserInfo = errors.New("no user info in context")
var ErrNoAgentInfo = errors.New("no agent info in context")

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

func AgentInfo(ctx context.Context) (*AgentInfoData, error) {
	if val, ok := ctx.Value(KeyAgentInfo).(AgentInfoData); ok {
		return &val, nil
	} else {
		return nil, ErrNoAgentInfo
	}
}
