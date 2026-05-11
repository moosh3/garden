package api

import (
	"crypto/subtle"
	"net/http"
	"os"
	"strings"
)

func authorize(r *http.Request) bool {
	expected := os.Getenv("API_BEARER_TOKEN")
	if expected == "" {
		return false
	}

	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}

	actual := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1
}
