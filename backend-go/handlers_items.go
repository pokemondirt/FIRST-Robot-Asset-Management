package main

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

func validProgram(p string) bool {
	return p == ProgramFRC || p == ProgramFTC || p == ProgramBOTH
}

func validTrackMode(m string) bool {
	return m == TrackSNP || m == TrackBLK
}

// scanItemFields scans an item row and returns location code string
func scanItemFields(scan func(...any) error, it *Item) (*string, error) {
	var locCode *string
	var activeInt int
	var createdAtStr string
	err := scan(
		&it.ID, &it.Barcode, &it.Program, &it.TrackMode,
		&it.NameZh, &it.NameEn, &it.Category, &it.Spec, &it.Unit,
		&it.MinStock, &it.LocationID, &it.Note, &activeInt, &createdAtStr,
		&it.Quantity, &locCode,
	)
	if err != nil {
		return nil, err
	}
	it.Active = activeInt == 1
	it.CreatedAt = parseTime(createdAtStr)
	return locCode, nil
}

const itemSelectSQL = `
	SELECT i.id, i.barcode, i.program, i.track_mode, i.name_zh, i.name_en,
	       i.category, i.spec, i.unit, i.min_stock, i.location_id, i.note, i.active, i.created_at,
	       COALESCE(s.quantity, 0),
	       l.code
	FROM items i
	LEFT JOIN stock s ON s.item_id = i.id
	LEFT JOIN locations l ON l.id = i.location_id
`

