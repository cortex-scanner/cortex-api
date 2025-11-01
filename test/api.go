package test

import (
	"bytes"
	"cortex/handler"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type Result struct {
	t     *testing.T
	RR    *httptest.ResponseRecorder
	Error error
}

func (r *Result) ExpectNoError() *Result {
	assert.NoError(r.t, r.Error)
	return r
}

func (r *Result) ExpectAPIError(statusCode int) *Result {
	var apiError handler.APIError
	assert.ErrorAs(r.t, r.Error, &apiError)
	assert.Equal(r.t, statusCode, apiError.StatusCode)
	return r
}

func (r *Result) ExpectStatusCode(statusCode int) *Result {
	assert.Equal(r.t, statusCode, r.RR.Code)
	return r
}

func (r *Result) ExpectSingleDataResponse(expectedBody string) *Result {
	assert.Equal(r.t, expectedBody, r.RR.Body.String())
	return r
}

func (r *Result) ExpectSingleDataResponseJSON(expectedBody handler.SingleDataResponse[any]) *Result {
	AssertJSON(r.t, r.RR.Body.String(), expectedBody)
	return r
}

type APIRunner struct {
	handlerFunc handler.APIFunc
	req         *http.Request
	rr          *httptest.ResponseRecorder
}

func NewTestRunner(handlerFunc handler.APIFunc) *APIRunner {
	return &APIRunner{
		handlerFunc: handlerFunc,
		req:         httptest.NewRequest(http.MethodGet, "/", nil),
		rr:          httptest.NewRecorder(),
	}
}

func (r *APIRunner) WithBodyString(body string) *APIRunner {
	r.req.Body = io.NopCloser(bytes.NewBufferString(body))
	return r
}

func (r *APIRunner) WithBody(body any) *APIRunner {
	jsonData, _ := json.Marshal(body)
	r.req.Body = io.NopCloser(bytes.NewBuffer(jsonData))
	return r
}

func (r *APIRunner) WithHeader(key, value string) *APIRunner {
	r.req.Header.Set(key, value)
	return r
}

func (r *APIRunner) WithPath(key, value string) *APIRunner {
	r.req.SetPathValue(key, value)
	return r
}

func (r *APIRunner) Run(t *testing.T) *Result {
	err := r.handlerFunc(r.rr, r.req)
	return &Result{
		t:     t,
		RR:    r.rr,
		Error: err,
	}
}
