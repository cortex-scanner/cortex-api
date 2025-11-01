package logging

import (
	"context"
	cortexContext "cortex/context"
	"log/slog"
)

const (
	FieldRequestID string = "requestId"
	FieldError     string = "error"
)

type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if val, ok := ctx.Value(cortexContext.KeyRequestID).(string); ok {
		r.AddAttrs(slog.String(FieldRequestID, val))
	}

	return h.Handler.Handle(ctx, r)
}