// GET /api/items
func handleListItems(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	if pageSize < 1 {
		pageSize = 50
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	program := q.Get("program")
	category := q.Get("category")
	trackMode := q.Get("track_mode")
	search := q.Get("q")
	activeOnly := q.Get("active_only") != "false"

	where := "WHERE 1=1"
	args := []any{}
	if activeOnly {
		where += " AND i.active = 1"
	}
	if program != "" {
		where += " AND i.program = ?"
		args = append(args, program)
	}
	if category != "" {
		where += " AND i.category = ?"
		args = append(args, category)
	}
	if trackMode != "" {
		where += " AND i.track_mode = ?"
		args = append(args, trackMode)
	}
	if search != "" {
		// ESCAPE keeps % and _ in the user's text literal instead of letting
		// them match every row.
		where += ` AND (i.barcode LIKE ? ESCAPE '\' OR i.name_zh LIKE ? ESCAPE '\' OR i.name_en LIKE ? ESCAPE '\')`
		pattern := likePattern(search)
		args = append(args, pattern, pattern, pattern)
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM items i " + where
	if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	selectQuery := itemSelectSQL + where + ` ORDER BY i.id DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, (page-1)*pageSize)

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		locCode, err := scanItemFields(rows.Scan, &it)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		it.LocationCode = locCode
		items = append(items, it)
	}
	if items == nil {
		items = []Item{}
	}

	writeJSON(w, 200, ItemListResponse{
		Items: items, Total: total, Page: page, PageSize: pageSize,
	})
}

// GET /api/items/{id}
func handleGetItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	var it Item
	locCode, err := scanItemFields(
		db.QueryRow(itemSelectSQL+" WHERE i.id = ?", id).Scan,
		&it,
	)
	if err != nil {
		writeError(w, 404, "Item not found")
		return
	}
	it.LocationCode = locCode
	writeJSON(w, 200, it)
}

// POST /api/items
func handleCreateItem(w http.ResponseWriter, r *http.Request) {
	var req ItemCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	// Trim before validating: a barcode stored as "  ABC  " can never be found
	// again, because lookups trim the scanned code.
	req.Barcode = strings.TrimSpace(req.Barcode)
	req.Program = strings.TrimSpace(req.Program)
	req.TrackMode = strings.TrimSpace(req.TrackMode)
	req.NameZh = strings.TrimSpace(req.NameZh)
	req.NameEn = strings.TrimSpace(req.NameEn)
	req.Category = strings.TrimSpace(req.Category)
	req.Spec = strings.TrimSpace(req.Spec)
	req.Unit = strings.TrimSpace(req.Unit)
	req.Note = strings.TrimSpace(req.Note)

	if req.Program == "" {
		req.Program = ProgramBOTH
	} else if !validProgram(req.Program) {
		writeError(w, 400, "invalid program")
		return
	}
	if req.TrackMode == "" {
		req.TrackMode = TrackBLK
	} else if !validTrackMode(req.TrackMode) {
		writeError(w, 400, "invalid track_mode")
		return
	}
	if req.Category == "" {
		req.Category = "MISC"
	}
	if req.Unit == "" {
		req.Unit = "pcs"
	}
	if req.NameZh == "" && req.NameEn == "" {
		writeError(w, 400, "name_zh or name_en required")
		return
	}
	if req.NameZh == "" {
		req.NameZh = req.NameEn
	}
	if req.NameEn == "" {
		req.NameEn = req.NameZh
	}
	if req.MinStock < 0 {
		writeError(w, 400, "min_stock cannot be negative")
		return
	}
	if req.QuantityInit < 0 {
		writeError(w, 400, "quantity_initial cannot be negative")
		return
	}
	if !checkLengths(w,
		fieldLimit{"barcode", req.Barcode, MaxBarcodeLen},
		fieldLimit{"name_zh", req.NameZh, MaxNameLen},
		fieldLimit{"name_en", req.NameEn, MaxNameLen},
		fieldLimit{"category", req.Category, MaxCategoryLen},
		fieldLimit{"spec", req.Spec, MaxSpecLen},
		fieldLimit{"unit", req.Unit, MaxUnitLen},
		fieldLimit{"note", req.Note, MaxNoteLen},
	) {
		return
	}
	if req.LocationID != nil {
		if msg := checkLocationExists(db, *req.LocationID); msg != "" {
			writeError(w, 400, msg)
			return
		}
	}

	active := 1
	if req.Active != nil && !*req.Active {
		active = 0
	}

	// Auto-generated barcodes use a max+1 scheme that is not atomic; retry
	// on UNIQUE collisions instead of failing a legitimate concurrent create.
	barcode := req.Barcode
	autoBarcode := barcode == ""
	const maxBarcodeAttempts = 8
	var err error
	var result sql.Result
	for attempt := 0; ; attempt++ {
		if barcode == "" {
			barcode = nextBarcode(db, req.Program, req.TrackMode)
		}
		result, err = db.Exec(`
			INSERT INTO items (barcode, program, track_mode, name_zh, name_en, category, spec, unit, min_stock, location_id, note, active, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, barcode, req.Program, req.TrackMode, req.NameZh, req.NameEn,
			req.Category, req.Spec, req.Unit, req.MinStock, req.LocationID, req.Note, active, nowSQL())
		if err == nil {
			break
		}
		if strings.Contains(err.Error(), "UNIQUE") {
			if autoBarcode && attempt < maxBarcodeAttempts-1 {
				barcode = ""
				continue
			}
			writeError(w, 400, "Barcode already exists: "+barcode)
			return
		}
		writeError(w, 500, err.Error())
		return
	}

	itemID, _ := result.LastInsertId()
	qty := req.QuantityInit
	if qty < 0 {
		qty = 0
	}
	db.Exec("INSERT OR IGNORE INTO stock (item_id, quantity) VALUES (?, ?)", itemID, qty)

	var it Item
	locCode, _ := scanItemFields(
		db.QueryRow(itemSelectSQL+" WHERE i.id = ?", itemID).Scan,
		&it,
	)
	it.LocationCode = locCode
	writeJSON(w, 200, it)
}

// PATCH /api/items/{id}
func handleUpdateItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	var req ItemUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	if req.Program != nil && !validProgram(*req.Program) {
		writeError(w, 400, "invalid program")
		return
	}
	if req.TrackMode != nil && !validTrackMode(*req.TrackMode) {
		writeError(w, 400, "invalid track_mode")
		return
	}

	// Load the current row first. This both proves the item exists (so patching
	// a missing id is a 404 instead of the misleading "no fields to update")
	// and supplies the values used to detect a program/mode change.
	var curProgram, curMode string
	if err := db.QueryRow("SELECT program, track_mode FROM items WHERE id = ?", id).Scan(&curProgram, &curMode); err != nil {
		writeError(w, 404, "Item not found")
		return
	}

	trimmed := func(p *string) *string {
		if p == nil {
			return nil
		}
		v := strings.TrimSpace(*p)
		return &v
	}
	req.NameZh = trimmed(req.NameZh)
	req.NameEn = trimmed(req.NameEn)
	req.Category = trimmed(req.Category)
	req.Spec = trimmed(req.Spec)
	req.Unit = trimmed(req.Unit)
	req.Note = trimmed(req.Note)

	if req.NameZh != nil && *req.NameZh == "" && (req.NameEn == nil || *req.NameEn == "") {
		writeError(w, 400, "name_zh or name_en required")
		return
	}
	if req.MinStock != nil && *req.MinStock < 0 {
		writeError(w, 400, "min_stock cannot be negative")
		return
	}
	if req.LocationID.Set && !req.LocationID.NullValue() {
		if msg := checkLocationExists(db, req.LocationID.Value); msg != "" {
			writeError(w, 400, msg)
			return
		}
	}
	stringLimits := []fieldLimit{}
	addLimit := func(field string, p *string, max int) {
		if p != nil {
			stringLimits = append(stringLimits, fieldLimit{field, *p, max})
		}
	}
	addLimit("name_zh", req.NameZh, MaxNameLen)
	addLimit("name_en", req.NameEn, MaxNameLen)
	addLimit("category", req.Category, MaxCategoryLen)
	addLimit("spec", req.Spec, MaxSpecLen)
	addLimit("unit", req.Unit, MaxUnitLen)
	addLimit("note", req.Note, MaxNoteLen)
	if !checkLengths(w, stringLimits...) {
		return
	}

	sets := []string{}
	args := []any{}
	if req.Program != nil {
		sets = append(sets, "program = ?")
		args = append(args, *req.Program)
	}
	if req.TrackMode != nil {
		sets = append(sets, "track_mode = ?")
		args = append(args, *req.TrackMode)
	}
	// Regenerate the barcode whenever program or track_mode actually changes,
	// using the NEW values so the barcode prefix always matches the item.
	if req.Program != nil || req.TrackMode != nil {
		newProgram, newMode := curProgram, curMode
		changed := false
		if req.Program != nil && *req.Program != curProgram {
			newProgram = *req.Program
			changed = true
		}
		if req.TrackMode != nil && *req.TrackMode != curMode {
			newMode = *req.TrackMode
			changed = true
		}
		if changed {
			sets = append(sets, "barcode = ?")
			args = append(args, nextBarcode(db, newProgram, newMode))
		}
	}
	if req.NameZh != nil {
		sets = append(sets, "name_zh = ?")
		args = append(args, *req.NameZh)
	}
	if req.NameEn != nil {
		sets = append(sets, "name_en = ?")
		args = append(args, *req.NameEn)
	}
	if req.Category != nil {
		sets = append(sets, "category = ?")
		args = append(args, *req.Category)
	}
	if req.Spec != nil {
		sets = append(sets, "spec = ?")
		args = append(args, *req.Spec)
	}
	if req.Unit != nil {
		sets = append(sets, "unit = ?")
		args = append(args, *req.Unit)
	}
	if req.MinStock != nil {
		sets = append(sets, "min_stock = ?")
		args = append(args, *req.MinStock)
	}
	if req.LocationID.Set {
		sets = append(sets, "location_id = ?")
		if req.LocationID.NullValue() {
			args = append(args, nil)
		} else {
			args = append(args, req.LocationID.Value)
		}
	}
	if req.Note != nil {
		sets = append(sets, "note = ?")
		args = append(args, *req.Note)
	}
	if req.Active != nil {
		active := 0
		if *req.Active {
			active = 1
		}
		sets = append(sets, "active = ?")
		args = append(args, active)
	}

	if len(sets) == 0 {
		writeError(w, 400, "no fields to update")
		return
	}

	args = append(args, id)
	query := "UPDATE items SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := db.Exec(query, args...); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	var it Item
	locCode, err := scanItemFields(
		db.QueryRow(itemSelectSQL+" WHERE i.id = ?", id).Scan,
		&it,
	)
	if err != nil {
		writeError(w, 404, "Item not found")
		return
	}
	it.LocationCode = locCode
	writeJSON(w, 200, it)
}

