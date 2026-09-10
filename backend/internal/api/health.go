package api

import (
	"database/sql"
	"net/http"
)

type HealthHandler struct {
	db *sql.DB
}

func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	// Verify table read
	var count int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&count); err != nil {
		WriteJSON(w, http.StatusInternalServerError, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *HealthHandler) APIHealth(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

