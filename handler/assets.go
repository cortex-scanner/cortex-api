package handler

import (
	"cortex/service"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type createAssetRequestBody struct {
	Endpoint string `json:"endpoint" validate:"required,max=2048"`
}

type updateAssetRequestBody struct {
	ID       string `json:"id" validate:"required,uuid4"`
	Endpoint string `json:"endpoint" validate:"required,max=2048"`
}

type AssetHandler struct {
	validate    *validator.Validate
	scanService service.ScanService
}

func NewAssetHandler(scanService service.ScanService) *AssetHandler {
	return &AssetHandler{
		scanService: scanService,
		validate:    validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h AssetHandler) HandleList(w http.ResponseWriter, r *http.Request) error {
	assets, err := h.scanService.ListAssets(r.Context())
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, assets); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AssetHandler) HandleGet(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	asset, err := h.scanService.GetAsset(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, asset); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AssetHandler) HandleCreate(w http.ResponseWriter, r *http.Request) error {
	var requestBody createAssetRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	asset, err := h.scanService.CreateAsset(r.Context(), requestBody.Endpoint)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOneCreated(w, r, asset); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AssetHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	var requestBody updateAssetRequestBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	asset, err := h.scanService.UpdateAsset(r.Context(), id, requestBody.Endpoint)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, asset); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AssetHandler) HandleDelete(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")

	asset, err := h.scanService.DeleteAsset(r.Context(), id)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondOne(w, r, asset); err != nil {
		return WrapError(err)
	}
	return nil
}
