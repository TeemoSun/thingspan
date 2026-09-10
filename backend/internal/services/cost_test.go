package services

import (
	"testing"

	"thingspan/internal/models"
)

func TestDaysBetweenAndAddDays(t *testing.T) {
	d := DaysBetween("2024-01-01", "2024-01-11")
	if d != 10 {
		t.Fatalf("expected 10 days, got %d", d)
	}

	res := AddDays("2024-01-01", 30)
	if res != "2024-01-31" {
		t.Fatalf("expected 2024-01-31, got %s", res)
	}
}

func TestCalcCostInUse(t *testing.T) {
	asset := &models.Asset{
		Status:        models.StatusInUse,
		PurchaseDate:  "2024-01-01",
		PurchasePrice: 1000.0,
	}

	cost := CalcCost(asset, "2024-01-11")
	if cost.PeriodDays != 10 {
		t.Errorf("expected 10 days, got %d", cost.PeriodDays)
	}
	if cost.TotalCost != 1000.0 {
		t.Errorf("expected 1000.0 total cost, got %f", cost.TotalCost)
	}
	if cost.DailyCost != 100.0 {
		t.Errorf("expected 100.0 daily cost, got %f", cost.DailyCost)
	}
	if cost.Formula != models.StatusInUse {
		t.Errorf("expected formula 'in_use', got %s", cost.Formula)
	}
}

func TestCalcCostInUseWithExpiry(t *testing.T) {
	expiry := "2024-01-31"
	asset := &models.Asset{
		Status:        models.StatusInUse,
		PurchaseDate:  "2024-01-01",
		PurchasePrice: 300.0,
		ExpiryDate:    &expiry,
		Category: &models.Category{
			HasExpiry: true,
		},
	}

	// Regardless of today's date, should divide by (expiry - purchase) = 30 days
	cost := CalcCost(asset, "2024-01-11")
	if cost.PeriodDays != 30 {
		t.Errorf("expected 30 days, got %d", cost.PeriodDays)
	}
	if cost.DailyCost != 10.0 {
		t.Errorf("expected 10.0 daily cost, got %f", cost.DailyCost)
	}
	if cost.Formula != models.StatusExpired {
		t.Errorf("expected formula 'expired', got %s", cost.Formula)
	}
}

func TestCalcCostSold(t *testing.T) {
	saleDate := "2024-01-21"
	salePrice := 400.0
	asset := &models.Asset{
		Status:        models.StatusSold,
		PurchaseDate:  "2024-01-01",
		PurchasePrice: 1000.0,
		SaleDate:      &saleDate,
		SalePrice:     &salePrice,
	}

	cost := CalcCost(asset, "2024-02-01")
	if cost.PeriodDays != 20 {
		t.Errorf("expected 20 days, got %d", cost.PeriodDays)
	}
	if cost.TotalCost != 600.0 {
		t.Errorf("expected 600.0 total cost, got %f", cost.TotalCost)
	}
	if cost.DailyCost != 30.0 {
		t.Errorf("expected 30.0 daily cost, got %f", cost.DailyCost)
	}
	if cost.Formula != models.StatusSold {
		t.Errorf("expected formula 'sold', got %s", cost.Formula)
	}
}

func TestCalcCostBroken(t *testing.T) {
	brokenDate := "2024-01-21"
	asset := &models.Asset{
		Status:        models.StatusBroken,
		PurchaseDate:  "2024-01-01",
		PurchasePrice: 1000.0,
		BrokenDate:    &brokenDate,
	}

	cost := CalcCost(asset, "2024-02-01")
	if cost.PeriodDays != 20 {
		t.Errorf("expected 20 days, got %d", cost.PeriodDays)
	}
	if cost.DailyCost != 50.0 {
		t.Errorf("expected 50.0 daily cost, got %f", cost.DailyCost)
	}
	if cost.Formula != models.StatusBroken {
		t.Errorf("expected formula 'broken', got %s", cost.Formula)
	}
}

func TestCalcCostExpired(t *testing.T) {
	expiryDate := "2024-01-21"
	asset := &models.Asset{
		Status:        models.StatusExpired,
		PurchaseDate:  "2024-01-01",
		PurchasePrice: 1000.0,
		ExpiryDate:    &expiryDate,
	}

	cost := CalcCost(asset, "2024-02-01")
	if cost.PeriodDays != 20 {
		t.Errorf("expected 20 days, got %d", cost.PeriodDays)
	}
	if cost.DailyCost != 50.0 {
		t.Errorf("expected 50.0 daily cost, got %f", cost.DailyCost)
	}
	if cost.Formula != models.StatusExpired {
		t.Errorf("expected formula 'expired', got %s", cost.Formula)
	}
}

func TestSyncExpiryStatus(t *testing.T) {
	expiryPast := "2024-01-10"
	asset := &models.Asset{
		Status:     models.StatusInUse,
		ExpiryDate: &expiryPast,
		Category: &models.Category{
			HasExpiry: true,
		},
	}

	today := "2024-01-15"
	changed := SyncExpiryStatus(asset, today)
	if !changed || asset.Status != models.StatusExpired {
		t.Fatalf("expected asset status to change to expired")
	}

	// Change expiry to future
	expiryFuture := "2024-01-20"
	asset.ExpiryDate = &expiryFuture
	changed = SyncExpiryStatus(asset, today)
	if !changed || asset.Status != models.StatusInUse {
		t.Fatalf("expected asset status to restore to in_use")
	}

	// Category without has_expiry
	asset.Category.HasExpiry = false
	asset.ExpiryDate = &expiryPast
	changed = SyncExpiryStatus(asset, today)
	if changed {
		t.Fatalf("expected no change when category has_expiry is false")
	}
}

func TestApplyWarranty(t *testing.T) {
	months := 12
	asset := &models.Asset{
		PurchaseDate:   "2024-01-01",
		WarrantyMonths: &months,
	}
	cat := &models.Category{
		HasWarranty: true,
	}

	ApplyWarranty(asset, cat, true)
	if asset.WarrantyEndDate == nil {
		t.Fatalf("expected warranty end date to be set")
	}

	// 12 * 30 = 360 days from 2024-01-01
	expected := AddDays("2024-01-01", 360)
	if *asset.WarrantyEndDate != expected {
		t.Fatalf("expected %s, got %s", expected, *asset.WarrantyEndDate)
	}

	// If category does not have warranty and force is true, end date should be cleared
	cat.HasWarranty = false
	ApplyWarranty(asset, cat, true)
	if asset.WarrantyEndDate != nil {
		t.Fatalf("expected warranty end date to be cleared")
	}
}

