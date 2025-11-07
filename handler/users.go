package handler

import (
	"cortex/service"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	authService service.AuthService
	validate    *validator.Validate
}

func NewUserHandler(authService service.AuthService) *UserHandler {
	return &UserHandler{
		authService: authService,
		validate:    validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h UserHandler) HandleListUsers(w http.ResponseWriter, r *http.Request) error {
	users, err := h.authService.ListUsers(r.Context())
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, users); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h UserHandler) HandleGetUser(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	user, err := h.authService.GetUser(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, user); err != nil {
		return WrapError(err)
	}
	return nil
}
