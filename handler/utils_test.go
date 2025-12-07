package handler_test

import (
	"cortex/handler"
	"cortex/test"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotFound(t *testing.T) {
	err := handler.NotFound("type", "id")
	assert.Equal(t, err.StatusCode, http.StatusNotFound)
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
		return handler.NotFound("test", "1")
	}
	expectedResponse := handler.ErrorResponse{
		ID:         "",
		APIVersion: 1,
		Error: handler.ErrorResponseValue{
			Code:    http.StatusNotFound,
			Message: "API error: test with id 1 not found",
			Errors:  make([]handler.ErrorResponseStack, 0),
		},
	}

	apiHandler := handler.Make(testHandler)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	apiHandler.ServeHTTP(rr, req)

	assert.Equal(t, rr.Code, http.StatusNotFound)

	test.AssertJSON(t, rr.Body.String(), expectedResponse)
}
