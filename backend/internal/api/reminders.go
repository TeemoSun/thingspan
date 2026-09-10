package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"thingspan/internal/models"
	"thingspan/internal/services"
)

type RemindersHandler struct {
	db      *sql.DB
	scanner *services.ReminderScanner
}

func NewRemindersHandler(db *sql.DB, scanner *services.ReminderScanner) *RemindersHandler {
	return &RemindersHandler{db: db, scanner: scanner}
}

func (h *RemindersHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT r.id, r.asset_id, COALESCE(a.name, ''), r.target_date, r.lead_days, r.sent_at, r.sent, r.dismissed
		FROM reminder_logs r
		LEFT JOIN assets a ON r.asset_id = a.id
		ORDER BY r.sent_at DESC
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "获取提醒记录失败")
		return
	}
	defer rows.Close()

	var list []models.ReminderOut
	for rows.Next() {
		var item models.ReminderOut
		var sentAtStr string
		if err := rows.Scan(
			&item.ID, &item.AssetID, &item.AssetName, &item.TargetDate, &item.LeadDays,
			&sentAtStr, &item.Sent, &item.Dismissed,
		); err != nil {
			WriteError(w, http.StatusInternalServerError, "扫描提醒记录失败")
			return
		}
		t, _ := time.Parse("2006-01-02 15:04:05", strings.Split(sentAtStr, ".")[0])
		item.SentAt = t.Format("2006-01-02T15:04:05")
		list = append(list, item)
	}

	if list == nil {
		list = []models.ReminderOut{}
	}
	WriteJSON(w, http.StatusOK, list)
}

func (h *RemindersHandler) Dismiss(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	reminderID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "无效的提醒 ID")
		return
	}

	item, err := h.scanner.DismissReminder(reminderID)
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "提醒记录不存在")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "忽略提醒失败")
		return
	}

	WriteJSON(w, http.StatusOK, models.ReminderOut{
		ID:         item.ID,
		AssetID:    item.AssetID,
		AssetName:  item.AssetName,
		TargetDate: item.TargetDate,
		LeadDays:   item.LeadDays,
		SentAt:     item.SentAt.Format("2006-01-02T15:04:05"),
		Sent:       item.Sent,
		Dismissed:  item.Dismissed,
	})
}

