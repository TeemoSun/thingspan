package models

import "time"

const (
	StatusInUse   = "in_use"
	StatusSold    = "sold"
	StatusBroken  = "broken"
	StatusExpired = "expired"
)

type Category struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	HasWarranty bool      `json:"has_warranty"`
	HasExpiry   bool      `json:"has_expiry"`
	CanSell     bool      `json:"can_sell"`
	CanBreak    bool      `json:"can_break"`
	HasSerial   bool      `json:"has_serial"`
	HasModel    bool      `json:"has_model"`
	CreatedAt   time.Time `json:"created_at"`
}

type Asset struct {
	ID              int64     `json:"id"`
	CategoryID      int64     `json:"category_id"`
	Name            string    `json:"name"`
	Icon            *string   `json:"icon"`
	Brand           *string   `json:"brand"`
	Model           *string   `json:"model"`
	SerialNumber    *string   `json:"serial_number"`
	PurchaseDate    string    `json:"purchase_date"` // YYYY-MM-DD
	PurchasePrice   float64   `json:"purchase_price"`
	WarrantyMonths  *int      `json:"warranty_months"`
	WarrantyEndDate *string   `json:"warranty_end_date"` // YYYY-MM-DD
	ExpiryDate      *string   `json:"expiry_date"`        // YYYY-MM-DD
	Status          string    `json:"status"`
	SaleDate        *string   `json:"sale_date"`   // YYYY-MM-DD
	SalePrice       *float64  `json:"sale_price"`
	BrokenDate      *string   `json:"broken_date"` // YYYY-MM-DD
	Notes           *string   `json:"notes"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	Category *Category `json:"category,omitempty"`
}

type ReminderLog struct {
	ID         int64     `json:"id"`
	AssetID    int64     `json:"asset_id"`
	AssetName  string    `json:"asset_name"`
	TargetDate string    `json:"target_date"` // YYYY-MM-DD
	LeadDays   int       `json:"lead_days"`
	SentAt     time.Time `json:"sent_at"`
	Sent       bool      `json:"sent"`
	Dismissed  bool      `json:"dismissed"`
}

