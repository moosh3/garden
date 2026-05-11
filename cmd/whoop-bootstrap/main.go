package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/moosh3/garden/pkg/api"
)

const (
	defaultRedirectURL = "http://localhost:8787/oauth/callback"
	whoopAuthURL       = "https://api.prod.whoop.com/oauth/oauth2/auth"
	whoopTokenURL      = "https://api.prod.whoop.com/oauth/oauth2/token"
	whoopScope         = "offline read:recovery"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

func main() {
	loadDotEnvLocal()

	clientID := os.Getenv("WHOOP_CLIENT_ID")
	secret := os.Getenv("WHOOP_CLIENT_SECRET")
	redirectURL := envOrDefault("WHOOP_REDIRECT_URL", defaultRedirectURL)
	if clientID == "" || secret == "" {
		log.Fatal("WHOOP_CLIENT_ID and WHOOP_CLIENT_SECRET are required")
	}

	kv, err := api.NewKVFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	state := randomState()
	done := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "invalid OAuth state", http.StatusBadRequest)
			done <- fmt.Errorf("invalid OAuth state")
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing OAuth code", http.StatusBadRequest)
			done <- fmt.Errorf("missing OAuth code")
			return
		}

		tokens, err := exchangeCode(r.Context(), clientID, secret, redirectURL, code)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			done <- err
			return
		}
		err = api.StoreWhoopTokens(r.Context(), kv, tokens.AccessToken, tokens.RefreshToken, tokens.TokenType, tokens.Scope, tokens.ExpiresIn)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			done <- err
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("WHOOP OAuth tokens stored. You can close this tab.\n"))
		done <- nil
	})

	server := &http.Server{Addr: ":8787", Handler: mux}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			done <- err
		}
	}()

	authURL, err := buildAuthURL(clientID, redirectURL, state)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Open this URL to connect WHOOP:")
	fmt.Println(authURL)

	err = <-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Stored WHOOP OAuth tokens in Vercel KV.")
}

func buildAuthURL(clientID, redirectURL, state string) (string, error) {
	u, err := url.Parse(whoopAuthURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	query.Set("client_id", clientID)
	query.Set("redirect_uri", redirectURL)
	query.Set("response_type", "code")
	query.Set("scope", whoopScope)
	query.Set("state", state)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func exchangeCode(ctx context.Context, clientID, secret, redirectURL, code string) (tokenResponse, error) {
	payload := url.Values{}
	payload.Set("grant_type", "authorization_code")
	payload.Set("code", code)
	payload.Set("client_id", clientID)
	payload.Set("client_secret", secret)
	payload.Set("redirect_uri", redirectURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, whoopTokenURL, strings.NewReader(payload.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return tokenResponse{}, fmt.Errorf("WHOOP token exchange failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokens tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return tokenResponse{}, err
	}
	if tokens.AccessToken == "" || tokens.RefreshToken == "" {
		return tokenResponse{}, fmt.Errorf("WHOOP token exchange response was missing tokens")
	}
	return tokens, nil
}

func randomState() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func loadDotEnvLocal() {
	data, err := os.ReadFile(".env.local")
	if err != nil {
		return
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
