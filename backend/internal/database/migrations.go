package database

import (
	"database/sql"
	"fmt"
	"strings"
)

func RunMigrations(db *sql.DB) error {
	// 1. Ensure basic tables exist
	createTablesSQL := `
	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name VARCHAR(100) NOT NULL UNIQUE,
		has_warranty BOOLEAN NOT NULL DEFAULT 0,
		has_expiry BOOLEAN NOT NULL DEFAULT 0,
		can_sell BOOLEAN NOT NULL DEFAULT 0,
		can_break BOOLEAN NOT NULL DEFAULT 0,
		has_serial BOOLEAN NOT NULL DEFAULT 0,
		has_model BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS assets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		category_id INTEGER NOT NULL REFERENCES categories(id),
		name VARCHAR(200) NOT NULL,
		icon VARCHAR(50) NULL,
		brand VARCHAR(100) NULL,
		model VARCHAR(100) NULL,
		serial_number VARCHAR(100) NULL,
		purchase_date DATE NOT NULL,
		purchase_price REAL NOT NULL,
		warranty_months INTEGER NULL,
		warranty_end_date DATE NULL,
		expiry_date DATE NULL,
		status VARCHAR(20) NOT NULL DEFAULT 'in_use',
		sale_date DATE NULL,
		sale_price REAL NULL,
		broken_date DATE NULL,
		notes TEXT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL
	);

	CREATE TABLE IF NOT EXISTS reminder_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		asset_id INTEGER NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
		target_date DATE NOT NULL,
		lead_days INTEGER NOT NULL,
		sent_at DATETIME NOT NULL,
		sent BOOLEAN NOT NULL DEFAULT 1,
		dismissed BOOLEAN NOT NULL DEFAULT 0,
		UNIQUE(asset_id, target_date, lead_days)
	);

	CREATE INDEX IF NOT EXISTS idx_assets_category_id ON assets(category_id);
	CREATE INDEX IF NOT EXISTS idx_assets_status ON assets(status);
	CREATE INDEX IF NOT EXISTS idx_reminder_logs_asset_id ON reminder_logs(asset_id);
	`

	if _, err := db.Exec(createTablesSQL); err != nil {
		return fmt.Errorf("failed to create base tables: %w", err)
	}

	// 2. Compatibility check for older databases (column checks)
	// Check categories columns
	categoryCols, err := getTableColumns(db, "categories")
	if err != nil {
		return err
	}
	catMissing := map[string]string{
		"has_warranty": "BOOLEAN NOT NULL DEFAULT 0",
		"has_expiry":   "BOOLEAN NOT NULL DEFAULT 0",
		"can_sell":     "BOOLEAN NOT NULL DEFAULT 0",
		"can_break":    "BOOLEAN NOT NULL DEFAULT 0",
		"has_serial":   "BOOLEAN NOT NULL DEFAULT 0",
		"has_model":    "BOOLEAN NOT NULL DEFAULT 0",
	}
	for col, colDef := range catMissing {
		if !categoryCols[col] {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE categories ADD COLUMN %s %s;", col, colDef)); err != nil {
				return fmt.Errorf("failed to add column %s to categories: %w", col, err)
			}
		}
	}

	// Check assets columns
	assetCols, err := getTableColumns(db, "assets")
	if err != nil {
		return err
	}
	assetMissing := map[string]string{
		"icon":            "VARCHAR(50) NULL",
		"warranty_months": "INTEGER NULL",
	}
	for col, colDef := range assetMissing {
		if !assetCols[col] {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE assets ADD COLUMN %s %s;", col, colDef)); err != nil {
				return fmt.Errorf("failed to add column %s to assets: %w", col, err)
			}
		}
	}

	// Check reminder_logs columns
	reminderCols, err := getTableColumns(db, "reminder_logs")
	if err != nil {
		return err
	}
	if !reminderCols["sent"] {
		if _, err := db.Exec("ALTER TABLE reminder_logs ADD COLUMN sent BOOLEAN NOT NULL DEFAULT 1;"); err != nil {
			return fmt.Errorf("failed to add column sent to reminder_logs: %w", err)
		}
	}

	return nil
}

func getTableColumns(db *sql.DB, tableName string) (map[string]bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s);", tableName))
	if err != nil {
		return nil, fmt.Errorf("failed to query table_info for %s: %w", tableName, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		cols[strings.ToLower(name)] = true
	}
	return cols, rows.Err()
}

