package services

import (
	"math"
	"sort"
	"time"

	"thingspan/internal/models"
)

func ParseDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02", s)
}

func DaysBetween(startDate, endDate string) int {
	t1, err1 := ParseDate(startDate)
	t2, err2 := ParseDate(endDate)
	if err1 != nil || err2 != nil {
		return 0
	}
	// Use UTC midnight to calculate integer calendar days accurately
	t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.UTC)
	t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.UTC)
	return int(t2.Sub(t1) / (24 * time.Hour))
}

func AddDays(startDate string, days int) string {
	t, err := ParseDate(startDate)
	if err != nil {
		return ""
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}

func RoundFloat(val float64, precision int) float64 {
	p := math.Pow10(precision)
	return math.Round(val*p) / p
}

// CalcCost computes the daily cost and period based on asset status and dates
func CalcCost(asset *models.Asset, today string) *models.CostOut {
	formula := models.StatusInUse
	totalCost := asset.PurchasePrice
	days := 0

	if asset.Status == models.StatusSold && asset.SaleDate != nil && *asset.SaleDate != "" && asset.SalePrice != nil {
		days = DaysBetween(asset.PurchaseDate, *asset.SaleDate)
		totalCost = RoundFloat(asset.PurchasePrice-*asset.SalePrice, 2)
		formula = models.StatusSold
	} else if asset.Status == models.StatusBroken && asset.BrokenDate != nil && *asset.BrokenDate != "" {
		days = DaysBetween(asset.PurchaseDate, *asset.BrokenDate)
		formula = models.StatusBroken
	} else if asset.Status == models.StatusExpired && asset.ExpiryDate != nil && *asset.ExpiryDate != "" {
		days = DaysBetween(asset.PurchaseDate, *asset.ExpiryDate)
		formula = models.StatusExpired
	} else if asset.Status == models.StatusInUse &&
		asset.ExpiryDate != nil && *asset.ExpiryDate != "" &&
		asset.Category != nil && asset.Category.HasExpiry {
		days = DaysBetween(asset.PurchaseDate, *asset.ExpiryDate)
		formula = models.StatusExpired
	} else {
		days = DaysBetween(asset.PurchaseDate, today)
		formula = models.StatusInUse
	}

	dailyCost := 0.0
	if days > 0 {
		dailyCost = RoundFloat(totalCost/float64(days), 4)
	}

	periodDays := days
	if periodDays < 0 {
		periodDays = 0
	}

	return &models.CostOut{
		PeriodDays: periodDays,
		TotalCost:  totalCost,
		DailyCost:  dailyCost,
		Formula:    formula,
	}
}

// SyncExpiryStatus updates asset status dynamically between in_use and expired
func SyncExpiryStatus(asset *models.Asset, today string) bool {
	if asset.Category == nil || !asset.Category.HasExpiry {
		return false
	}

	hasExpiryDate := asset.ExpiryDate != nil && *asset.ExpiryDate != ""
	if asset.Status == models.StatusInUse && hasExpiryDate && *asset.ExpiryDate < today {
		asset.Status = models.StatusExpired
		return true
	}
	if asset.Status == models.StatusExpired && (!hasExpiryDate || *asset.ExpiryDate >= today) {
		asset.Status = models.StatusInUse
		return true
	}

	return false
}

// ApplyWarranty calculates warranty_end_date = purchase_date + (warranty_months * 30 days)
func ApplyWarranty(asset *models.Asset, category *models.Category, force bool) {
	hasWarrantyMonths := asset.WarrantyMonths != nil && *asset.WarrantyMonths > 0
	if category != nil && category.HasWarranty && asset.PurchaseDate != "" && hasWarrantyMonths {
		if force || asset.WarrantyEndDate == nil || *asset.WarrantyEndDate == "" {
			endDate := AddDays(asset.PurchaseDate, *asset.WarrantyMonths*30)
			asset.WarrantyEndDate = &endDate
		}
	} else if force {
		asset.WarrantyEndDate = nil
	}
}

type ReminderTarget struct {
	DateType   string // "warranty" | "expiry"
	TargetDate string // YYYY-MM-DD
}

// ReminderTargetDates returns list of targets (warranty and expiry), optionally filtered by target_date >= today
func ReminderTargetDates(asset *models.Asset, today string) []ReminderTarget {
	var targets []ReminderTarget
	if asset.WarrantyEndDate != nil && *asset.WarrantyEndDate != "" {
		if today == "" || *asset.WarrantyEndDate >= today {
			targets = append(targets, ReminderTarget{
				DateType:   "warranty",
				TargetDate: *asset.WarrantyEndDate,
			})
		}
	}
	if asset.ExpiryDate != nil && *asset.ExpiryDate != "" {
		if today == "" || *asset.ExpiryDate >= today {
			targets = append(targets, ReminderTarget{
				DateType:   "expiry",
				TargetDate: *asset.ExpiryDate,
			})
		}
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].TargetDate < targets[j].TargetDate
	})
	return targets
}

