package handler

import (
	"cortex/service"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type createConfigRequestBody struct {
	Name   string `json:"name" validate:"required,max=1000"`
	Engine string `json:"engine" validate:"required,oneof=naabu"`
}

type updateConfigRequestBody struct {
	ID   string `json:"id" validate:"required,uuid4"`
	Name string `json:"name" validate:"required,max=1000"`
}

type ScanConfigHandler struct {
	validate    *validator.Validate
	scanService service.ScanService
}

func NewScanConfigHandler(scanService service.ScanService) *ScanConfigHandler {
	return &ScanConfigHandler{
		scanService: scanService,
		validate:    validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h ScanConfigHandler) HandleList(w http.ResponseWriter, r *http.Request) error {
	configs, err := h.scanService.ListScanConfigs(r.Context())
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, configs); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanConfigHandler) HandleGet(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	config, err := h.scanService.GetScanConfig(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, config); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanConfigHandler) HandleCreate(w http.ResponseWriter, r *http.Request) error {
	var requestBody createConfigRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	config, err := h.scanService.CreateScanConfig(r.Context(), requestBody.Name)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOneCreated(w, r, config); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanConfigHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	var requestBody updateConfigRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	config, err := h.scanService.UpdateScanConfig(r.Context(), id, requestBody.Name)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, config); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanConfigHandler) HandleDelete(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	config, err := h.scanService.DeleteScanConfig(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, config); err != nil {
		return WrapError(err)
	}
	return nil
}
