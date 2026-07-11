package httptransport

import (
	"net/http"
	"strings"
)

const corsAllowedHeaders = "Accept, Authorization, Content-Type, X-Requested-With"

func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	allowed := allowedOrigins(origins)
	return func(next http.Handler) http.Handler {
		if len(allowed) == 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			allowedOrigin, ok := corsAllowedOrigin(allowed, origin)
			if ok {
				writeCORSHeaders(w.Header(), allowedOrigin)
				if isCORSPreflight(r) {
					writeCORSPreflightHeaders(w.Header(), r)
					w.WriteHeader(http.StatusNoContent)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func allowedOrigins(origins []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowed[origin] = struct{}{}
		}
	}
	return allowed
}

func corsAllowedOrigin(allowed map[string]struct{}, origin string) (string, bool) {
	if _, ok := allowed["*"]; ok {
		// Wildcard CORS is intended for local web/mobile development setups.
		return "*", true
	}
	if _, ok := allowed[origin]; ok {
		return origin, true
	}
	return "", false
}

func writeCORSHeaders(header http.Header, origin string) {
	header.Add("Vary", "Origin")
	header.Set("Access-Control-Allow-Origin", origin)
}

func isCORSPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

func writeCORSPreflightHeaders(header http.Header, r *http.Request) {
	header.Add("Vary", "Access-Control-Request-Method")
	header.Add("Vary", "Access-Control-Request-Headers")
	header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	header.Set("Access-Control-Allow-Headers", corsAllowedHeaders)
}
