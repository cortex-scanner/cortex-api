package handler

import (
	"cortex/service"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type FindingHandler struct {
	service  service.FindingService
	validate *validator.Validate
}

func NewFindingHandler(service service.FindingService) *FindingHandler {
	return &FindingHandler{
		service:  service,
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h FindingHandler) HandleGet(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	finding, err := h.service.GetFinding(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, finding); err != nil {
		return WrapError(err)
	}
	return nil
}
