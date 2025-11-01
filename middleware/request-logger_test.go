package middleware_test

import (
	"bytes"
	"cortex/middleware"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestLogger(t *testing.T) {
	var logBuffer bytes.Buffer
	mockLogger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	slog.SetDefault(mockLogger)
	reqLogger := middleware.NewRequestLoggerMiddleware()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1"

	rr := httptest.NewRecorder()
	reqLogger.OnRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, logBuffer.String(), "\"status\":201")
	assert.Contains(t, logBuffer.String(), "\"method\":\"GET\"")
	assert.Contains(t, logBuffer.String(), "\"path\":\"/\"")
	assert.Contains(t, logBuffer.String(), "\"src\":\"127.0.0.1\"")
}

func TestRequestLoggerXForwardedFor(t *testing.T) {
	var logBuffer bytes.Buffer
	mockLogger := slog.New(slog.NewJSONHandler(&logBuffer, nil))
	slog.SetDefault(mockLogger)

	reqLogger := middleware.NewRequestLoggerMiddleware()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1"
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	rr := httptest.NewRecorder()
	reqLogger.OnRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	})).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, logBuffer.String(), "\"src\":\"192.168.1.1\"")
}
