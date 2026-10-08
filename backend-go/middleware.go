package main

import (
	"encoding/json"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// guardMiddleware closes the cross-site request hole that a LAN deployment
// otherwise leaves open.
//
// The server listens on every interface and the API has no authentication, so
// any web page an operator visits could reach it. CORS only restricts reading
// responses: a cross-origin request that uses one of the "simple" content
// types is sent without a preflight and the write still happens. Rejecting
// those content types therefore blocks the attack at the point that matters,
// while the app itself is unaffected because it always sends
// application/json from the same origin.
//
// Note that the API stays reachable from other machines on the LAN by design;
// this only stops browsers from being used as a proxy for it.
func guardMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")

		if r.Method == http.MethodOptions {
			// No CORS headers are returned, so a browser preflight fails and
			// cross-origin JSON calls are blocked.
			w.WriteHeader(http.StatusNoContent)
			return
		}

		switch r.Method {
		case http.MethodGet, http.MethodHead:
		default:
			if ct := r.Header.Get("Content-Type"); ct != "" {
				mediaType, _, err := mime.ParseMediaType(ct)
				if err != nil {
					writeError(w, 400, "invalid Content-Type")
					return
				}
				switch mediaType {
				case "application/x-www-form-urlencoded", "text/plain":
					writeError(w, 415, "Content-Type must be application/json")
					return
				case "multipart/form-data":
					// Only the CSV import legitimately posts a form body.
					if r.URL.Path != "/api/data/import" {
						writeError(w, 415, "Content-Type must be application/json")
						return
					}
				}
			}
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, detail any) {
	if status >= 500 {
		log.Printf("ERROR %d: %v", status, detail)
		detail = "Internal server error"
	}
	writeJSON(w, status, map[string]any{"detail": detail})
}

type spaHandler struct {
	distDir string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/") {
		http.NotFound(w, r)
		return
	}

	rel := strings.TrimPrefix(r.URL.Path, "/")
	if rel == "" {
		http.ServeFile(w, r, filepath.Join(h.distDir, "index.html"))
		return
	}

	target := filepath.Join(h.distDir, filepath.FromSlash(rel))
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		http.ServeFile(w, r, target)
		return
	}

	// Vue hash router: unknown paths fall back to index.html so deep links
	// like /#/checkout work from any entry point.
	http.ServeFile(w, r, filepath.Join(h.distDir, "index.html"))
}
