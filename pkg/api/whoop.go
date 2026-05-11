package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	whoopTokenKey   = "whoop:oauth_tokens"
	whoopCacheKey   = "whoop:readiness_summary"
	whoopCacheTTL   = 12 * time.Hour
	defaultWhoopAPI = "https://api.prod.whoop.com"
)

type ConfigError string

func (e ConfigError) Error() string { return string(e) }

func IsConfigError(err error) bool {
	var configErr ConfigError
	return errors.As(err, &configErr)
}

type WhoopService struct {
	kv         KVStore
	httpClient *http.Client
	baseURL    string
	clientID   string
	secret     string
	now        func() time.Time
}

type WhoopSummary struct {
	Source      string         `json:"source"`
	FetchedAt   time.Time      `json:"fetched_at"`
	CachedUntil time.Time      `json:"cached_until"`
	Readiness   WhoopReadiness `json:"readiness"`
}

type WhoopReadiness struct {
	Score            int      `json:"score"`
	State            string   `json:"state"`
	RestingHeartRate *int     `json:"resting_heart_rate,omitempty"`
	HRVRMSSDMilli    *float64 `json:"hrv_rmssd_milli,omitempty"`
	SPO2Percentage   *float64 `json:"spo2_percentage,omitempty"`
	SkinTempCelsius  *float64 `json:"skin_temp_celsius,omitempty"`
}

