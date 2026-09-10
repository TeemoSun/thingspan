package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"thingspan/internal/config"
	"thingspan/internal/models"
	"thingspan/internal/services"
)

type AssetsHandler struct {
	db  *sql.DB
	cfg *config.Config
}

func NewAssetsHandler(db *sql.DB, cfg *config.Config) *AssetsHandler {
	return &AssetsHandler{db: db, cfg: cfg}
}

func formatDBTime(s string) string {
	if s == "" {
		return ""
	}
	s = strings.Replace(s, " ", "T", 1)
	return s
}

func (h *AssetsHandler) toAssetOut(a *models.Asset, today string) models.AssetOut {
	cost := services.CalcCost(a, today)
	catName := ""
	if a.Category != nil {
		catName = a.Category.Name
	}
	return models.AssetOut{
		ID:              a.ID,
		CategoryID:      a.CategoryID,
		CategoryName:    catName,
		Name:            a.Name,
		Icon:            a.Icon,
		Brand:           a.Brand,
		Model:           a.Model,
		SerialNumber:    a.SerialNumber,
		PurchaseDate:    a.PurchaseDate,
		PurchasePrice:   a.PurchasePrice,
		WarrantyMonths:  a.WarrantyMonths,
		WarrantyEndDate: a.WarrantyEndDate,
		ExpiryDate:      a.ExpiryDate,
		Status:          a.Status,
		SaleDate:        a.SaleDate,
		SalePrice:       a.SalePrice,
		BrokenDate:      a.BrokenDate,
		Notes:           a.Notes,
		CreatedAt:       a.CreatedAt.Format("2006-01-02T15:04:05"),
		UpdatedAt:       a.UpdatedAt.Format("2006-01-02T15:04:05"),
		Cost:            cost,
	}
}

func (h *AssetsHandler) validateCategory(categoryID int64) (*models.Category, error) {
	var c models.Category
	var createdAtStr string
	err := h.db.QueryRow(`
		SELECT id, name, has_warranty, has_expiry, can_sell, can_break, has_serial, has_model, created_at
		FROM categories WHERE id = ?
	`, categoryID).Scan(
		&c.ID, &c.Name, &c.HasWarranty, &c.HasExpiry, &c.CanSell, &c.CanBreak, &c.HasSerial, &c.HasModel, &createdAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("类别不存在")
	}
	if err != nil {
		return nil, err
	}
	t, _ := time.Parse("2006-01-02 15:04:05", strings.Split(createdAtStr, ".")[0])
	c.CreatedAt = t
	return &c, nil
}

func checkStatusFields(cat *models.Category, status string, saleDate *string, salePrice *float64, brokenDate *string, checkFlags bool) error {
	if status == models.StatusSold {
		if checkFlags && !cat.CanSell {
			return fmt.Errorf("该类别未勾选「可售出」，无法标记已售出")
		}
		if saleDate == nil || *saleDate == "" || salePrice == nil {
			return fmt.Errorf("标记已售出需填写售出日期和售出价格")
		}
	}
	if status == models.StatusBroken {
		if checkFlags && !cat.CanBreak {
			return fmt.Errorf("该类别未勾选「可损坏」，无法标记已损坏")
		}
		if brokenDate == nil || *brokenDate == "" {
			return fmt.Errorf("标记已损坏需填写损坏日期")
		}
	}
	return nil
}

