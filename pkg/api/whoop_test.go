package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type memoryKV struct {
	mu     sync.Mutex
	values map[string][]byte
	setErr error
	getErr error
}

func newMemoryKV() *memoryKV {
	return &memoryKV{values: map[string][]byte{}}
}

func (kv *memoryKV) Get(ctx context.Context, key string, dest any) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.getErr != nil {
		return kv.getErr
	}
	value, ok := kv.values[key]
	if !ok {
		return ErrKVKeyNotFound
	}
	return json.Unmarshal(value, dest)
}

func (kv *memoryKV) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	if kv.setErr != nil {
		return kv.setErr
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	kv.values[key] = encoded
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testHTTPClient(fn roundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestWhoopReadinessUsesFreshCache(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	kv := newMemoryKV()
	cached := WhoopSummary{
		Source:      "whoop",
		FetchedAt:   now.Add(-time.Hour),
		CachedUntil: now.Add(time.Hour),
		Readiness:   WhoopReadiness{Score: 91, State: "SCORED"},
	}
	if err := kv.Set(context.Background(), whoopCacheKey, cached, 0); err != nil {
		t.Fatal(err)
	}

	called := false
	client := testHTTPClient(func(req *http.Request) (*http.Response, error) {
		called = true
		return jsonResponse(http.StatusInternalServerError, `{}`), nil
	})

	service := NewWhoopService(kv, client, "http://whoop.test", "client", "secret", func() time.Time { return now })
	got, err := service.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Readiness.Score != 91 {
		t.Fatalf("score = %d, want 91", got.Readiness.Score)
	}
	if called {
		t.Fatal("WHOOP API was called for a fresh cache hit")
	}
}

func TestWhoopReadinessRefreshesStaleCache(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	kv := newMemoryKV()
	_ = kv.Set(context.Background(), whoopCacheKey, WhoopSummary{
		Source:      "whoop",
		FetchedAt:   now.Add(-24 * time.Hour),
		CachedUntil: now.Add(-time.Hour),
		Readiness:   WhoopReadiness{Score: 1, State: "STALE"},
	}, 0)
	_ = kv.Set(context.Background(), whoopTokenKey, whoopTokens{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}, 0)

	client := testHTTPClient(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path != "/developer/v2/recovery" {
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer access" {
			t.Fatalf("authorization = %q, want Bearer access", got)
		}
		return jsonResponse(http.StatusOK, `{"records":[{"score_state":"SCORED","score":{"recovery_score":87,"resting_heart_rate":52,"hrv_rmssd_milli":64.2,"spo2_percentage":98.1,"skin_temp_celsius":33.4}}]}`), nil
	})

	service := NewWhoopService(kv, client, "http://whoop.test", "client", "secret", func() time.Time { return now })
	got, err := service.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Readiness.Score != 87 {
		t.Fatalf("score = %d, want 87", got.Readiness.Score)
	}
	if !got.CachedUntil.Equal(now.Add(whoopCacheTTL)) {
		t.Fatalf("cached until = %s, want %s", got.CachedUntil, now.Add(whoopCacheTTL))
	}
}

func TestWhoopReadinessFallsBackToStaleCacheOnWhoopError(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	kv := newMemoryKV()
	_ = kv.Set(context.Background(), whoopCacheKey, WhoopSummary{
		Source:      "whoop",
		FetchedAt:   now.Add(-24 * time.Hour),
		CachedUntil: now.Add(-time.Hour),
		Readiness:   WhoopReadiness{Score: 55, State: "SCORED"},
	}, 0)
	_ = kv.Set(context.Background(), whoopTokenKey, whoopTokens{
		AccessToken:  "access",
		RefreshToken: "refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(time.Hour),
	}, 0)

	client := testHTTPClient(func(req *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusBadGateway, `nope`), nil
	})

	service := NewWhoopService(kv, client, "http://whoop.test", "client", "secret", func() time.Time { return now })
	got, err := service.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.Readiness.Score != 55 {
		t.Fatalf("score = %d, want stale score 55", got.Readiness.Score)
	}
}

func TestWhoopTokenRefreshPersistsRotatedTokens(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	kv := newMemoryKV()
	_ = kv.Set(context.Background(), whoopTokenKey, whoopTokens{
		AccessToken:  "expired",
		RefreshToken: "old-refresh",
		TokenType:    "bearer",
		ExpiresAt:    now.Add(-time.Minute),
	}, 0)

	client := testHTTPClient(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/oauth/oauth2/token":
			return jsonResponse(http.StatusOK, `{"access_token":"new-access","expires_in":3600,"refresh_token":"new-refresh","scope":"offline read:recovery","token_type":"bearer"}`), nil
		case "/developer/v2/recovery":
			if got := req.Header.Get("Authorization"); got != "Bearer new-access" {
				t.Fatalf("authorization = %q, want Bearer new-access", got)
			}
			return jsonResponse(http.StatusOK, `{"records":[{"score_state":"SCORED","score":{"recovery_score":73}}]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
		}
		return jsonResponse(http.StatusInternalServerError, `{}`), nil
	})

	service := NewWhoopService(kv, client, "http://whoop.test", "client", "secret", func() time.Time { return now })
	_, err := service.Readiness(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var stored whoopTokens
	if err := kv.Get(context.Background(), whoopTokenKey, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.AccessToken != "new-access" || stored.RefreshToken != "new-refresh" {
		t.Fatalf("stored tokens = %#v", stored)
	}
}

func TestWhoopReadinessReturnsErrorWithoutCache(t *testing.T) {
	now := time.Date(2026, 5, 11, 12, 0, 0, 0, time.UTC)
	kv := newMemoryKV()
	kv.getErr = errors.New("kv unavailable")

	service := NewWhoopService(kv, nil, "http://example.test", "client", "secret", func() time.Time { return now })
	_, err := service.Readiness(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}
