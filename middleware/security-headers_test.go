package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	rr := httptest.NewRecorder()

	mockHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	middleware := SecurityHeaders()
	handler := middleware(mockHandler)
	handler.ServeHTTP(rr, req)

	// Check that the headers are present
	assert.NotEmpty(t, rr.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, rr.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, rr.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, rr.Header().Get("X-Permitted-Cross-Domain-Policies"))
	assert.NotEmpty(t, rr.Header().Get("Referrer-Policy"))
}
