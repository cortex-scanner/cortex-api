package middleware_test

import (
	"cortex/context"
	"cortex/middleware"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestID(t *testing.T) {
	t.Run("uses existing request ID from header", func(t *testing.T) {
		existingID := "test-request-id"
		var capturedID string

		reqID := middleware.NewRequestIDMiddleware(func() string {
			t.Error("generator shouldn't be called when ID exists")
			return ""
		})

		testHandler := reqID.OnRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = r.Context().Value(context.KeyRequestID).(string)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Request-ID", existingID)
		rr := httptest.NewRecorder()

		testHandler.ServeHTTP(rr, req)

		assert.Equal(t, existingID, capturedID)
	})

	t.Run("generates new request ID when header is missing", func(t *testing.T) {
		// Setup
		generatedID := "generated-id"
		var capturedID string

		reqID := middleware.NewRequestIDMiddleware(func() string {
			return generatedID
		})

		testHandler := reqID.OnRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = r.Context().Value(context.KeyRequestID).(string)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		// Act
		testHandler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, generatedID, capturedID)
	})

	t.Run("uses custom header name", func(t *testing.T) {
		// Setup
		existingID := "test-request-id"
		customHeader := "Custom-Request-ID"
		var capturedID string

		reqID := middleware.NewRequestIDMiddleware(func() string {
			t.Error("generator shouldn't be called when ID exists")
			return ""
		})
		reqID.RequestIDHeader = customHeader

		testHandler := reqID.OnRequest(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = r.Context().Value(context.KeyRequestID).(string)
		}))

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(customHeader, existingID)
		rr := httptest.NewRecorder()

		// Act
		testHandler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, existingID, capturedID)
	})
}

func TestNewUUIDv4RequestID(t *testing.T) {
	uuidMw := middleware.NewUUIDv4RequestIDMiddleWare()
	id := uuidMw.RequestIDGenerator()

	assert.NotEmpty(t, id)
	assert.Regexp(t, "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$", id)
}
