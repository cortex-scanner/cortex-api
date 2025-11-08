package logging

import (
	"context"
	cortexContext "cortex/context"
	"log/slog"
)

const (
	FieldRequestID    string = "requestId"
	FieldError        string = "error"
	FieldScanConfigID string = "scanConfigId"
	FieldAssetID      string = "assetId"
	FieldScanID       string = "scanId"
	FieldUserID       string = "userId"
	FieldUsername     string = "username"
	FieldTokenID      string = "tokenId"
)

type ContextHandler struct {
	slog.Handler
}

func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if val, ok := ctx.Value(cortexContext.KeyRequestID).(string); ok {
		r.AddAttrs(slog.String(FieldRequestID, val))
	}

	if val, ok := ctx.Value(cortexContext.KeyUserInfo).(cortexContext.UserInfoData); ok {
		r.AddAttrs(
			slog.String(FieldUserID, val.UserID),
			slog.String(FieldUsername, val.Username),
			slog.String(FieldTokenID, val.TokenID),
		)
	}

	return h.Handler.Handle(ctx, r)
}
