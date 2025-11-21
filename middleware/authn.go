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

const userTokenHeader = "Authorization"
const agentTokenHeader = "X-Agent-Token"

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
		h.logger.DebugContext(r.Context(), "authenticating request")

		// Try user authentication first
		ctx, userAuthSuccess := h.tryUserAuthentication(r)

		// Try agent authentication if user auth failed
		if !userAuthSuccess {
			var agentAuthSuccess bool
			ctx, agentAuthSuccess = h.tryAgentAuthentication(r)
			if !agentAuthSuccess {
				h.logger.DebugContext(r.Context(), "both user and agent authentication failed")
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// tryUserAuthentication attempts to authenticate using user token and returns updated context and success status
func (h *Authentication) tryUserAuthentication(r *http.Request) (context.Context, bool) {
	// check for user token header
	authHeader := r.Header.Get(userTokenHeader)
	if authHeader == "" {
		h.logger.DebugContext(r.Context(), "no user token found")
		return r.Context(), false
	}

	headerPrefix := "Bearer "
	tokenString, formatOk := strings.CutPrefix(authHeader, headerPrefix)
	if !formatOk {
		h.logger.DebugContext(r.Context(), "invalid user token format, expected Bearer")
		return r.Context(), false
	}

	// validate user token
	user, tokenId, err := h.authService.ValidateToken(r.Context(), tokenString)
	if err != nil {
		h.logger.DebugContext(r.Context(), "failed to validate user token", logging.FieldError, err)
		return r.Context(), false
	}

	h.logger.DebugContext(r.Context(), "authenticated user", logging.FieldUserID, user.ID,
		logging.FieldTokenID, tokenId)

	info := cortexContext.UserInfoData{
		UserID:   user.ID,
		Username: user.Username,
		TokenID:  tokenId,
	}

	ctx := context.WithValue(r.Context(), cortexContext.KeyUserInfo, info)
	return ctx, true
}

// tryAgentAuthentication attempts to authenticate using agent token and returns updated context and success status
func (h *Authentication) tryAgentAuthentication(r *http.Request) (context.Context, bool) {
	// check for agent token header
	agentToken := r.Header.Get(agentTokenHeader)
	if agentToken == "" {
		h.logger.DebugContext(r.Context(), "no agent token found")
		return r.Context(), false
	}

	// validate agent token
	agent, err := h.authService.ValidateAgentToken(r.Context(), agentToken)
	if err != nil {
		h.logger.DebugContext(r.Context(), "failed to validate agent token", logging.FieldError, err)
		return r.Context(), false
	}

	h.logger.DebugContext(r.Context(), "authenticated agent", logging.FieldAgentID, agent.ID)

	info := cortexContext.AgentInfoData{
		AgentID: agent.ID,
	}

	ctx := context.WithValue(r.Context(), cortexContext.KeyAgentInfo, info)
	return ctx, true
}
