package handler_test

import (
	"cortex/handler"
	"cortex/test"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	err := handler.NotFound("type", "id")
	assert.Equal(t, err.StatusCode, http.StatusNotFound)
}

func TestInvalidRequest(t *testing.T) {
	err := handler.InvalidRequest("message", nil)
	assert.Equal(t, err.StatusCode, http.StatusBadRequest)
}

func TestOtherError(t *testing.T) {
	err := handler.OtherError(errors.New("test"))
	assert.Equal(t, err.StatusCode, http.StatusInternalServerError)
}

func TestRespondError(t *testing.T) {
	testErr := errors.New("test")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	expectedResponse := handler.ErrorResponse{
		ID:         "",
		APIVersion: 1,
		Error: handler.ErrorResponseValue{
			Code:    http.StatusBadRequest,
			Message: "test",
			Errors:  make([]handler.ErrorResponseStack, 0),
		},
	}

	handler.RespondError(rr, req, http.StatusBadRequest, testErr)

	assert.Equal(t, rr.Code, http.StatusBadRequest)
	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestRespondOneSimple(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	err := handler.RespondOne(rr, req, "test")

	expectedResponse := handler.SingleDataResponse[string]{
		ID:         "",
		APIVersion: 1,
		Data:       "test",
	}

	assert.Nil(t, err)
	assert.Equal(t, rr.Code, http.StatusOK)
	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestRespondOneStruct(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	type TestStruct struct {
		Test string `json:"test"`
	}

	data := TestStruct{
		Test: "test",
	}

	expectedResponse := handler.SingleDataResponse[TestStruct]{
		ID:         "",
		APIVersion: 1,
		Data:       data,
	}

	err := handler.RespondOne(rr, req, data)
	assert.Nil(t, err)
	assert.Equal(t, rr.Code, http.StatusOK)
	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestRespondOneCreated(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	err := handler.RespondOneCreated(rr, req, "test")

	expectedResponse := handler.SingleDataResponse[string]{
		ID:         "",
		APIVersion: 1,
		Data:       "test",
	}

	assert.Nil(t, err)
	assert.Equal(t, rr.Code, http.StatusCreated)
	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestRespondMany(t *testing.T) {
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	data := []string{"test1", "test2"}
	err := handler.RespondMany(rr, req, data)

	expectedResponse := handler.ArrayDataResponse[string]{
		ID:         "",
		APIVersion: 1,
		Data: handler.APIComponentArray[string]{
			TotalItems:       2,
			Items:            data,
			StartIndex:       0,
			CurrentItemCount: 2,
		},
	}

	assert.Nil(t, err)
	assert.Equal(t, rr.Code, http.StatusOK)
	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestMakeGenericError(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) error {
		return errors.New("test")
	}

	expectedResponse := handler.ErrorResponse{
		ID:         "",
		APIVersion: 1,
		Error: handler.ErrorResponseValue{
			Code:    http.StatusInternalServerError,
			Message: "test",
			Errors:  make([]handler.ErrorResponseStack, 0),
		},
	}

	apiHandler := handler.Make(testHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	apiHandler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusInternalServerError)
	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestMakeAPIError(t *testing.T) {
	testHandler := func(w http.ResponseWriter, r *http.Request) error {
		return handler.InvalidRequest("test", errors.New("test"))
	}
	expectedResponse := handler.ErrorResponse{
		ID:         "",
		APIVersion: 1,
		Error: handler.ErrorResponseValue{
			Code:    http.StatusBadRequest,
			Message: "API error: test",
			Errors:  make([]handler.ErrorResponseStack, 0),
		},
	}

	apiHandler := handler.Make(testHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	apiHandler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusBadRequest)

	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}

func TestParseAndValidateBodySuccess(t *testing.T) {
	type TestStruct struct {
		Required string `json:"required" validate:"required"`
	}
	structValidator := validator.New(validator.WithRequiredStructEnabled())

	expectedParsedBody := TestStruct{
		Required: "test",
	}

	var body TestStruct
	req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("{\"required\":\"test\"}"))
	err := handler.ParseAndValidateBody(&body, req, structValidator)
	assert.Nil(t, err)
	assert.Equal(t, expectedParsedBody, body)
}

func TestParseAndValidateBodyFail(t *testing.T) {
	type TestStruct struct {
		Required string `json:"required" validate:"required"`
	}
	structValidator := validator.New(validator.WithRequiredStructEnabled())

	t.Run("Parse invalid json", func(t *testing.T) {
		var body TestStruct
		req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("{invalid"))
		err := handler.ParseAndValidateBody(&body, req, structValidator)
		assert.NotNil(t, err)

		var apiErr handler.APIError
		assert.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})

	t.Run("Parse invalid struct", func(t *testing.T) {
		var body TestStruct
		req := httptest.NewRequest(http.MethodGet, "/", strings.NewReader("{}"))
		err := handler.ParseAndValidateBody(&body, req, structValidator)
		assert.NotNil(t, err)

		var apiErr handler.APIError
		assert.ErrorAs(t, err, &apiErr)
		assert.Equal(t, http.StatusBadRequest, apiErr.StatusCode)
	})
}
