package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"thingspan/internal/config"
	"thingspan/internal/database"
	"thingspan/internal/models"
	"thingspan/internal/security"
	"thingspan/internal/services"
)

func setupTestServer(t *testing.T) (http.Handler, *sql.DB, string) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	cfg := &config.Config{
		AppPassword:              "test-password-123",
		JWTSecret:                "test-jwt-secret-12345678901234567890",
		AccessTokenExpireMinutes: 30,
		RefreshTokenExpireDays:   30,
		TZ:                       "Asia/Shanghai",
		DataDir:                  "./test-data",
		Port:                     8000,
		StaticDir:                "./static",
		LeadDays:                 []int{1, 7, 30},
		ReminderLeadDays:         "30,7,1",
		ReminderCheckHour:        9,
		Location:                 time.FixedZone("CST", 8*3600),
	}

	pm, _ := security.NewPasswordManager(cfg.AppPassword)
	jwtm := security.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenExpireMinutes, cfg.RefreshTokenExpireDays)
	rl := security.NewRateLimiter()

	scanner := services.NewReminderScanner(db, cfg, services.NewEmailSender(cfg))
	router := NewRouter(cfg, db, pm, jwtm, rl, scanner)

	// Pre-generate valid access token for convenience
	token, _ := jwtm.CreateAccessToken()

	return router, db, token
}

func doRequest(handler http.Handler, method, target string, body interface{}, token string) *httptest.ResponseRecorder {
	var bodyReader *bytes.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(data)
	} else {
		bodyReader = bytes.NewReader([]byte{})
	}

	req := httptest.NewRequest(method, target, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestHealthAndSecurityHeaders(t *testing.T) {
	handler, db, _ := setupTestServer(t)
	defer db.Close()

	rec := doRequest(handler, "GET", "/healthz", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /healthz, got %d", rec.Code)
	}

	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("expected X-Content-Type-Options nosniff")
	}
	if rec.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("expected X-Frame-Options SAMEORIGIN")
	}

	recAPIHealth := doRequest(handler, "GET", "/api/health", nil, "")
	if recAPIHealth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK from /api/health, got %d", recAPIHealth.Code)
	}
}

func TestAuthFlow(t *testing.T) {
	handler, db, _ := setupTestServer(t)
	defer db.Close()

	// 1. Login with wrong password
	rec := doRequest(handler, "POST", "/api/auth/login", models.LoginRequest{Password: "wrong"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for wrong password, got %d", rec.Code)
	}

	// 2. Login with correct password
	rec = doRequest(handler, "POST", "/api/auth/login", models.LoginRequest{Password: "test-password-123"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for correct password, got %d", rec.Code)
	}

	var tokenResp models.TokenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &tokenResp); err != nil {
		t.Fatalf("failed to parse token response: %v", err)
	}
	if tokenResp.AccessToken == "" || tokenResp.RefreshToken == "" {
		t.Fatalf("expected access and refresh tokens")
	}

	// 3. Refresh token
	rec = doRequest(handler, "POST", "/api/auth/refresh", models.RefreshRequest{RefreshToken: tokenResp.RefreshToken}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for token refresh, got %d", rec.Code)
	}

	var newTokens models.TokenResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &newTokens)

	// 4. Old refresh token should now be revoked
	rec = doRequest(handler, "POST", "/api/auth/refresh", models.RefreshRequest{RefreshToken: tokenResp.RefreshToken}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when reusing revoked refresh token, got %d", rec.Code)
	}
}

