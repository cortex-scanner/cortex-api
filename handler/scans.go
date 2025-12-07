package handler

import (
	"cortex/service"
	"net/http"
	"time"
)

type runScanRequestBody struct {
	ScanConfigId string   `json:"configId"`
	AssetIDs     []string `json:"assetIds"`
}

type updateScanRequestBody struct {
	Status         string `json:"status"`
	StartTimestamp int    `json:"startTime"`
	EndTimestamp   int    `json:"endTime"`
}

type ScanHandler struct {
	scanService service.ScanService
}

func NewScanHandler(scanService service.ScanService) *ScanHandler {
	return &ScanHandler{
		scanService: scanService,
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
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

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
	err := ValidateRequestBody(r, &requestBody,
		Field(&requestBody.ScanConfigId, Required(), UUID()),
		Field(&requestBody.AssetIDs, Required(), MinItems(1), Each(UUID())),
	)
	if err != nil {
		return WrapError(err)
	}

	scan, err := h.scanService.RunScan(r.Context(), requestBody.ScanConfigId, requestBody.AssetIDs)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, scan); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h ScanHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) error {
	id, err := ValidateParam(r, "id")
	if err != nil {
		return WrapError(err)
	}

	var requestBody updateScanRequestBody
	err = ValidateRequestBody(r, &requestBody,
		Field(&requestBody.Status, In("queued", "running", "complete", "failed", "cancelled")),
		Field(&requestBody.StartTimestamp, Min(0)),
		Field(&requestBody.EndTimestamp, Min(0)),
	)
	if err != nil {
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
