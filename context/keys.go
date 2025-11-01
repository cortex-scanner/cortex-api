package context

import (
	"context"
)

type Key string

const (
	KeyRequestID Key = "request-id"
)

func RequestID(ctx context.Context) string {
	if val, ok := ctx.Value(KeyRequestID).(string); ok {
		return val
	}

	return ""
}