func TestCategoryCRUD(t *testing.T) {
	handler, db, token := setupTestServer(t)
	defer db.Close()

	// 1. Protected without auth
	rec := doRequest(handler, "GET", "/api/categories", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth, got %d", rec.Code)
	}

	// 2. Create category
	catCreate := models.CategoryCreate{
		Name:        "数码产品",
		HasWarranty: true,
		HasExpiry:   false,
		CanSell:     true,
		CanBreak:    true,
		HasSerial:   true,
		HasModel:    true,
	}
	rec = doRequest(handler, "POST", "/api/categories", catCreate, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var catOut models.CategoryOut
	_ = json.Unmarshal(rec.Body.Bytes(), &catOut)
	if catOut.Name != "数码产品" || !catOut.HasWarranty {
		t.Fatalf("unexpected category data: %+v", catOut)
	}

	// 3. Duplicate name
	rec = doRequest(handler, "POST", "/api/categories", catCreate, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for duplicate name, got %d", rec.Code)
	}

	// 4. List categories
	rec = doRequest(handler, "GET", "/api/categories", nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var catList []models.CategoryOut
	_ = json.Unmarshal(rec.Body.Bytes(), &catList)
	if len(catList) != 1 {
		t.Fatalf("expected 1 category, got %d", len(catList))
	}

	// 5. Update category
	catCreate.Name = "数码数码"
	rec = doRequest(handler, "PUT", fmt.Sprintf("/api/categories/%d", catOut.ID), catCreate, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// 6. Delete category
	rec = doRequest(handler, "DELETE", fmt.Sprintf("/api/categories/%d", catOut.ID), nil, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAssetFullFlow(t *testing.T) {
	handler, db, token := setupTestServer(t)
	defer db.Close()

	// 1. Create a category with warranty and sell capabilities
	recCat := doRequest(handler, "POST", "/api/categories", models.CategoryCreate{
		Name:        "手机电脑",
		HasWarranty: true,
		HasExpiry:   false,
		CanSell:     true,
		CanBreak:    true,
		HasSerial:   true,
		HasModel:    true,
	}, token)
	var cat models.CategoryOut
	_ = json.Unmarshal(recCat.Body.Bytes(), &cat)

	// 2. Create Asset with 12 months warranty
	months := 12
	brand := "Apple"
	model := "MacBook Pro"
	serial := "C02XYZ"
	createPayload := models.AssetCreatePayload{
		CategoryID:     cat.ID,
		Name:           "MacBook Pro 16",
		Brand:          &brand,
		Model:          &model,
		SerialNumber:   &serial,
		PurchaseDate:   "2024-01-01",
		PurchasePrice:  16000.0,
		WarrantyMonths: &months,
	}

	rec := doRequest(handler, "POST", "/api/assets", createPayload, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("create asset failed: %d %s", rec.Code, rec.Body.String())
	}

	var assetOut models.AssetOut
	_ = json.Unmarshal(rec.Body.Bytes(), &assetOut)
	if assetOut.WarrantyEndDate == nil {
		t.Fatalf("expected warranty end date to be auto-computed")
	}
	expectedEnd := services.AddDays("2024-01-01", 360)
	if *assetOut.WarrantyEndDate != expectedEnd {
		t.Fatalf("expected warranty end date %s, got %s", expectedEnd, *assetOut.WarrantyEndDate)
	}

	// 3. Status transition validations
	// Transition to sold without date/price -> 400
	soldStatus := models.StatusSold
	rec = doRequest(handler, "PUT", fmt.Sprintf("/api/assets/%d", assetOut.ID), map[string]interface{}{
		"status": soldStatus,
	}, token)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for sold transition without date/price, got %d", rec.Code)
	}

	// Transition to sold with valid date and price
	saleDate := "2024-06-01"
	salePrice := 10000.0
	rec = doRequest(handler, "PUT", fmt.Sprintf("/api/assets/%d", assetOut.ID), map[string]interface{}{
		"status":     soldStatus,
		"sale_date":  saleDate,
		"sale_price": salePrice,
	}, token)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid sold transition, got %d %s", rec.Code, rec.Body.String())
	}

	_ = json.Unmarshal(rec.Body.Bytes(), &assetOut)
	if assetOut.Cost == nil || assetOut.Cost.Formula != "sold" {
		t.Fatalf("expected formula 'sold', got %+v", assetOut.Cost)
	}
	// 16000 - 10000 = 6000 total cost
	if assetOut.Cost.TotalCost != 6000.0 {
		t.Fatalf("expected total cost 6000, got %f", assetOut.Cost.TotalCost)
	}

	// 4. Test dynamic expiry status sync
	recExpCat := doRequest(handler, "POST", "/api/categories", models.CategoryCreate{
		Name:      "会员",
		HasExpiry: true,
	}, token)
	var expCat models.CategoryOut
	_ = json.Unmarshal(recExpCat.Body.Bytes(), &expCat)

	pastExpiry := "2020-01-01"
	recExpAsset := doRequest(handler, "POST", "/api/assets", models.AssetCreatePayload{
		CategoryID:    expCat.ID,
		Name:          "过期会员",
		PurchaseDate:  "2019-01-01",
		PurchasePrice: 100.0,
		ExpiryDate:    &pastExpiry,
	}, token)
	if recExpAsset.Code != http.StatusOK {
		t.Fatalf("failed to create expired asset: %s", recExpAsset.Body.String())
	}
	var expAssetOut models.AssetOut
	_ = json.Unmarshal(recExpAsset.Body.Bytes(), &expAssetOut)
	if expAssetOut.Status != models.StatusExpired {
		t.Fatalf("expected status to automatically sync to 'expired', got %s", expAssetOut.Status)
	}

	// 5. Test Dashboard
	recDash := doRequest(handler, "GET", "/api/dashboard", nil, token)
	if recDash.Code != http.StatusOK {
		t.Fatalf("expected 200 from dashboard, got %d", recDash.Code)
	}
	var dash models.DashboardOut
	_ = json.Unmarshal(recDash.Body.Bytes(), &dash)
	if dash.TotalAssets != 2 {
		t.Fatalf("expected 2 total assets, got %d", dash.TotalAssets)
	}

	// 6. Test Icons
	recIcons := doRequest(handler, "GET", "/api/icons", nil, token)
	if recIcons.Code != http.StatusOK {
		t.Fatalf("expected 200 from icons, got %d", recIcons.Code)
	}
	var iconsResp struct {
		Icons []string `json:"icons"`
	}
	_ = json.Unmarshal(recIcons.Body.Bytes(), &iconsResp)
	if len(iconsResp.Icons) == 0 {
		t.Fatalf("expected icons list to be populated")
	}

	// 7. Unknown /api/ path -> 404 JSON
	rec404 := doRequest(handler, "GET", "/api/nonexistent", nil, token)
	if rec404.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown api path, got %d", rec404.Code)
	}
	var errDetail models.ErrorDetail
	_ = json.Unmarshal(rec404.Body.Bytes(), &errDetail)
	if errDetail.Detail != "Not Found" {
		t.Fatalf("expected detail 'Not Found', got %s", errDetail.Detail)
	}
}

