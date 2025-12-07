package handler

import (
	cortexContext "cortex/context"
	"cortex/repository"
	"cortex/service"
	"net/http"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type usernamePasswordLoginRequestBody struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type tokenResponse struct {
	Token string           `json:"token"`
	User  *repository.User `json:"user"`
}

func (h AuthHandler) HandleUsernamePasswordLogin(w http.ResponseWriter, r *http.Request) error {
	var requestBody usernamePasswordLoginRequestBody
	err := ValidateRequestBody(r, &requestBody,
		Field(&requestBody.Username, Required(), Length(1, AnyLength)),
		Field(&requestBody.Password, Required(), Length(1, AnyLength)),
	)
	if err != nil {
		// always return 401 to not leak information for now
		return APIError{
			StatusCode: http.StatusUnauthorized,
			Message:    "unauthorized",
		}
	}

	// validate credentials
	user, err := h.authService.CheckUsernamePassword(r.Context(), requestBody.Username, requestBody.Password)
	if err != nil {
		return WrapError(err)
	}

	// create new session for user and set cookie
	src := r.RemoteAddr
	if r.Header.Get("X-Forwarded-For") != "" {
		src = r.Header.Get("X-Forwarded-For")
	}

	tokenOptions := service.CreateTokenOptions{
		UserID:    user.ID,
		UserAgent: r.UserAgent(),
		SourceIP:  src,
	}

	_, tokenString, err := h.authService.CreateSessionToken(r.Context(), tokenOptions)
	if err != nil {
		return WrapError(err)
	}

	response := tokenResponse{
		Token: tokenString,
		User:  user,
	}

	if err = RespondOne(w, r, response); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AuthHandler) HandleValidateToken(w http.ResponseWriter, r *http.Request) error {
	userInfo, err := cortexContext.UserInfo(r.Context())
	if err != nil {
		return WrapError(err)
	}

	user, err := h.authService.GetUser(r.Context(), userInfo.UserID)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, user); err != nil {
		return WrapError(err)
	}

	return nil
}
