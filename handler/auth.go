package handler

import (
	"cortex/service"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type AuthHandler struct {
	authService service.AuthService
	validate    *validator.Validate
}

const sessionCookieName = "SESSID"

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validate:    validator.New(validator.WithRequiredStructEnabled()),
	}
}

type usernamePasswordLoginRequestBody struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

func (h AuthHandler) HandleUsernamePasswordLogin(w http.ResponseWriter, r *http.Request) error {
	var requestBody usernamePasswordLoginRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
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

	sessionOpt := service.CreateSessionOptions{
		UserID:    user.ID,
		UserAgent: r.UserAgent(),
		SourceIP:  src,
	}
	session, err := h.authService.CreateSession(r.Context(), sessionOpt)
	if err != nil {
		return WrapError(err)
	}

	cookie := http.Cookie{
		Name:     sessionCookieName,
		Value:    session.Token,
		Path:     "/",
		HttpOnly: true,
		// TODO: make this configurable
		Secure:  false,
		Expires: session.ExpiresAt,
	}
	http.SetCookie(w, &cookie)

	if err = RespondOne(w, r, user); err != nil {
		return WrapError(err)
	}
	return nil
}
