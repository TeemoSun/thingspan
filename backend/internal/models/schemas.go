package models

type LoginRequest struct {
	Password string `json:"password"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type CategoryCreate struct {
	Name        string `json:"name"`
	HasWarranty bool   `json:"has_warranty"`
	HasExpiry   bool   `json:"has_expiry"`
	CanSell     bool   `json:"can_sell"`
	CanBreak    bool   `json:"can_break"`
	HasSerial   bool   `json:"has_serial"`
	HasModel    bool   `json:"has_model"`
}

type CategoryUpdate = CategoryCreate

type CategoryOut struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	HasWarranty bool   `json:"has_warranty"`
	HasExpiry   bool   `json:"has_expiry"`
	CanSell     bool   `json:"can_sell"`
	CanBreak    bool   `json:"can_break"`
	HasSerial   bool   `json:"has_serial"`
	HasModel    bool   `json:"has_model"`
	AssetsCount int    `json:"assets_count"`
}

type CostOut struct {
	PeriodDays int     `json:"period_days"`
	TotalCost  float64 `json:"total_cost"`
	DailyCost  float64 `json:"daily_cost"`
	Formula    string  `json:"formula"`
}

type AssetOut struct {
	ID              int64    `json:"id"`
	CategoryID      int64    `json:"category_id"`
	CategoryName    string   `json:"category_name"`
	Name            string   `json:"name"`
	Icon            *string  `json:"icon"`
	Brand           *string  `json:"brand"`
	Model           *string  `json:"model"`
	SerialNumber    *string  `json:"serial_number"`
	PurchaseDate    string   `json:"purchase_date"`
	PurchasePrice   float64  `json:"purchase_price"`
	WarrantyMonths  *int     `json:"warranty_months"`
	WarrantyEndDate *string  `json:"warranty_end_date"`
	ExpiryDate      *string  `json:"expiry_date"`
	Status          string   `json:"status"`
	SaleDate        *string  `json:"sale_date"`
	SalePrice       *float64 `json:"sale_price"`
	BrokenDate      *string  `json:"broken_date"`
	Notes           *string  `json:"notes"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
	Cost            *CostOut `json:"cost"`
}

type AssetListOut struct {
	Items []AssetOut `json:"items"`
	Total int        `json:"total"`
}

type AssetCreatePayload struct {
	CategoryID      int64    `json:"category_id"`
	Name            string   `json:"name"`
	Icon            *string  `json:"icon"`
	Brand           *string  `json:"brand"`
	Model           *string  `json:"model"`
	SerialNumber    *string  `json:"serial_number"`
	PurchaseDate    string   `json:"purchase_date"`
	PurchasePrice   float64  `json:"purchase_price"`
	WarrantyMonths  *int     `json:"warranty_months"`
	WarrantyEndDate *string  `json:"warranty_end_date"`
	ExpiryDate      *string  `json:"expiry_date"`
	Status          *string  `json:"status"`
	SaleDate        *string  `json:"sale_date"`
	SalePrice       *float64 `json:"sale_price"`
	BrokenDate      *string  `json:"broken_date"`
	Notes           *string  `json:"notes"`
}

type ExpiringAsset struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	CategoryName string `json:"category_name"`
	TargetDate   string `json:"target_date"`
	DaysLeft     int    `json:"days_left"`
	DateType     string `json:"date_type"` // "warranty" | "expiry"
}

type DashboardOut struct {
	TotalAssets    int             `json:"total_assets"`
	InUseAssets    int             `json:"in_use_assets"`
	TotalInvested  float64         `json:"total_invested"`
	DailyCostTotal float64         `json:"daily_cost_total"`
	ExpiringSoon   []ExpiringAsset `json:"expiring_soon"`
}

type ReminderOut struct {
	ID         int64  `json:"id"`
	AssetID    int64  `json:"asset_id"`
	AssetName  string `json:"asset_name"`
	TargetDate string `json:"target_date"`
	LeadDays   int    `json:"lead_days"`
	SentAt     string `json:"sent_at"`
	Sent       bool   `json:"sent"`
	Dismissed  bool   `json:"dismissed"`
}

type ErrorDetail struct {
	Detail string `json:"detail"`
}

