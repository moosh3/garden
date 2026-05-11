package handler

import (
	"net/http"

	"github.com/moosh3/garden/pkg/api"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	api.WhoopHandler(w, r)
}
