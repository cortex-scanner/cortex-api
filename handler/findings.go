package handler

import (
	"cortex/service"
	"net/http"
)

type FindingHandler struct {
	service service.FindingService
}

func NewFindingHandler(service service.FindingService) *FindingHandler {
	return &FindingHandler{
		service: service,
	}
}

func (h FindingHandler) HandleGet(w http.ResponseWriter, r *http.Request) error {
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	finding, err := h.service.GetFinding(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, finding); err != nil {
		return WrapError(err)
	}
	return nil
}
