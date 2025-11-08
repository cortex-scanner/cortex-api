package middleware

import (
	"context"
	cortexContext "cortex/context"
	"cortex/logging"
	"cortex/service"
	"log/slog"
	"net/http"
	"strings"
)

const tokenHeader = "Authorization"

type Authentication struct {
	logger      *slog.Logger
	authService service.AuthService
}

func NewAuthenticationMiddleware(authService service.AuthService) *Authentication {
	return &Authentication{
		logger:      logging.GetLogger(logging.Auth),
		authService: authService,
	}
}

func (h *Authentication) OnRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.DebugContext(r.Context(), "authenticating user")

		// check for token header
		authHeader := r.Header.Get(tokenHeader)
		if authHeader == "" {
			h.logger.DebugContext(r.Context(), "failed to get token from request")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		headerPrefix := "Bearer "
		tokenString, formatOk := strings.CutPrefix(authHeader, headerPrefix)
		if !formatOk {
			h.logger.DebugContext(r.Context(), "invalid token format, expected Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// validate token
		user, tokenId, err := h.authService.ValidateToken(r.Context(), tokenString)
		if err != nil {
			h.logger.DebugContext(r.Context(), "failed to validate token", logging.FieldError, err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
		if err != nil {
			h.logger.DebugContext(r.Context(), "failed to validate session", logging.FieldError, err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		h.logger.DebugContext(r.Context(), "authenticated user", logging.FieldUserID, user.ID,
			logging.FieldTokenID, tokenId)

		info := cortexContext.UserInfoData{
			UserID:   user.ID,
			Username: user.Username,
			TokenID:  tokenId,
		}

		ctx := context.WithValue(r.Context(), cortexContext.KeyUserInfo, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
