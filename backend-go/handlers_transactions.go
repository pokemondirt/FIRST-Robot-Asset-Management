package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

const txSelectSQL = `
	SELECT t.id, t.type, t.item_id, t.quantity, t.note, t.created_at,
	       i.barcode, i.name_zh, i.name_en,
	       o.display_name
	FROM transactions t
	JOIN items i ON i.id = t.item_id
	LEFT JOIN operators o ON o.id = t.operator_id
`

func scanTxRow(scan func(...any) error, t *Transaction) (*string, error) {
	var opName *string
	var createdAtStr string
	err := scan(
		&t.ID, &t.Type, &t.ItemID, &t.Quantity, &t.Note, &createdAtStr,
		&t.Barcode, &t.NameZh, &t.NameEn,
		&opName,
	)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = parseTime(createdAtStr)
	return opName, nil
}

// POST /api/transactions
func handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	var req TransactionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	// Only clamp non-ADJUST quantities: ADJUST sets an absolute stock value,
	// so 0 (zero out) and other exact values must pass through unchanged.
	if req.Type != TxAdjust && req.Quantity < 1 {
		req.Quantity = 1
	}

	// Find item
	var (
		itemID    int
		trackMode string
		active    int
	)
	err := db.QueryRow(`
		SELECT i.id, i.track_mode, i.active
		FROM items i
		WHERE i.barcode = ?
	`, req.Barcode).Scan(&itemID, &trackMode, &active)
	if err != nil {
		writeError(w, 400, map[string]string{"code": "NOT_FOUND", "message": "Item not found"})
		return
	}
	if active == 0 {
		writeError(w, 400, map[string]string{"code": "NOT_FOUND", "message": "Item not found"})
		return
	}

	qty := req.Quantity

	// SNP mode: force qty=1 for IN/OUT/RETURN
	if trackMode == TrackSNP && (req.Type == TxIn || req.Type == TxOut || req.Type == TxReturn) {
		qty = 1
	}

	// Validate quantity for non-adjust
	if req.Type != TxAdjust && qty < 1 {
		writeError(w, 400, map[string]string{"code": "INVALID_QUANTITY", "message": "Quantity must be at least 1"})
		return
	}

	tx, err := db.Begin()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	// Read current stock within transaction
	var currentQty int
	if err := tx.QueryRow("SELECT quantity FROM stock WHERE item_id = ?", itemID).Scan(&currentQty); err != nil {
		tx.Exec("INSERT OR IGNORE INTO stock (item_id, quantity) VALUES (?, 0)", itemID)
		currentQty = 0
	}

	switch req.Type {
	case TxOut:
		if currentQty < qty {
			writeError(w, 400, map[string]string{
				"code":    "INSUFFICIENT_STOCK",
				"message": "Insufficient stock (have " + strconv.Itoa(currentQty) + ", need " + strconv.Itoa(qty) + ")",
			})
			return
		}
		if _, err := tx.Exec("UPDATE stock SET quantity = quantity - ? WHERE item_id = ?", qty, itemID); err != nil {
			writeError(w, 500, err.Error())
			return
		}
	case TxIn, TxReturn:
		if _, err := tx.Exec("UPDATE stock SET quantity = quantity + ? WHERE item_id = ?", qty, itemID); err != nil {
			writeError(w, 500, err.Error())
			return
		}
	case TxAdjust:
		if qty < 0 {
			writeError(w, 400, map[string]string{
				"code":    "INVALID_QUANTITY",
				"message": "Adjust quantity cannot be negative",
			})
			return
		}
		if _, err := tx.Exec("UPDATE stock SET quantity = ? WHERE item_id = ?", qty, itemID); err != nil {
			writeError(w, 500, err.Error())
			return
		}
	default:
		writeError(w, 400, map[string]string{"code": "INVALID_TYPE", "message": "Invalid transaction type"})
		return
	}

	result, err := tx.Exec(`
		INSERT INTO transactions (type, item_id, quantity, operator_id, note, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, req.Type, itemID, qty, req.OperatorID, strings.TrimSpace(req.Note), nowSQL())
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	txID, _ := result.LastInsertId()

	var t Transaction
	opName, _ := scanTxRow(
		db.QueryRow(txSelectSQL+" WHERE t.id = ?", txID).Scan,
		&t,
	)
	t.OperatorName = opName
	writeJSON(w, 200, t)
}

// GET /api/transactions
func handleListTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	itemIDStr := q.Get("item_id")
	txType := q.Get("type")

	where := "WHERE 1=1"
	args := []any{}
	if itemIDStr != "" {
		if id, err := strconv.Atoi(itemIDStr); err == nil {
			where += " AND t.item_id = ?"
			args = append(args, id)
		}
	}
	if txType != "" {
		where += " AND t.type = ?"
		args = append(args, txType)
	}

	var total int
	if err := db.QueryRow("SELECT COUNT(*) FROM transactions t "+where, args...).Scan(&total); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	selectArgs := append(args, pageSize, (page-1)*pageSize)
	rows, err := db.Query(txSelectSQL+where+` ORDER BY t.created_at DESC LIMIT ? OFFSET ?`, selectArgs...)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	transactions := []Transaction{}
	for rows.Next() {
		var t Transaction
		opName, err := scanTxRow(rows.Scan, &t)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		t.OperatorName = opName
		transactions = append(transactions, t)
	}
	if transactions == nil {
		transactions = []Transaction{}
	}

	writeJSON(w, 200, TransactionListResponse{
		Transactions: transactions, Total: total, Page: page, PageSize: pageSize,
	})
}

// POST /api/transactions/batch
func handleBatchTransactions(w http.ResponseWriter, r *http.Request) {
	var req BatchTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	if len(req.Lines) == 0 {
		writeError(w, 400, map[string]string{"code": "EMPTY_LINES", "message": "No transaction lines provided"})
		return
	}

	dbTx, err := db.Begin()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer dbTx.Rollback()

	results := []Transaction{}
	errorLines := []BatchTransactionLine{}

	for _, line := range req.Lines {
		// Find item
		var (
			itemID    int
			trackMode string
			active    int
		)
		err := dbTx.QueryRow(`
			SELECT i.id, i.track_mode, i.active
			FROM items i
			WHERE i.barcode = ?
		`, line.Barcode).Scan(&itemID, &trackMode, &active)
		if err != nil || active == 0 {
			errorLines = append(errorLines, line)
			continue
		}

		qty := line.Quantity
		if qty < 1 {
			qty = 1
		}

		// SNP mode: force qty=1 for IN/OUT/RETURN
		if trackMode == TrackSNP && (line.Type == TxIn || line.Type == TxOut || line.Type == TxReturn) {
			qty = 1
		}

		// Read current stock
		var currentQty int
		if err := dbTx.QueryRow("SELECT quantity FROM stock WHERE item_id = ?", itemID).Scan(&currentQty); err != nil {
			dbTx.Exec("INSERT OR IGNORE INTO stock (item_id, quantity) VALUES (?, 0)", itemID)
			currentQty = 0
		}

		switch line.Type {
		case TxOut:
			if currentQty < qty {
				errorLines = append(errorLines, line)
				continue
			}
			if _, err := dbTx.Exec("UPDATE stock SET quantity = quantity - ? WHERE item_id = ?", qty, itemID); err != nil {
				errorLines = append(errorLines, line)
				continue
			}
		case TxIn, TxReturn:
			if _, err := dbTx.Exec("UPDATE stock SET quantity = quantity + ? WHERE item_id = ?", qty, itemID); err != nil {
				errorLines = append(errorLines, line)
				continue
			}
		default:
			errorLines = append(errorLines, line)
			continue
		}

		result, err := dbTx.Exec(`
			INSERT INTO transactions (type, item_id, quantity, operator_id, note, created_at)
			VALUES (?, ?, ?, ?, ?, ?)
		`, line.Type, itemID, qty, req.OperatorID, strings.TrimSpace(req.Note), nowSQL())
		if err != nil {
			errorLines = append(errorLines, line)
			continue
		}

		txID, _ := result.LastInsertId()

		var t Transaction
		opName, _ := scanTxRow(
			dbTx.QueryRow(txSelectSQL+" WHERE t.id = ?", txID).Scan,
			&t,
		)
		t.OperatorName = opName
		results = append(results, t)
	}

	if len(errorLines) > 0 && len(results) == 0 {
		writeError(w, 400, map[string]interface{}{
			"code":    "ALL_FAILED",
			"message": "All transaction lines failed",
		})
		return
	}

	if err := dbTx.Commit(); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	if results == nil {
		results = []Transaction{}
	}

	writeJSON(w, 200, BatchTransactionResult{
		Success:      len(errorLines) == 0,
		Transactions: results,
		Errors:       errorLines,
	})
}