// DELETE /api/items/{id}
func handleDeleteItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	result, err := db.Exec("UPDATE items SET active = 0 WHERE id = ?", id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, 404, "Item not found")
		return
	}
	writeJSON(w, 200, OKResponse{OK: true})
}

// DELETE /api/items/{id}/purge
// Permanently removes an item together with its stock row and all
// transaction history. Unlike the soft delete (handleDeleteItem), this is
// irreversible.
func handlePurgeItem(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	// Require the purge password before any destructive action.
	var req PurgeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	if !purgePasswordConfigured() {
		writeError(w, 400, "尚未设置删除保护密码，请先在设置页设置")
		return
	}
	if !verifyPurgePassword(req.Password) {
		writeError(w, 403, "密码错误")
		return
	}

	tx, err := db.Begin()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	// Delete in FK order: transactions and stock reference items.
	if _, err := tx.Exec("DELETE FROM transactions WHERE item_id = ?", id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if _, err := tx.Exec("DELETE FROM stock WHERE item_id = ?", id); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	result, err := tx.Exec("DELETE FROM items WHERE id = ?", id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, 404, "Item not found")
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	writeJSON(w, 200, OKResponse{OK: true})
}

// GET /api/items/lookup/{code}
func handleLookupCode(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.PathValue("code"))

	var it Item
	locCode, err := scanItemFields(
		db.QueryRow(itemSelectSQL+" WHERE i.barcode = ? AND i.active = 1", code).Scan,
		&it,
	)
	if err == nil {
		it.LocationCode = locCode
		writeJSON(w, 200, ScanLookupResponse{Kind: "item", Item: &it})
		return
	}

	var loc Location
	err = db.QueryRow("SELECT id, code, name_zh, name_en FROM locations WHERE code = ?", code).Scan(
		&loc.ID, &loc.Code, &loc.NameZh, &loc.NameEn,
	)
	if err == nil {
		writeJSON(w, 200, ScanLookupResponse{Kind: "location", Location: &loc})
		return
	}

	writeJSON(w, 200, ScanLookupResponse{Kind: "unknown"})
}

// POST /api/items/next-barcode
func handleNextBarcode(w http.ResponseWriter, r *http.Request) {
	program := r.URL.Query().Get("program")
	trackMode := r.URL.Query().Get("track_mode")
	if program == "" {
		program = ProgramBOTH
	}
	if trackMode == "" {
		trackMode = TrackBLK
	}
	writeJSON(w, 200, NextBarcodeResponse{Barcode: nextBarcode(db, program, trackMode)})
}
