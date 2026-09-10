package api

import (
	"database/sql"
	"net/http"
	"sort"
	"strings"
	"time"

	"thingspan/internal/config"
	"thingspan/internal/models"
	"thingspan/internal/services"
)

type DashboardHandler struct {
	db  *sql.DB
	cfg *config.Config
}

func NewDashboardHandler(db *sql.DB, cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{db: db, cfg: cfg}
}

func (h *DashboardHandler) Get(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT a.id, a.category_id, a.name, a.icon, a.brand, a.model, a.serial_number,
		       a.purchase_date, a.purchase_price, a.warranty_months, a.warranty_end_date, a.expiry_date,
		       a.status, a.sale_date, a.sale_price, a.broken_date, a.notes, a.created_at, a.updated_at,
		       c.id, c.name, c.has_warranty, c.has_expiry, c.can_sell, c.can_break, c.has_serial, c.has_model
		FROM assets a
		JOIN categories c ON a.category_id = c.id
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "获取看板数据失败")
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

	totalInvested := 0.0
	dailyCostTotal := 0.0
	inUseAssets := 0

	for _, a := range assets {
		totalInvested += a.PurchasePrice
		cost := services.CalcCost(a, today)
		dailyCostTotal += cost.DailyCost
		if a.Status == models.StatusInUse {
			inUseAssets++
		}
	}

	horizon := services.AddDays(today, 30)
	var expiringSoon []models.ExpiringAsset

	for _, a := range assets {
		if a.Status != models.StatusInUse {
			continue
		}
		targets := services.ReminderTargetDates(a, today)
		for _, target := range targets {
			if target.TargetDate >= today && target.TargetDate <= horizon {
				catName := ""
				if a.Category != nil {
					catName = a.Category.Name
				}
				daysLeft := services.DaysBetween(today, target.TargetDate)
				expiringSoon = append(expiringSoon, models.ExpiringAsset{
					ID:           a.ID,
					Name:         a.Name,
					CategoryName: catName,
					TargetDate:   target.TargetDate,
					DaysLeft:     daysLeft,
					DateType:     target.DateType,
				})
			}
		}
	}

	sort.Slice(expiringSoon, func(i, j int) bool {
		return expiringSoon[i].DaysLeft < expiringSoon[j].DaysLeft
	})

	if expiringSoon == nil {
		expiringSoon = []models.ExpiringAsset{}
	}

	WriteJSON(w, http.StatusOK, models.DashboardOut{
		TotalAssets:    len(assets),
		InUseAssets:    inUseAssets,
		TotalInvested:  services.RoundFloat(totalInvested, 2),
		DailyCostTotal: services.RoundFloat(dailyCostTotal, 2),
		ExpiringSoon:   expiringSoon,
	})
}

