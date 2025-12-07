package handler

import (
	"cortex/service"
	"net/http"
)

type createConfigRequestBody struct {
	Name   string `json:"name"`
	Engine string `json:"engine"`
}

type updateConfigRequestBody struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ScanConfigHandler struct {
	scanService service.ScanService
}

func NewScanConfigHandler(scanService service.ScanService) *ScanConfigHandler {
	return &ScanConfigHandler{
		scanService: scanService,
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
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

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
	err := ValidateRequestBody(r, &requestBody,
		Field(&requestBody.Name, Required(), Length(1, 1000)),
		Field(&requestBody.Engine, Required(), In("naabu")),
	)
	if err != nil {
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
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	var requestBody updateConfigRequestBody
	err = ValidateRequestBody(r, &requestBody,
		Field(&requestBody.ID, Required(), UUID()),
		Field(&requestBody.Name, Required(), Length(1, 1000)),
	)
	if err != nil {
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
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	config, err := h.scanService.DeleteScanConfig(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, config); err != nil {
		return WrapError(err)
	}
	return nil
}
