package handler

import (
	"cortex/repository"
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

type createAssetFindingBody struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

type AssetHandler struct {
	validate       *validator.Validate
	scanService    service.ScanService
	findingService service.FindingService
}

func NewAssetHandler(scanService service.ScanService, findingService service.FindingService) *AssetHandler {
	return &AssetHandler{
		scanService:    scanService,
		findingService: findingService,
		validate:       validator.New(validator.WithRequiredStructEnabled()),
	}
}

func (h AssetHandler) HandleList(w http.ResponseWriter, r *http.Request) error {
	statsRequested := r.URL.Query().Get("stats") == "true"

	if statsRequested {
		// respond with stats
		assets, err := h.scanService.ListAssetsWithStats(r.Context())
		if err != nil {
			return WrapError(err)
		}

		if err = RespondMany(w, r, assets); err != nil {
			return WrapError(err)
		}

	} else {
		// plain asset
		assets, err := h.scanService.ListAssets(r.Context())
		if err != nil {
			return WrapError(err)
		}

		if err = RespondMany(w, r, assets); err != nil {
			return WrapError(err)
		}
	}

	return nil
}

func (h AssetHandler) HandleGet(w http.ResponseWriter, r *http.Request) error {
	id := r.PathValue("id")
	statsRequested := r.URL.Query().Get("stats") == "true"

	if statsRequested {
		// respond with stats
		asset, err := h.scanService.GetAssetWithStats(r.Context(), id)
		if err != nil {
			return WrapError(err)
		}

		if err = RespondOne(w, r, asset); err != nil {
			return WrapError(err)
		}
	} else {
		// plain asset
		asset, err := h.scanService.GetAsset(r.Context(), id)
		if err != nil {
			return WrapError(err)
		}

		if err = RespondOne(w, r, asset); err != nil {
			return WrapError(err)
		}
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

func (h AssetHandler) HandleListAssetFindings(w http.ResponseWriter, r *http.Request) error {
	assetId := r.PathValue("id")

	results, err := h.scanService.ListAssetFindings(r.Context(), assetId)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, results); err != nil {
		return WrapError(err)
	}
	return nil
}

func (h AssetHandler) HandleCreateFinding(w http.ResponseWriter, r *http.Request) error {
	assetId := r.PathValue("id")
	var requestBody createAssetFindingBody
	if err := ParseAndValidateBody(&requestBody, r, h.validate); err != nil {
		return WrapError(err)
	}

	// check if asset exists
	_, err := h.scanService.GetAsset(r.Context(), assetId)
	if err != nil {
		return WrapError(err)
	}

	finding, err := h.findingService.CreateFinding(r.Context(), service.CreateFindingOptions{
		AssetID: assetId,
		Type:    repository.FindingType(requestBody.Type),
		Data:    requestBody.Data,
	})

	if err != nil {
		return WrapError(err)
	}

	if err = RespondOneCreated(w, r, finding); err != nil {
		return WrapError(err)
	}

	return nil
}

func (h AssetHandler) HandleListAssetHistory(w http.ResponseWriter, r *http.Request) error {
	assetId := r.PathValue("id")

	results, err := h.scanService.ListAssetHistory(r.Context(), assetId)
	if err != nil {
		return WrapError(err)
	}

	if err = RespondMany(w, r, results); err != nil {
		return WrapError(err)
	}
	return nil
}
