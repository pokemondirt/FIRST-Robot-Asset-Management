package main

import (
	"net/http"
	"sort"
	"time"
)

// startOfToday returns the start of the current LOCAL day converted to UTC.
// created_at values are stored in UTC (nowSQL), so comparing them against a
// naive local-midnight string would misattribute early-morning transactions
// (e.g. 00:00-08:00 in UTC+8) to the wrong day.
func startOfToday() time.Time {
	now := time.Now()
	localMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return localMidnight.UTC()
}

func startOfWeek() time.Time {
	now := time.Now()
	localToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekday := localToday.Weekday()
	offset := int(weekday)
	if offset == 0 {
		offset = 6
	} else {
		offset--
	}
	return localToday.AddDate(0, 0, -offset).UTC()
}

// GET /api/dashboard
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	startToday := startOfToday()
	startWeek := startOfWeek()

	todayStr := startToday.Format("2006-01-02 15:04:05")
	weekStr := startWeek.Format("2006-01-02 15:04:05")

	// Today inbound count
	var todayTxCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM transactions
		WHERE type = ? AND created_at >= ?
	`, TxIn, todayStr).Scan(&todayTxCount); err != nil {
		todayTxCount = 0
	}

	// Today inbound quantity
	var todayQty int
	if err := db.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0) FROM transactions
		WHERE type = ? AND created_at >= ?
	`, TxIn, todayStr).Scan(&todayQty); err != nil {
		todayQty = 0
	}

	// Week inbound quantity
	var weekQty int
	if err := db.QueryRow(`
		SELECT COALESCE(SUM(quantity), 0) FROM transactions
		WHERE type = ? AND created_at >= ?
	`, TxIn, weekStr).Scan(&weekQty); err != nil {
		weekQty = 0
	}

	// Recent inbound
	rows, err := db.Query(`
		SELECT t.id, i.barcode, i.name_zh, i.name_en, t.quantity, o.display_name, t.created_at
		FROM transactions t
		JOIN items i ON i.id = t.item_id
		LEFT JOIN operators o ON o.id = t.operator_id
		WHERE t.type = ?
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT 15
	`, TxIn)
	recentLines := []RecentInboundLine{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var line RecentInboundLine
			var opName *string
			var createdAtStr string
			if err := rows.Scan(&line.ID, &line.Barcode, &line.NameZh, &line.NameEn,
				&line.Quantity, &opName, &createdAtStr); err != nil {
				continue
			}
			line.OperatorName = opName
			line.CreatedAt = parseTime(createdAtStr)
			recentLines = append(recentLines, line)
		}
	}
	if recentLines == nil {
		recentLines = []RecentInboundLine{}
	}

	// Low stock alerts
	itemRows, err := db.Query(`
		SELECT i.id, i.barcode, i.name_zh, i.name_en, i.program, i.category,
		       COALESCE(s.quantity, 0), i.min_stock, i.unit
		FROM items i
		LEFT JOIN stock s ON s.item_id = i.id
		WHERE i.active = 1
	`)
	alerts := []LowStockAlert{}
	if err == nil {
		defer itemRows.Close()
		for itemRows.Next() {
			var (
				id       int
				barcode  string
				nameZh   string
				nameEn   string
				program  string
				category string
				qty      int
				minStock int
				unit     string
			)
			if err := itemRows.Scan(&id, &barcode, &nameZh, &nameEn, &program, &category,
				&qty, &minStock, &unit); err != nil {
				continue
			}
			isEmpty := qty <= 0
			isLow := minStock > 0 && qty < minStock
			if !isEmpty && !isLow {
				continue
			}

			level := "warning"
			if isEmpty {
				level = "critical"
			}

			shortage := 0
			if minStock > 0 {
				shortage = minStock - qty
				if shortage < 0 {
					shortage = 0
				}
			} else if isEmpty {
				shortage = 1
			}

			alerts = append(alerts, LowStockAlert{
				ID: id, Barcode: barcode, NameZh: nameZh, NameEn: nameEn,
				Program: program, Category: category,
				Quantity: qty, MinStock: minStock, Shortage: shortage,
				Unit: unit, Level: level,
			})
		}
	}

	sort.Slice(alerts, func(i, j int) bool {
		if alerts[i].Level != alerts[j].Level {
			return alerts[i].Level == "critical"
		}
		if alerts[i].Shortage != alerts[j].Shortage {
			return alerts[i].Shortage > alerts[j].Shortage
		}
		return alerts[i].Quantity < alerts[j].Quantity
	})
	if alerts == nil {
		alerts = []LowStockAlert{}
	}

	writeJSON(w, 200, DashboardResponse{
		GeneratedAt: time.Now(),
		Inbound: InboundSummary{
			TodayTransactions: todayTxCount,
			TodayQuantity:     todayQty,
			WeekQuantity:      weekQty,
		},
		RecentInbound:  recentLines,
		LowStockAlerts: alerts,
		AlertCount:     len(alerts),
	})
}