func (h *AssetsHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	categoryIDStr := q.Get("category_id")
	statusFilter := q.Get("status")
	search := strings.TrimSpace(q.Get("search"))
	sortBy := q.Get("sort_by")
	sortDir := q.Get("sort_dir")
	if sortDir == "" {
		sortDir = "desc"
	}

	var queryBuilder strings.Builder
	queryBuilder.WriteString(`
		SELECT a.id, a.category_id, a.name, a.icon, a.brand, a.model, a.serial_number,
		       a.purchase_date, a.purchase_price, a.warranty_months, a.warranty_end_date, a.expiry_date,
		       a.status, a.sale_date, a.sale_price, a.broken_date, a.notes, a.created_at, a.updated_at,
		       c.id, c.name, c.has_warranty, c.has_expiry, c.can_sell, c.can_break, c.has_serial, c.has_model
		FROM assets a
		JOIN categories c ON a.category_id = c.id
		WHERE 1=1
	`)

	var args []interface{}
	if categoryIDStr != "" {
		if cid, err := strconv.ParseInt(categoryIDStr, 10, 64); err == nil && cid > 0 {
			queryBuilder.WriteString(" AND a.category_id = ?")
			args = append(args, cid)
		}
	}

	if search != "" {
		like := "%" + search + "%"
		queryBuilder.WriteString(" AND (a.name LIKE ? OR a.brand LIKE ? OR a.model LIKE ? OR a.serial_number LIKE ?)")
		args = append(args, like, like, like, like)
	}

	queryBuilder.WriteString(" ORDER BY a.created_at DESC")

	rows, err := h.db.Query(queryBuilder.String(), args...)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "查询资产失败")
		return
	}
	defer rows.Close()

	today := h.cfg.TodayStr()
	localNowStr := h.cfg.LocalNow().Format("2006-01-02 15:04:05")

	var assets []*models.Asset
	for rows.Next() {
		var a models.Asset
		var c models.Category
		var icon, brand, model, serialNumber, warrantyEndDate, expiryDate, saleDate, brokenDate, notes sql.NullString
		var warrantyMonths sql.NullInt64
		var salePrice sql.NullFloat64
		var createdAtStr, updatedAtStr string

		if err := rows.Scan(
			&a.ID, &a.CategoryID, &a.Name, &icon, &brand, &model, &serialNumber,
			&a.PurchaseDate, &a.PurchasePrice, &warrantyMonths, &warrantyEndDate, &expiryDate,
			&a.Status, &saleDate, &salePrice, &brokenDate, &notes, &createdAtStr, &updatedAtStr,
			&c.ID, &c.Name, &c.HasWarranty, &c.HasExpiry, &c.CanSell, &c.CanBreak, &c.HasSerial, &c.HasModel,
		); err != nil {
			WriteError(w, http.StatusInternalServerError, "扫描资产失败")
			return
		}

		if icon.Valid {
			a.Icon = &icon.String
		}
		if brand.Valid {
			a.Brand = &brand.String
		}
		if model.Valid {
			a.Model = &model.String
		}
		if serialNumber.Valid {
			a.SerialNumber = &serialNumber.String
		}
		if warrantyMonths.Valid {
			v := int(warrantyMonths.Int64)
			a.WarrantyMonths = &v
		}
		if warrantyEndDate.Valid {
			a.WarrantyEndDate = &warrantyEndDate.String
		}
		if expiryDate.Valid {
			a.ExpiryDate = &expiryDate.String
		}
		if saleDate.Valid {
			a.SaleDate = &saleDate.String
		}
		if salePrice.Valid {
			a.SalePrice = &salePrice.Float64
		}
		if brokenDate.Valid {
			a.BrokenDate = &brokenDate.String
		}
		if notes.Valid {
			a.Notes = &notes.String
		}

		tCreated, _ := time.Parse("2006-01-02 15:04:05", strings.Split(createdAtStr, ".")[0])
		tUpdated, _ := time.Parse("2006-01-02 15:04:05", strings.Split(updatedAtStr, ".")[0])
		a.CreatedAt = tCreated
		a.UpdatedAt = tUpdated

		a.Category = &c
		assets = append(assets, &a)
	}
	_ = rows.Close()

	// Sync expiry status
	for _, a := range assets {
		if services.SyncExpiryStatus(a, today) {
			_, _ = h.db.Exec(`UPDATE assets SET status = ?, updated_at = ? WHERE id = ?`, a.Status, localNowStr, a.ID)
		}
	}

	// Filter by status in-memory if requested
	var filtered []*models.Asset
	for _, a := range assets {
		if statusFilter == "" || a.Status == statusFilter {
			filtered = append(filtered, a)
		}
	}

	var outs []models.AssetOut
	for _, a := range filtered {
		outs = append(outs, h.toAssetOut(a, today))
	}

	// In-memory sorting if specified
	if sortBy != "" {
		sort.Slice(outs, func(i, j int) bool {
			var less bool
			if sortBy == "purchase_date" {
				less = outs[i].PurchaseDate < outs[j].PurchaseDate
			} else if sortBy == "purchase_price" {
				less = outs[i].PurchasePrice < outs[j].PurchasePrice
			} else if sortBy == "daily_cost" {
				c1, c2 := 0.0, 0.0
				if outs[i].Cost != nil {
					c1 = outs[i].Cost.DailyCost
				}
				if outs[j].Cost != nil {
					c2 = outs[j].Cost.DailyCost
				}
				less = c1 < c2
			}
			if sortDir == "desc" {
				return !less
			}
			return less
		})
	}

	if outs == nil {
		outs = []models.AssetOut{}
	}
	WriteJSON(w, http.StatusOK, models.AssetListOut{
		Items: outs,
		Total: len(outs),
	})
}

