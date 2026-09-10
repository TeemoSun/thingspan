package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"thingspan/internal/config"
	"thingspan/internal/models"
)

type CategoriesHandler struct {
	db  *sql.DB
	cfg *config.Config
}

func NewCategoriesHandler(db *sql.DB, cfg *config.Config) *CategoriesHandler {
	return &CategoriesHandler{db: db, cfg: cfg}
}

func (h *CategoriesHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT c.id, c.name, c.has_warranty, c.has_expiry, c.can_sell, c.can_break, c.has_serial, c.has_model,
		       COUNT(a.id) AS assets_count
		FROM categories c
		LEFT JOIN assets a ON c.id = a.category_id
		GROUP BY c.id
		ORDER BY c.created_at ASC
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "获取类别失败")
		return
	}
	defer rows.Close()

	var list []models.CategoryOut
	for rows.Next() {
		var item models.CategoryOut
		if err := rows.Scan(
			&item.ID, &item.Name, &item.HasWarranty, &item.HasExpiry, &item.CanSell, &item.CanBreak,
			&item.HasSerial, &item.HasModel, &item.AssetsCount,
		); err != nil {
			WriteError(w, http.StatusInternalServerError, "扫描类别数据失败")
			return
		}
		list = append(list, item)
	}

	if list == nil {
		list = []models.CategoryOut{}
	}
	WriteJSON(w, http.StatusOK, list)
}

func (h *CategoriesHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body models.CategoryCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || len(body.Name) > 50 {
		WriteError(w, http.StatusBadRequest, "类别名称长度须在 1 到 50 字符之间")
		return
	}

	var existID int64
	err := h.db.QueryRow(`SELECT id FROM categories WHERE name = ?`, body.Name).Scan(&existID)
	if err == nil {
		WriteError(w, http.StatusBadRequest, "类别名称已存在")
		return
	}

	localNowStr := h.cfg.LocalNow().Format("2006-01-02 15:04:05")
	res, err := h.db.Exec(`
		INSERT INTO categories (name, has_warranty, has_expiry, can_sell, can_break, has_serial, has_model, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, body.Name, body.HasWarranty, body.HasExpiry, body.CanSell, body.CanBreak, body.HasSerial, body.HasModel, localNowStr)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "创建类别失败")
		return
	}

	id, _ := res.LastInsertId()
	WriteJSON(w, http.StatusOK, models.CategoryOut{
		ID:          id,
		Name:        body.Name,
		HasWarranty: body.HasWarranty,
		HasExpiry:   body.HasExpiry,
		CanSell:     body.CanSell,
		CanBreak:    body.CanBreak,
		HasSerial:   body.HasSerial,
		HasModel:    body.HasModel,
		AssetsCount: 0,
	})
}

func (h *CategoriesHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "无效的类别 ID")
		return
	}

	var existID int64
	err = h.db.QueryRow(`SELECT id FROM categories WHERE id = ?`, categoryID).Scan(&existID)
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "类别不存在")
		return
	}

	var body models.CategoryUpdate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || len(body.Name) > 50 {
		WriteError(w, http.StatusBadRequest, "类别名称长度须在 1 到 50 字符之间")
		return
	}

	var dupID int64
	err = h.db.QueryRow(`SELECT id FROM categories WHERE name = ? AND id != ?`, body.Name, categoryID).Scan(&dupID)
	if err == nil {
		WriteError(w, http.StatusBadRequest, "类别名称已存在")
		return
	}

	_, err = h.db.Exec(`
		UPDATE categories
		SET name = ?, has_warranty = ?, has_expiry = ?, can_sell = ?, can_break = ?, has_serial = ?, has_model = ?
		WHERE id = ?
	`, body.Name, body.HasWarranty, body.HasExpiry, body.CanSell, body.CanBreak, body.HasSerial, body.HasModel, categoryID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "更新类别失败")
		return
	}

	var assetCount int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM assets WHERE category_id = ?`, categoryID).Scan(&assetCount)

	WriteJSON(w, http.StatusOK, models.CategoryOut{
		ID:          categoryID,
		Name:        body.Name,
		HasWarranty: body.HasWarranty,
		HasExpiry:   body.HasExpiry,
		CanSell:     body.CanSell,
		CanBreak:    body.CanBreak,
		HasSerial:   body.HasSerial,
		HasModel:    body.HasModel,
		AssetsCount: assetCount,
	})
}

func (h *CategoriesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	categoryID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "无效的类别 ID")
		return
	}

	var existID int64
	err = h.db.QueryRow(`SELECT id FROM categories WHERE id = ?`, categoryID).Scan(&existID)
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "类别不存在")
		return
	}

	var assetCount int
	_ = h.db.QueryRow(`SELECT COUNT(*) FROM assets WHERE category_id = ?`, categoryID).Scan(&assetCount)
	if assetCount > 0 {
		WriteError(w, http.StatusBadRequest, "该类别下还有资产，无法删除")
		return
	}

	_, err = h.db.Exec(`DELETE FROM categories WHERE id = ?`, categoryID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "删除类别失败")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

