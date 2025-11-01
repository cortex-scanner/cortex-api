package middleware

import (
	"context"
	cortexContext "cortex/context"
	"net/http"

	"github.com/google/uuid"
)

type RequestIDMiddleware struct {
	RequestIDHeader    string
	RequestIDGenerator func() string
}

func NewRequestIDMiddleware(generatorFunc func() string) *RequestIDMiddleware {
	return &RequestIDMiddleware{
		RequestIDGenerator: generatorFunc,
		RequestIDHeader:    "X-Request-ID",
	}
}

func NewUUIDv4RequestIDMiddleWare() *RequestIDMiddleware {
	return NewRequestIDMiddleware(func() string {
		return uuid.New().String()
	})
}

func (h *RequestIDMiddleware) OnRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(h.RequestIDHeader)

		if requestID == "" {
			requestID = h.RequestIDGenerator()
		}

		ctx := context.WithValue(r.Context(), cortexContext.KeyRequestID, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
