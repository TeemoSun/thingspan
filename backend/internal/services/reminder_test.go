package services

import (
	"database/sql"
	"testing"
	"time"

	"thingspan/internal/config"
	"thingspan/internal/database"
	"thingspan/internal/models"
)

type MockEmailSender struct {
	sentCount int
	subjects  []string
}

func (m *MockEmailSender) SendEmail(subject, html, plain string) bool {
	m.sentCount++
	m.subjects = append(m.subjects, subject)
	return true
}

func TestReminderScanner(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open memory db: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	cfg := &config.Config{
		TZ:                "Asia/Shanghai",
		LeadDays:          []int{1, 7, 30},
		ReminderLeadDays:  "30,7,1",
		ReminderCheckHour: 9,
		Location:          time.FixedZone("CST", 8*3600),
	}
	today := cfg.TodayStr()

	// Insert category with has_expiry
	res, err := db.Exec(`
		INSERT INTO categories (name, has_warranty, has_expiry, can_sell, can_break, has_serial, has_model, created_at)
		VALUES ('会员服务', 0, 1, 0, 0, 0, 0, datetime('now'))
	`)
	if err != nil {
		t.Fatalf("failed to insert category: %v", err)
	}
	catID, _ := res.LastInsertId()

	// 1. Asset expiring in 5 days (within 7-day lead window)
	targetDate := AddDays(today, 5)
	_, err = db.Exec(`
		INSERT INTO assets (category_id, name, purchase_date, purchase_price, expiry_date, status, created_at, updated_at)
		VALUES (?, 'Netflix 年费', ?, 300, ?, 'in_use', datetime('now'), datetime('now'))
	`, catID, today, targetDate)
	if err != nil {
		t.Fatalf("failed to insert asset: %v", err)
	}

	// 2. Asset already expired 2 days ago
	pastExpiry := AddDays(today, -2)
	_, err = db.Exec(`
		INSERT INTO assets (category_id, name, purchase_date, purchase_price, expiry_date, status, created_at, updated_at)
		VALUES (?, '旧会员', ?, 100, ?, 'in_use', datetime('now'), datetime('now'))
	`, catID, today, pastExpiry)
	if err != nil {
		t.Fatalf("failed to insert asset: %v", err)
	}

	sender := &MockEmailSender{}
	scanner := NewReminderScanner(db, cfg, sender)

	// Run first scan
	if err := scanner.RunScan(); err != nil {
		t.Fatalf("RunScan failed: %v", err)
	}

	if sender.sentCount != 1 {
		t.Fatalf("expected 1 email to be sent, got %d", sender.sentCount)
	}

	// Check reminder log inserted
	var count int
	_ = db.QueryRow(`SELECT count(*) FROM reminder_logs WHERE sent = 1`).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 sent reminder log, got %d", count)
	}

	// Check that the past-due asset was automatically marked expired
	var status string
	_ = db.QueryRow(`SELECT status FROM assets WHERE name = '旧会员'`).Scan(&status)
	if status != models.StatusExpired {
		t.Fatalf("expected '旧会员' status to be expired, got %s", status)
	}

	// Run second scan on same day -> should not send duplicates
	if err := scanner.RunScan(); err != nil {
		t.Fatalf("second RunScan failed: %v", err)
	}
	if sender.sentCount != 1 {
		t.Fatalf("expected email count to remain 1, got %d", sender.sentCount)
	}
}

