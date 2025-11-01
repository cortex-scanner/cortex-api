package middleware

import (
	"net/http"
)

func SecurityHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// prevent browsers from interpreting content types
			w.Header().Add("X-Content-Type-Options", "nosniff")
			// deny framing
			w.Header().Add("X-Frame-Options", "DENY")
			// CSP
			w.Header().Add("Content-Security-Policy",
				"default-src 'self'; script-src 'none'; object-src 'none'; style-src 'none'; img-src 'none'; "+
					"frame-ancestors 'none'; base-uri 'none'; form-action 'none';")
			// prevent access from flash and adobe
			w.Header().Add("X-Permitted-Cross-Domain-Policies", "none")
			// disable referrers
			w.Header().Add("Referrer-Policy", "no-referrer")

			next.ServeHTTP(w, r)
		})
	}
}
