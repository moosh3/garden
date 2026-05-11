package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type readinessProvider interface {
	Readiness(ctx context.Context) (WhoopSummary, error)
}

var newWhoopService = func() readinessProvider {
	return NewWhoopServiceFromEnv()
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET for this endpoint.")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"service": "garden-api",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func WhoopHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET for this endpoint.")
		return
	}
	if !authorize(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid bearer token.")
		return
	}

	summary, err := newWhoopService().Readiness(r.Context())
	if err != nil {
		writeError(w, statusForError(err), "whoop_unavailable", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, summary)
}

func DataHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Use GET for this endpoint.")
		return
	}
	if !authorize(r) {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid bearer token.")
		return
	}

	source := r.URL.Query().Get("source")
	switch source {
	case "":
		writeError(w, http.StatusBadRequest, "missing_source", "Provide a source query parameter.")
	case "whoop":
		summary, err := newWhoopService().Readiness(r.Context())
		if err != nil {
			writeError(w, statusForError(err), "whoop_unavailable", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, summary)
	default:
		writeError(w, http.StatusBadRequest, "unsupported_source", "Unsupported source: "+source+".")
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func statusForError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if IsConfigError(err) {
		return http.StatusServiceUnavailable
	}
	return http.StatusBadGateway
}
