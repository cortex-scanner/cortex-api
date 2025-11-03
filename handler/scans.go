package handler

import (
	"cortex/service"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
)

type runScanRequestBody struct {
	ScanConfigId string `json:"configId" validate:"required,uuid4"`
	Type         string `json:"type" validate:"required,oneof=discovery vuln discovery+vuln"`
}

type updateScanRequestBody struct {
	Status         string `json:"status"`
	StartTimestamp int    `json:"startTime"`
	EndTimestamp   int    `json:"endTime"`
}

type ScanHandler struct {
	scanService service.ScanService
	validate    *validator.Validate
}

func NewScanHandler(scanService service.ScanService) *ScanHandler {
	return &ScanHandler{
		scanService: scanService,
		validate:    validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h ScanHandler) HandleList(w http.ResponseWriter, r *http.Request) error {
	scans, err := h.scanService.ListScans(r.Context())
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, scans); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanHandler) HandleGet(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	scan, err := h.scanService.GetScan(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, scan); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanHandler) HandleRun(w http.ResponseWriter, r *http.Request) error {
	var requestBody runScanRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	scan, err := h.scanService.RunScan(r.Context(), requestBody.ScanConfigId, requestBody.Type)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, scan); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	var requestBody updateScanRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	update := service.ScanUpdateOptions{}

	update.Status = requestBody.Status
	update.StartTime = time.Unix(int64(requestBody.StartTimestamp), 0)
	update.EndTime = time.Unix(int64(requestBody.EndTimestamp), 0)

	scan, err := h.scanService.UpdateScan(r.Context(), id, update)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, scan); err != nil {
		return WrapError(err)
	}

	return nil
}
