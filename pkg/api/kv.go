package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var ErrKVKeyNotFound = errors.New("kv key not found")

type KVStore interface {
	Get(ctx context.Context, key string, dest any) error
	Set(ctx context.Context, key string, value any, ttl time.Duration) error
}

type UpstashKV struct {
	url        string
	token      string
	httpClient *http.Client
}

func NewKVFromEnv() (*UpstashKV, error) {
	url := strings.TrimRight(os.Getenv("KV_REST_API_URL"), "/")
	token := os.Getenv("KV_REST_API_TOKEN")
	if url == "" || token == "" {
		return nil, ConfigError("KV_REST_API_URL and KV_REST_API_TOKEN are required")
	}
	return &UpstashKV{
		url:        url,
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (kv *UpstashKV) Get(ctx context.Context, key string, dest any) error {
	var response struct {
		Result *string `json:"result"`
		Error  string  `json:"error"`
	}
	if err := kv.command(ctx, &response, "GET", key); err != nil {
		return err
	}
	if response.Error != "" {
		return fmt.Errorf("kv get %q: %s", key, response.Error)
	}
	if response.Result == nil {
		return ErrKVKeyNotFound
	}
	return json.Unmarshal([]byte(*response.Result), dest)
}

func (kv *UpstashKV) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}

	args := []any{"SET", key, string(encoded)}
	if ttl > 0 {
		args = append(args, "EX", int(ttl.Seconds()))
	}

	var response struct {
		Result string `json:"result"`
		Error  string `json:"error"`
	}
	if err := kv.command(ctx, &response, args...); err != nil {
		return err
	}
	if response.Error != "" {
		return fmt.Errorf("kv set %q: %s", key, response.Error)
	}
	return nil
}

func (kv *UpstashKV) command(ctx context.Context, dest any, args ...any) error {
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, kv.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+kv.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := kv.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("kv command failed with status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}
