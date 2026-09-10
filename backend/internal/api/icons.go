package api

import (
	_ "embed"
	"net/http"
)

//go:embed icon_names.json
var iconNamesJSON []byte

type IconsHandler struct{}

func NewIconsHandler() *IconsHandler {
	return &IconsHandler{}
}

func (h *IconsHandler) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(iconNamesJSON)
}

