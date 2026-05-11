package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type fakeReadinessProvider struct {
	summary WhoopSummary
	err     error
}

func (p fakeReadinessProvider) Readiness(ctx context.Context) (WhoopSummary, error) {
	return p.summary, p.err
}

func TestAuthMiddleware(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", "secret")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "wrong token", authHeader: "Bearer nope", wantStatus: http.StatusUnauthorized},
		{name: "valid token", authHeader: "Bearer secret", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/data", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rr := httptest.NewRecorder()

			DataHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestDataHandlerSourceRouter(t *testing.T) {
	t.Setenv("API_BEARER_TOKEN", "secret")
	originalNewWhoopService := newWhoopService
	defer func() { newWhoopService = originalNewWhoopService }()
	newWhoopService = func() readinessProvider {
		return fakeReadinessProvider{summary: WhoopSummary{
			Source:      "whoop",
			FetchedAt:   time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC),
			CachedUntil: time.Date(2026, 5, 12, 0, 0, 0, 0, time.UTC),
			Readiness:   WhoopReadiness{Score: 87, State: "SCORED"},
		}}
	}

	tests := []struct {
		name       string
		target     string
		wantStatus int
	}{
		{name: "whoop source", target: "/api/v1/data?source=whoop", wantStatus: http.StatusOK},
		{name: "missing source", target: "/api/v1/data", wantStatus: http.StatusBadRequest},
		{name: "unsupported source", target: "/api/v1/data?source=strava", wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.target, nil)
			req.Header.Set("Authorization", "Bearer secret")
			rr := httptest.NewRecorder()

			DataHandler(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rr.Code, tt.wantStatus)
			}
		})
	}
}

func TestHealthHandlerDoesNotRequireBearerToken(t *testing.T) {
	_ = os.Unsetenv("API_BEARER_TOKEN")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rr := httptest.NewRecorder()

	HealthHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}
