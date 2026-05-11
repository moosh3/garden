package handler

import (
	"net/http"

	"github.com/moosh3/garden/internal/api"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	api.HealthHandler(w, r)
}