type whoopTokens struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type whoopRecoveryResponse struct {
	Records []struct {
		ScoreState string `json:"score_state"`
		Score      struct {
			RecoveryScore   int      `json:"recovery_score"`
			RestingHR       *int     `json:"resting_heart_rate"`
			HRVRMSSD        *float64 `json:"hrv_rmssd_milli"`
			SPO2Percentage  *float64 `json:"spo2_percentage"`
			SkinTempCelsius *float64 `json:"skin_temp_celsius"`
		} `json:"score"`
	} `json:"records"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func NewWhoopServiceFromEnv() *WhoopService {
	kv, _ := NewKVFromEnv()
	return &WhoopService{
		kv:         kv,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    envOrDefault("WHOOP_API_BASE_URL", defaultWhoopAPI),
		clientID:   os.Getenv("WHOOP_CLIENT_ID"),
		secret:     os.Getenv("WHOOP_CLIENT_SECRET"),
		now:        func() time.Time { return time.Now().UTC() },
	}
}

func NewWhoopService(kv KVStore, httpClient *http.Client, baseURL, clientID, secret string, now func() time.Time) *WhoopService {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	return &WhoopService{
		kv:         kv,
		httpClient: httpClient,
		baseURL:    baseURL,
		clientID:   clientID,
		secret:     secret,
		now:        now,
	}
}

func (s *WhoopService) Readiness(ctx context.Context) (WhoopSummary, error) {
	if s.kv == nil {
		return WhoopSummary{}, ConfigError("Vercel KV is not configured")
	}

	var cached WhoopSummary
	cacheErr := s.kv.Get(ctx, whoopCacheKey, &cached)
	if cacheErr == nil && s.now().Before(cached.CachedUntil) {
		return cached, nil
	}

	summary, err := s.fetchReadiness(ctx)
	if err != nil {
		if cacheErr == nil {
			return cached, nil
		}
		return WhoopSummary{}, err
	}

	if err := s.kv.Set(ctx, whoopCacheKey, summary, whoopCacheTTL); err != nil {
		return WhoopSummary{}, err
	}
	return summary, nil
}

func (s *WhoopService) fetchReadiness(ctx context.Context) (WhoopSummary, error) {
	token, err := s.validAccessToken(ctx)
	if err != nil {
		return WhoopSummary{}, err
	}

	endpoint, err := url.JoinPath(s.baseURL, "/developer/v2/recovery")
	if err != nil {
		return WhoopSummary{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?limit=1", nil)
	if err != nil {
		return WhoopSummary{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return WhoopSummary{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return WhoopSummary{}, fmt.Errorf("whoop recovery request failed with status %d", resp.StatusCode)
	}

	var recovery whoopRecoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&recovery); err != nil {
		return WhoopSummary{}, err
	}
	if len(recovery.Records) == 0 {
		return WhoopSummary{}, errors.New("whoop returned no recovery records")
	}

	record := recovery.Records[0]
	now := s.now()
	return WhoopSummary{
		Source:      "whoop",
		FetchedAt:   now,
		CachedUntil: now.Add(whoopCacheTTL),
		Readiness: WhoopReadiness{
			Score:            record.Score.RecoveryScore,
			State:            record.ScoreState,
			RestingHeartRate: record.Score.RestingHR,
			HRVRMSSDMilli:    record.Score.HRVRMSSD,
			SPO2Percentage:   record.Score.SPO2Percentage,
			SkinTempCelsius:  record.Score.SkinTempCelsius,
		},
	}, nil
}

func (s *WhoopService) validAccessToken(ctx context.Context) (string, error) {
	if s.clientID == "" || s.secret == "" {
		return "", ConfigError("WHOOP_CLIENT_ID and WHOOP_CLIENT_SECRET are required")
	}

	var tokens whoopTokens
	if err := s.kv.Get(ctx, whoopTokenKey, &tokens); err != nil {
		return "", fmt.Errorf("WHOOP token state is missing: %w", err)
	}
	if tokens.AccessToken != "" && s.now().Add(2*time.Minute).Before(tokens.ExpiresAt) {
		return tokens.AccessToken, nil
	}
	if tokens.RefreshToken == "" {
		return "", errors.New("WHOOP refresh token is missing")
	}

	refreshed, err := s.refreshTokens(ctx, tokens.RefreshToken)
	if err != nil {
		return "", err
	}
	if err := s.kv.Set(ctx, whoopTokenKey, refreshed, 0); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

func (s *WhoopService) refreshTokens(ctx context.Context, refreshToken string) (whoopTokens, error) {
	endpoint, err := url.JoinPath(s.baseURL, "/oauth/oauth2/token")
	if err != nil {
		return whoopTokens{}, err
	}

	payload := url.Values{}
	payload.Set("grant_type", "refresh_token")
	payload.Set("refresh_token", refreshToken)
	payload.Set("client_id", s.clientID)
	payload.Set("client_secret", s.secret)
	payload.Set("scope", "offline read:recovery")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(payload.Encode()))
	if err != nil {
		return whoopTokens{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return whoopTokens{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return whoopTokens{}, fmt.Errorf("WHOOP token refresh failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return whoopTokens{}, err
	}
	if tr.AccessToken == "" || tr.RefreshToken == "" {
		return whoopTokens{}, errors.New("WHOOP token refresh response was missing tokens")
	}

	return whoopTokensFromResponse(tr, s.now()), nil
}

func whoopTokensFromResponse(tr tokenResponse, now time.Time) whoopTokens {
	tokenType := tr.TokenType
	if tokenType == "" {
		tokenType = "bearer"
	}
	return whoopTokens{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		TokenType:    tokenType,
		Scope:        tr.Scope,
		ExpiresAt:    now.Add(time.Duration(tr.ExpiresIn) * time.Second),
	}
}

func StoreWhoopTokens(ctx context.Context, kv KVStore, accessToken, refreshToken, tokenType, scope string, expiresIn int) error {
	if kv == nil {
		return ConfigError("Vercel KV is not configured")
	}
	if accessToken == "" || refreshToken == "" {
		return errors.New("access token and refresh token are required")
	}
	return kv.Set(ctx, whoopTokenKey, whoopTokensFromResponse(tokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    tokenType,
		Scope:        scope,
		ExpiresIn:    expiresIn,
	}, time.Now().UTC()), 0)
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
