package middleware

import (
	"context"
	cortexContext "cortex/context"
	"cortex/logging"
	"cortex/service"
	"log/slog"
	"net/http"
)

const sessionCookieName = "SESSID"

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

		// check for session cookie
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil {
			h.logger.DebugContext(r.Context(), "failed to get cookie from request", logging.FieldError, err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// validate session
		user, err := h.authService.ValidateSession(r.Context(), cookie.Value)
		if err != nil {
			h.logger.DebugContext(r.Context(), "failed to validate session", logging.FieldError, err)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		h.logger.DebugContext(r.Context(), "authenticated user", logging.FieldUserID, user.ID,
			logging.FieldSessionToken, cookie.Value)

		info := cortexContext.UserInfoData{
			UserID:       user.ID,
			Username:     user.Username,
			SessionToken: cookie.Value,
		}

		ctx := context.WithValue(r.Context(), cortexContext.KeyUserInfo, info)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