func (h *AssetsHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body models.AssetCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if body.Name == "" || len(body.Name) > 100 {
		WriteError(w, http.StatusBadRequest, "资产名称长度须在 1 到 100 字符之间")
		return
	}
	if body.PurchaseDate == "" {
		WriteError(w, http.StatusBadRequest, "购买日期不能为空")
		return
	}
	if body.PurchasePrice < 0 {
		WriteError(w, http.StatusBadRequest, "购买价格不能为负数")
		return
	}
	body.PurchasePrice = services.RoundFloat(body.PurchasePrice, 2)

	cat, err := h.validateCategory(body.CategoryID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	status := models.StatusInUse
	if body.Status != nil && *body.Status != "" {
		status = *body.Status
	}

	if err := checkStatusFields(cat, status, body.SaleDate, body.SalePrice, body.BrokenDate, true); err != nil {
		WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	asset := &models.Asset{
		CategoryID:      body.CategoryID,
		Name:            body.Name,
		Icon:            body.Icon,
		Brand:           body.Brand,
		Model:           body.Model,
		SerialNumber:    body.SerialNumber,
		PurchaseDate:    body.PurchaseDate,
		PurchasePrice:   body.PurchasePrice,
		WarrantyMonths:  body.WarrantyMonths,
		WarrantyEndDate: body.WarrantyEndDate,
		ExpiryDate:      body.ExpiryDate,
		Status:          status,
		SaleDate:        body.SaleDate,
		SalePrice:       body.SalePrice,
		BrokenDate:      body.BrokenDate,
		Notes:           body.Notes,
		Category:        cat,
	}

	services.ApplyWarranty(asset, cat, true)
	today := h.cfg.TodayStr()
	services.SyncExpiryStatus(asset, today)

	localNowStr := h.cfg.LocalNow().Format("2006-01-02 15:04:05")
	res, err := h.db.Exec(`
		INSERT INTO assets (
			category_id, name, icon, brand, model, serial_number, purchase_date, purchase_price,
			warranty_months, warranty_end_date, expiry_date, status, sale_date, sale_price, broken_date, notes,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, asset.CategoryID, asset.Name, asset.Icon, asset.Brand, asset.Model, asset.SerialNumber, asset.PurchaseDate, asset.PurchasePrice,
		asset.WarrantyMonths, asset.WarrantyEndDate, asset.ExpiryDate, asset.Status, asset.SaleDate, asset.SalePrice, asset.BrokenDate, asset.Notes,
		localNowStr, localNowStr)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "创建资产失败")
		return
	}

	id, _ := res.LastInsertId()
	asset.ID = id
	tNow := h.cfg.LocalNow()
	asset.CreatedAt = tNow
	asset.UpdatedAt = tNow

	WriteJSON(w, http.StatusOK, h.toAssetOut(asset, today))
}

func (h *AssetsHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "无效的资产 ID")
		return
	}

	var a models.Asset
	var c models.Category
	var icon, brand, model, serialNumber, warrantyEndDate, expiryDate, saleDate, brokenDate, notes sql.NullString
	var warrantyMonths sql.NullInt64
	var salePrice sql.NullFloat64
	var createdAtStr, updatedAtStr string

	err = h.db.QueryRow(`
		SELECT a.id, a.category_id, a.name, a.icon, a.brand, a.model, a.serial_number,
		       a.purchase_date, a.purchase_price, a.warranty_months, a.warranty_end_date, a.expiry_date,
		       a.status, a.sale_date, a.sale_price, a.broken_date, a.notes, a.created_at, a.updated_at,
		       c.id, c.name, c.has_warranty, c.has_expiry, c.can_sell, c.can_break, c.has_serial, c.has_model
		FROM assets a
		JOIN categories c ON a.category_id = c.id
		WHERE a.id = ?
	`, assetID).Scan(
		&a.ID, &a.CategoryID, &a.Name, &icon, &brand, &model, &serialNumber,
		&a.PurchaseDate, &a.PurchasePrice, &warrantyMonths, &warrantyEndDate, &expiryDate,
		&a.Status, &saleDate, &salePrice, &brokenDate, &notes, &createdAtStr, &updatedAtStr,
		&c.ID, &c.Name, &c.HasWarranty, &c.HasExpiry, &c.CanSell, &c.CanBreak, &c.HasSerial, &c.HasModel,
	)
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "资产不存在")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "获取资产失败")
		return
	}

	if icon.Valid {
		a.Icon = &icon.String
	}
	if brand.Valid {
		a.Brand = &brand.String
	}
	if model.Valid {
		a.Model = &model.String
	}
	if serialNumber.Valid {
		a.SerialNumber = &serialNumber.String
	}
	if warrantyMonths.Valid {
		v := int(warrantyMonths.Int64)
		a.WarrantyMonths = &v
	}
	if warrantyEndDate.Valid {
		a.WarrantyEndDate = &warrantyEndDate.String
	}
	if expiryDate.Valid {
		a.ExpiryDate = &expiryDate.String
	}
	if saleDate.Valid {
		a.SaleDate = &saleDate.String
	}
	if salePrice.Valid {
		a.SalePrice = &salePrice.Float64
	}
	if brokenDate.Valid {
		a.BrokenDate = &brokenDate.String
	}
	if notes.Valid {
		a.Notes = &notes.String
	}

	tCreated, _ := time.Parse("2006-01-02 15:04:05", strings.Split(createdAtStr, ".")[0])
	tUpdated, _ := time.Parse("2006-01-02 15:04:05", strings.Split(updatedAtStr, ".")[0])
	a.CreatedAt = tCreated
	a.UpdatedAt = tUpdated
	a.Category = &c

	today := h.cfg.TodayStr()
	if services.SyncExpiryStatus(&a, today) {
		localNowStr := h.cfg.LocalNow().Format("2006-01-02 15:04:05")
		_, _ = h.db.Exec(`UPDATE assets SET status = ?, updated_at = ? WHERE id = ?`, a.Status, localNowStr, a.ID)
	}

	WriteJSON(w, http.StatusOK, h.toAssetOut(&a, today))
}

func (h *AssetsHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "无效的资产 ID")
		return
	}

	var a models.Asset
	var c models.Category
	var icon, brand, model, serialNumber, warrantyEndDate, expiryDate, saleDate, brokenDate, notes sql.NullString
	var warrantyMonths sql.NullInt64
	var salePrice sql.NullFloat64
	var createdAtStr, updatedAtStr string

	err = h.db.QueryRow(`
		SELECT a.id, a.category_id, a.name, a.icon, a.brand, a.model, a.serial_number,
		       a.purchase_date, a.purchase_price, a.warranty_months, a.warranty_end_date, a.expiry_date,
		       a.status, a.sale_date, a.sale_price, a.broken_date, a.notes, a.created_at, a.updated_at,
		       c.id, c.name, c.has_warranty, c.has_expiry, c.can_sell, c.can_break, c.has_serial, c.has_model
		FROM assets a
		JOIN categories c ON a.category_id = c.id
		WHERE a.id = ?
	`, assetID).Scan(
		&a.ID, &a.CategoryID, &a.Name, &icon, &brand, &model, &serialNumber,
		&a.PurchaseDate, &a.PurchasePrice, &warrantyMonths, &warrantyEndDate, &expiryDate,
		&a.Status, &saleDate, &salePrice, &brokenDate, &notes, &createdAtStr, &updatedAtStr,
		&c.ID, &c.Name, &c.HasWarranty, &c.HasExpiry, &c.CanSell, &c.CanBreak, &c.HasSerial, &c.HasModel,
	)
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "资产不存在")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "获取资产失败")
		return
	}

	if icon.Valid {
		a.Icon = &icon.String
	}
	if brand.Valid {
		a.Brand = &brand.String
	}
	if model.Valid {
		a.Model = &model.String
	}
	if serialNumber.Valid {
		a.SerialNumber = &serialNumber.String
	}
	if warrantyMonths.Valid {
		v := int(warrantyMonths.Int64)
		a.WarrantyMonths = &v
	}
	if warrantyEndDate.Valid {
		a.WarrantyEndDate = &warrantyEndDate.String
	}
	if expiryDate.Valid {
		a.ExpiryDate = &expiryDate.String
	}
	if saleDate.Valid {
		a.SaleDate = &saleDate.String
	}
	if salePrice.Valid {
		a.SalePrice = &salePrice.Float64
	}
	if brokenDate.Valid {
		a.BrokenDate = &brokenDate.String
	}
	if notes.Valid {
		a.Notes = &notes.String
	}

	tCreated, _ := time.Parse("2006-01-02 15:04:05", strings.Split(createdAtStr, ".")[0])
	tUpdated, _ := time.Parse("2006-01-02 15:04:05", strings.Split(updatedAtStr, ".")[0])
	a.CreatedAt = tCreated
	a.UpdatedAt = tUpdated
	a.Category = &c

	// Read request into raw map to detect unset fields
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		WriteError(w, http.StatusBadRequest, "无效的请求格式")
		return
	}

	currentCat := a.Category
	if _, ok := data["category_id"]; ok && data["category_id"] != nil {
		var newCatID int64
		switch v := data["category_id"].(type) {
		case float64:
			newCatID = int64(v)
		case int64:
			newCatID = v
		}
		newCat, err := h.validateCategory(newCatID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		a.CategoryID = newCatID
		currentCat = newCat
		a.Category = newCat
	}

	// Status transition logic
	statusVal := a.Status
	if v, ok := data["status"].(string); ok && v != "" {
		statusVal = v
	}

	// Parse potential sale/broken values from data or existing
	targetSaleDate := a.SaleDate
	if _, ok := data["sale_date"]; ok {
		if s, isStr := data["sale_date"].(string); isStr && s != "" {
			targetSaleDate = &s
		} else {
			targetSaleDate = nil
		}
	}
	targetSalePrice := a.SalePrice
	if _, ok := data["sale_price"]; ok {
		if f, isNum := data["sale_price"].(float64); isNum {
			targetSalePrice = &f
		} else {
			targetSalePrice = nil
		}
	}
	targetBrokenDate := a.BrokenDate
	if _, ok := data["broken_date"]; ok {
		if s, isStr := data["broken_date"].(string); isStr && s != "" {
			targetBrokenDate = &s
		} else {
			targetBrokenDate = nil
		}
	}

	if _, statusInReq := data["status"]; statusInReq && statusVal != a.Status {
		if err := checkStatusFields(currentCat, statusVal, targetSaleDate, targetSalePrice, targetBrokenDate, true); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		if statusVal != models.StatusSold {
			a.SaleDate = nil
			a.SalePrice = nil
		} else {
			a.SaleDate = targetSaleDate
			a.SalePrice = targetSalePrice
		}
		if statusVal != models.StatusBroken {
			a.BrokenDate = nil
		} else {
			a.BrokenDate = targetBrokenDate
		}
		a.Status = statusVal
	}

	// Update base fields if present
	if v, ok := data["name"].(string); ok {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" || len(trimmed) > 100 {
			WriteError(w, http.StatusBadRequest, "资产名称长度须在 1 到 100 字符之间")
			return
		}
		a.Name = trimmed
	}
	if _, ok := data["icon"]; ok {
		if s, isStr := data["icon"].(string); isStr && s != "" {
			a.Icon = &s
		} else {
			a.Icon = nil
		}
	}
	if _, ok := data["brand"]; ok {
		if s, isStr := data["brand"].(string); isStr && s != "" {
			a.Brand = &s
		} else {
			a.Brand = nil
		}
	}
	if _, ok := data["model"]; ok {
		if s, isStr := data["model"].(string); isStr && s != "" {
			a.Model = &s
		} else {
			a.Model = nil
		}
	}
	if _, ok := data["serial_number"]; ok {
		if s, isStr := data["serial_number"].(string); isStr && s != "" {
			a.SerialNumber = &s
		} else {
			a.SerialNumber = nil
		}
	}
	if v, ok := data["purchase_date"].(string); ok && v != "" {
		a.PurchaseDate = v
	}
	if v, ok := data["purchase_price"].(float64); ok {
		if v < 0 {
			WriteError(w, http.StatusBadRequest, "购买价格不能为负数")
			return
		}
		a.PurchasePrice = services.RoundFloat(v, 2)
	}
	if _, ok := data["warranty_months"]; ok {
		if v, isNum := data["warranty_months"].(float64); isNum && v > 0 {
			intV := int(v)
			a.WarrantyMonths = &intV
		} else {
			a.WarrantyMonths = nil
		}
	}
	if _, ok := data["warranty_end_date"]; ok {
		if s, isStr := data["warranty_end_date"].(string); isStr && s != "" {
			a.WarrantyEndDate = &s
		} else {
			a.WarrantyEndDate = nil
		}
	}
	if _, ok := data["expiry_date"]; ok {
		if s, isStr := data["expiry_date"].(string); isStr && s != "" {
			a.ExpiryDate = &s
		} else {
			a.ExpiryDate = nil
		}
	}
	if _, ok := data["notes"]; ok {
		if s, isStr := data["notes"].(string); isStr && s != "" {
			a.Notes = &s
		} else {
			a.Notes = nil
		}
	}

	// If asset is currently sold, update & check non-null fields
	if a.Status == models.StatusSold {
		if _, ok := data["sale_date"]; ok {
			a.SaleDate = targetSaleDate
		}
		if _, ok := data["sale_price"]; ok {
			a.SalePrice = targetSalePrice
		}
		if err := checkStatusFields(currentCat, a.Status, a.SaleDate, a.SalePrice, a.BrokenDate, false); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// If asset is currently broken, update & check non-null fields
	if a.Status == models.StatusBroken {
		if _, ok := data["broken_date"]; ok {
			a.BrokenDate = targetBrokenDate
		}
		if err := checkStatusFields(currentCat, a.Status, a.SaleDate, a.SalePrice, a.BrokenDate, false); err != nil {
			WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Force warranty calculation if category, purchase_date or warranty_months changed
	_, hasCat := data["category_id"]
	_, hasDate := data["purchase_date"]
	_, hasMonths := data["warranty_months"]
	forceWarranty := hasCat || hasDate || hasMonths
	services.ApplyWarranty(&a, currentCat, forceWarranty)

	today := h.cfg.TodayStr()
	services.SyncExpiryStatus(&a, today)

	localNowStr := h.cfg.LocalNow().Format("2006-01-02 15:04:05")
	a.UpdatedAt = h.cfg.LocalNow()

	_, err = h.db.Exec(`
		UPDATE assets
		SET category_id = ?, name = ?, icon = ?, brand = ?, model = ?, serial_number = ?,
		    purchase_date = ?, purchase_price = ?, warranty_months = ?, warranty_end_date = ?, expiry_date = ?,
		    status = ?, sale_date = ?, sale_price = ?, broken_date = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, a.CategoryID, a.Name, a.Icon, a.Brand, a.Model, a.SerialNumber,
		a.PurchaseDate, a.PurchasePrice, a.WarrantyMonths, a.WarrantyEndDate, a.ExpiryDate,
		a.Status, a.SaleDate, a.SalePrice, a.BrokenDate, a.Notes, localNowStr, a.ID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "更新资产失败")
		return
	}

	WriteJSON(w, http.StatusOK, h.toAssetOut(&a, today))
}

func (h *AssetsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	assetID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "无效的资产 ID")
		return
	}

	var existID int64
	err = h.db.QueryRow(`SELECT id FROM assets WHERE id = ?`, assetID).Scan(&existID)
	if err == sql.ErrNoRows {
		WriteError(w, http.StatusNotFound, "资产不存在")
		return
	}

	// Cascade delete reminder logs and asset
	_, _ = h.db.Exec(`DELETE FROM reminder_logs WHERE asset_id = ?`, assetID)
	_, err = h.db.Exec(`DELETE FROM assets WHERE id = ?`, assetID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "删除资产失败")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

