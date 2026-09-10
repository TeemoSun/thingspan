package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"thingspan/internal/config"
	"thingspan/internal/models"
)

type ReminderScanner struct {
	db     *sql.DB
	cfg    *config.Config
	sender EmailSender
}

func NewReminderScanner(db *sql.DB, cfg *config.Config, sender EmailSender) *ReminderScanner {
	return &ReminderScanner{
		db:     db,
		cfg:    cfg,
		sender: sender,
	}
}

func (rs *ReminderScanner) BuildEmailBodies(assetName, categoryName, targetDate string, daysLeft int, dateType string) (subject, htmlBody, plainBody string) {
	typeLabel := "到期"
	typeNoun := "资产"
	if dateType == "warranty" {
		typeLabel = "保修"
		typeNoun = "保修"
	}

	when := fmt.Sprintf("%d 天后到期", daysLeft)
	if daysLeft <= 0 {
		when = "已到期"
	}

	subject = fmt.Sprintf("【Thingspan】%s %s %s", assetName, typeLabel, when)

	plainBody = fmt.Sprintf(
		"【%s到期提醒】\n\n资产：%s\n类别：%s\n%s日期：%s\n距离：%s\n\n请及时处理。",
		typeNoun, assetName, categoryName, typeNoun, targetDate, when,
	)

	htmlBody = fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<style>
body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f4f5f7; margin: 0; padding: 24px; }
.card { max-width: 520px; margin: 0 auto; background: #ffffff; border-radius: 12px; box-shadow: 0 4px 12px rgba(0,0,0,0.06); padding: 32px; border: 1px solid #e2e8f0; }
.badge { display: inline-block; padding: 4px 10px; border-radius: 9999px; font-size: 12px; font-weight: 600; background: #fee2e2; color: #dc2626; margin-bottom: 16px; }
h2 { margin: 0 0 20px; font-size: 20px; color: #1e293b; }
.item { margin-bottom: 12px; font-size: 14px; line-height: 1.6; color: #475569; }
.item strong { color: #0f172a; width: 80px; display: inline-block; }
.footer { margin-top: 28px; padding-top: 16px; border-top: 1px solid #f1f5f9; font-size: 12px; color: #94a3b8; text-align: center; }
</style>
</head>
<body>
<div class="card">
  <div class="badge">Thingspan 资产提醒</div>
  <h2>%s%s</h2>
  <div class="item"><strong>资产名称：</strong>%s</div>
  <div class="item"><strong>所属类别：</strong>%s</div>
  <div class="item"><strong>%s日期：</strong>%s</div>
  <div class="item"><strong>临期状态：</strong><span style="color: #dc2626; font-weight: 600;">%s</span></div>
  <div class="footer">本邮件由 Thingspan 个人资产系统自动发出</div>
</div>
</body>
</html>`, typeNoun, when, assetName, categoryName, typeNoun, targetDate, when)

	return subject, htmlBody, plainBody
}

func (rs *ReminderScanner) RunScan() error {
	today := rs.cfg.TodayStr()
	localNowStr := rs.cfg.LocalNow().Format("2006-01-02 15:04:05")

	// 1. Send reminders
	rows, err := rs.db.Query(`
		SELECT a.id, a.name, a.purchase_date, a.purchase_price, a.warranty_months, a.warranty_end_date, a.expiry_date, a.status,
		       c.id, c.name, c.has_warranty, c.has_expiry, c.can_sell, c.can_break, c.has_serial, c.has_model
		FROM assets a
		JOIN categories c ON a.category_id = c.id
		WHERE a.status = ?
	`, models.StatusInUse)
	if err != nil {
		log.Printf("获取待提醒资产列表失败: %v", err)
		return err
	}
	defer rows.Close()

	type AssetWithCategory struct {
		Asset    models.Asset
		Category models.Category
	}
	var assets []AssetWithCategory

	for rows.Next() {
		var a models.Asset
		var c models.Category
		var icon, brand, model, serialNumber, warrantyEndDate, expiryDate, saleDate, brokenDate, notes sql.NullString
		var warrantyMonths sql.NullInt64
		var salePrice sql.NullFloat64

		if err := rows.Scan(
			&a.ID, &a.Name, &a.PurchaseDate, &a.PurchasePrice, &warrantyMonths, &warrantyEndDate, &expiryDate, &a.Status,
			&c.ID, &c.Name, &c.HasWarranty, &c.HasExpiry, &c.CanSell, &c.CanBreak, &c.HasSerial, &c.HasModel,
		); err != nil {
			log.Printf("扫描资产行失败: %v", err)
			continue
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
		_ = icon
		_ = brand
		_ = model
		_ = serialNumber
		_ = saleDate
		_ = salePrice
		_ = brokenDate
		_ = notes

		a.Category = &c
		assets = append(assets, AssetWithCategory{Asset: a, Category: c})
	}
	_ = rows.Close()

	for _, item := range assets {
		targets := ReminderTargetDates(&item.Asset, today)
		if len(targets) == 0 {
			continue
		}

		sentToday := false
		for _, target := range targets {
			if sentToday {
				break
			}

			for _, lead := range rs.cfg.LeadDays {
				maxTargetDate := AddDays(today, lead)
				if target.TargetDate > maxTargetDate {
					continue
				}

				// Check existing log
				var logID int64
				var sent bool
				err := rs.db.QueryRow(`
					SELECT id, sent FROM reminder_logs
					WHERE asset_id = ? AND target_date = ? AND lead_days = ?
				`, item.Asset.ID, target.TargetDate, lead).Scan(&logID, &sent)

				if err == nil && sent {
					// Already sent successfully for this tier
					break
				}

				daysLeft := DaysBetween(today, target.TargetDate)
				catName := item.Category.Name
				if catName == "" {
					catName = "-"
				}

				subject, htmlBody, plainBody := rs.BuildEmailBodies(item.Asset.Name, catName, target.TargetDate, daysLeft, target.DateType)
				ok := rs.sender.SendEmail(subject, htmlBody, plainBody)

				if logID > 0 {
					if ok {
						_, _ = rs.db.Exec(`UPDATE reminder_logs SET sent = 1 WHERE id = ?`, logID)
					}
				} else {
					_, _ = rs.db.Exec(`
						INSERT INTO reminder_logs (asset_id, target_date, lead_days, sent_at, sent, dismissed)
						VALUES (?, ?, ?, ?, ?, 0)
					`, item.Asset.ID, target.TargetDate, lead, localNowStr, ok)
				}

				sentToday = true
				break
			}
		}
	}

	// 2. Automatically mark expired assets
	res, err := rs.db.Exec(`
		UPDATE assets
		SET status = ?, updated_at = ?
		WHERE status = ?
		  AND expiry_date IS NOT NULL
		  AND expiry_date < ?
		  AND category_id IN (SELECT id FROM categories WHERE has_expiry = 1)
	`, models.StatusExpired, localNowStr, models.StatusInUse, today)
	if err != nil {
		log.Printf("自动标记过期资产失败: %v", err)
	} else if count, _ := res.RowsAffected(); count > 0 {
		log.Printf("已自动标记 %d 件资产为已过期", count)
	}

	return nil
}

// DismissReminder marks a reminder log as dismissed
func (rs *ReminderScanner) DismissReminder(reminderID int64) (*models.ReminderLog, error) {
	res, err := rs.db.Exec(`UPDATE reminder_logs SET dismissed = 1 WHERE id = ?`, reminderID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, sql.ErrNoRows
	}

	var l models.ReminderLog
	var assetName sql.NullString
	var sentAtStr string
	err = rs.db.QueryRow(`
		SELECT r.id, r.asset_id, COALESCE(a.name, ''), r.target_date, r.lead_days, r.sent_at, r.sent, r.dismissed
		FROM reminder_logs r
		LEFT JOIN assets a ON r.asset_id = a.id
		WHERE r.id = ?
	`, reminderID).Scan(&l.ID, &l.AssetID, &assetName, &l.TargetDate, &l.LeadDays, &sentAtStr, &l.Sent, &l.Dismissed)
	if err != nil {
		return nil, err
	}
	l.AssetName = assetName.String
	// Parse sent_at
	t, _ := time.Parse("2006-01-02 15:04:05", strings.Split(sentAtStr, ".")[0])
	l.SentAt = t
	return &l, nil
}

