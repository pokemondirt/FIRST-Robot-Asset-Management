package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// --- Locations ---

// GET /api/locations
func handleListLocations(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, code, name_zh, name_en FROM locations ORDER BY code")
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	locs := []Location{}
	for rows.Next() {
		var loc Location
		if err := rows.Scan(&loc.ID, &loc.Code, &loc.NameZh, &loc.NameEn); err != nil {
			continue
		}
		locs = append(locs, loc)
	}
	if locs == nil {
		locs = []Location{}
	}
	writeJSON(w, 200, locs)
}

// POST /api/locations
func handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	var req LocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	req.Code = strings.TrimSpace(req.Code)
	req.NameZh = strings.TrimSpace(req.NameZh)
	req.NameEn = strings.TrimSpace(req.NameEn)
	if req.Code == "" {
		writeError(w, 400, "code required")
		return
	}
	if !checkLengths(w,
		fieldLimit{"code", req.Code, MaxCodeLen},
		fieldLimit{"name_zh", req.NameZh, MaxLocationNameLen},
		fieldLimit{"name_en", req.NameEn, MaxLocationNameLen},
	) {
		return
	}

	// Case-insensitive: "A-01" and "a-01" are the same shelf to a human, and
	// the barcode-like codes are typed by hand.
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM locations WHERE code = ? COLLATE NOCASE", req.Code).Scan(&exists)
	if exists > 0 {
		writeError(w, 400, "Location code exists")
		return
	}

	result, err := db.Exec(
		"INSERT INTO locations (code, name_zh, name_en) VALUES (?, ?, ?)",
		req.Code, req.NameZh, req.NameEn,
	)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	id, _ := result.LastInsertId()

	var loc Location
	db.QueryRow("SELECT id, code, name_zh, name_en FROM locations WHERE id = ?", id).Scan(
		&loc.ID, &loc.Code, &loc.NameZh, &loc.NameEn,
	)
	writeJSON(w, 200, loc)
}

// PATCH /api/locations/{id}
func handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	var req LocationUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	var found int
	if err := db.QueryRow("SELECT COUNT(*) FROM locations WHERE id = ?", id).Scan(&found); err != nil || found == 0 {
		writeError(w, 404, "Location not found")
		return
	}

	sets := []string{}
	args := []any{}
	limits := []fieldLimit{}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" {
			writeError(w, 400, "code required")
			return
		}
		limits = append(limits, fieldLimit{"code", code, MaxCodeLen})
		// Case-insensitive duplicate check (exclude self).
		var dup int
		db.QueryRow("SELECT COUNT(*) FROM locations WHERE code = ? COLLATE NOCASE AND id != ?", code, id).Scan(&dup)
		if dup > 0 {
			writeError(w, 400, "Location code exists")
			return
		}
		sets = append(sets, "code = ?")
		args = append(args, code)
	}
	// An empty name is a legitimate value here: that is how a name is cleared.
	// The previous `if req.NameZh != ""` test made clearing impossible while
	// the endpoint still answered 200.
	if req.NameZh != nil {
		name := strings.TrimSpace(*req.NameZh)
		limits = append(limits, fieldLimit{"name_zh", name, MaxLocationNameLen})
		sets = append(sets, "name_zh = ?")
		args = append(args, name)
	}
	if req.NameEn != nil {
		name := strings.TrimSpace(*req.NameEn)
		limits = append(limits, fieldLimit{"name_en", name, MaxLocationNameLen})
		sets = append(sets, "name_en = ?")
		args = append(args, name)
	}

	if !checkLengths(w, limits...) {
		return
	}
	if len(sets) == 0 {
		writeError(w, 400, "no fields to update")
		return
	}

	args = append(args, id)
	query := "UPDATE locations SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	if _, err := db.Exec(query, args...); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	var loc Location
	if err := db.QueryRow("SELECT id, code, name_zh, name_en FROM locations WHERE id = ?", id).Scan(
		&loc.ID, &loc.Code, &loc.NameZh, &loc.NameEn,
	); err != nil {
		writeError(w, 404, "Location not found")
		return
	}
	writeJSON(w, 200, loc)
}

// DELETE /api/locations/{id}
func handleDeleteLocation(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	// Check if any items reference this location
	var refCount int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE location_id = ?", id).Scan(&refCount)
	if refCount > 0 {
		writeError(w, 400, map[string]any{
			"code":    "LOCATION_IN_USE",
			"message": "Location in use by " + strconv.Itoa(refCount) + " item(s)",
			"count":   refCount,
		})
		return
	}

	result, err := db.Exec("DELETE FROM locations WHERE id = ?", id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, 404, "Location not found")
		return
	}
	writeJSON(w, 200, OKResponse{OK: true})
}

// --- Operators ---

// GET /api/operators
func handleListOperators(w http.ResponseWriter, r *http.Request) {
	activeOnly := r.URL.Query().Get("active_only") != "false"

	query := "SELECT id, display_name, active FROM operators"
	args := []any{}
	if activeOnly {
		query += " WHERE active = 1"
	}
	query += " ORDER BY display_name"

	rows, err := db.Query(query, args...)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	ops := []Operator{}
	for rows.Next() {
		var op Operator
		var activeInt int
		if err := rows.Scan(&op.ID, &op.DisplayName, &activeInt); err != nil {
			continue
		}
		op.Active = activeInt == 1
		ops = append(ops, op)
	}
	if ops == nil {
		ops = []Operator{}
	}
	writeJSON(w, 200, ops)
}

// POST /api/operators
func handleCreateOperator(w http.ResponseWriter, r *http.Request) {
	var req OperatorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		writeError(w, 400, "Name required")
		return
	}
	if !checkLengths(w, fieldLimit{"display_name", req.DisplayName, MaxOperatorLen}) {
		return
	}

	// Check existing: reactivate a previously deactivated operator instead of
	// silently returning an inactive entry (it would never show up in the UI).
	var existing Operator
	var activeInt int
	err := db.QueryRow(
		"SELECT id, display_name, active FROM operators WHERE display_name = ?",
		req.DisplayName,
	).Scan(&existing.ID, &existing.DisplayName, &activeInt)
	if err == nil {
		if activeInt == 0 {
			db.Exec("UPDATE operators SET active = 1 WHERE id = ?", existing.ID)
			activeInt = 1
		}
		existing.Active = activeInt == 1
		writeJSON(w, 200, existing)
		return
	}

	result, err := db.Exec(
		"INSERT INTO operators (display_name, active) VALUES (?, 1)",
		req.DisplayName,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			db.QueryRow(
				"SELECT id, display_name, active FROM operators WHERE display_name = ?",
				req.DisplayName,
			).Scan(&existing.ID, &existing.DisplayName, &activeInt)
			if activeInt == 0 {
				db.Exec("UPDATE operators SET active = 1 WHERE id = ?", existing.ID)
				activeInt = 1
			}
			existing.Active = activeInt == 1
			writeJSON(w, 200, existing)
			return
		}
		writeError(w, 500, err.Error())
		return
	}
	id, _ := result.LastInsertId()
	writeJSON(w, 200, Operator{ID: int(id), DisplayName: req.DisplayName, Active: true})
}

// DELETE /api/operators/{id}
func handleDeleteOperator(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	result, err := db.Exec("UPDATE operators SET active = 0 WHERE id = ?", id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, 404, "Operator not found")
		return
	}
	writeJSON(w, 200, OKResponse{OK: true})
}

// --- Settings ---

func getSetting(key, defaultVal string) string {
	var val string
	err := db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&val)
	if err != nil || val == "" {
		return defaultVal
	}
	return val
}

// --- purge password protection (salted hash, never stored in plaintext) ---

func purgePasswordSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashPurgePassword(pw, salt string) string {
	h := sha256.New()
	h.Write([]byte(salt))
	h.Write([]byte(pw))
	return hex.EncodeToString(h.Sum(nil))
}

func purgePasswordConfigured() bool {
	return getSetting("purge_password_hash", "") != "" && getSetting("purge_password_salt", "") != ""
}

// verifyPurgePassword reports whether pw matches the stored protection.
func verifyPurgePassword(pw string) bool {
	hash := getSetting("purge_password_hash", "")
	salt := getSetting("purge_password_salt", "")
	if hash == "" || salt == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(hash), []byte(hashPurgePassword(pw, salt))) == 1
}

// GET /api/settings
func handleGetSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, SettingsResponse{
		HasPurgePassword: purgePasswordConfigured(),
	})
}

// PATCH /api/settings
//
// Only the purge password lives here now: the label size settings were removed
// together with the label printer integration (the printer has its own web
// page, so nothing in this app needs those numbers).
func handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var req SettingsUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	if req.PurgePassword == nil {
		handleGetSettings(w, r)
		return
	}

	// Validate before writing anything, then store the salt and the hash
	// together so a rejected or interrupted update cannot leave a half-set
	// password behind.
	pw := strings.TrimSpace(*req.PurgePassword)
	var salt, hash string
	if pw != "" {
		length := utf8.RuneCountInString(pw)
		if length < MinPurgePasswordLen {
			writeError(w, 400, "password too short (min 4 chars)")
			return
		}
		if length > MaxPurgePasswordLen {
			writeError(w, 400, "password too long (max 128 chars)")
			return
		}
		var err error
		salt, err = purgePasswordSalt()
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		hash = hashPurgePassword(pw, salt)
	}

	tx, err := db.Begin()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	put := func(key, value string) error {
		_, err := tx.Exec("INSERT OR REPLACE INTO app_settings (key, value) VALUES (?, ?)", key, value)
		return err
	}
	if err := put("purge_password_salt", salt); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	if err := put("purge_password_hash", hash); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	if err := tx.Commit(); err != nil {
		writeError(w, 500, err.Error())
		return
	}
	handleGetSettings(w, r)
}

// GET /api/settings/stats
func handleStats(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT COALESCE(s.quantity, 0), i.min_stock
		FROM items i
		LEFT JOIN stock s ON s.item_id = i.id
		WHERE i.active = 1
	`)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	totalItems := 0
	lowStock := 0
	totalQty := 0
	for rows.Next() {
		var qty, minStock int
		if err := rows.Scan(&qty, &minStock); err != nil {
			continue
		}
		totalItems++
		totalQty += qty
		// Same definition as the dashboard alerts: empty (<=0) counts as low
		// even when min_stock is 0, otherwise min_stock is the threshold.
		if qty <= 0 || (minStock > 0 && qty < minStock) {
			lowStock++
		}
	}

	writeJSON(w, 200, StatsResponse{
		TotalItems:    totalItems,
		LowStockCount: lowStock,
		TotalQuantity: totalQty,
	})
}

// --- Import / Export ---

// GET /api/data/export/csv
func handleExportCSV(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`
		SELECT i.barcode, i.name_zh, i.name_en, i.program, i.track_mode,
		       i.category, i.spec, i.unit, i.min_stock,
		       COALESCE(l.code, ''), COALESCE(s.quantity, 0), i.note
		FROM items i
		LEFT JOIN stock s ON s.item_id = i.id
		LEFT JOIN locations l ON l.id = i.location_id
		WHERE i.active = 1
		ORDER BY i.barcode
	`)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=items_export.csv")
	// BOM for Excel UTF-8
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Write([]string{"barcode", "name_zh", "name_en", "program", "track_mode", "category", "spec", "unit", "min_stock", "location_code", "quantity", "note"})

	for rows.Next() {
		var barcode, nameZh, nameEn, program, trackMode, category, spec, unit, locCode, note string
		var minStock, quantity int
		if err := rows.Scan(&barcode, &nameZh, &nameEn, &program, &trackMode,
			&category, &spec, &unit, &minStock, &locCode, &quantity, &note); err != nil {
			continue
		}
		cw.Write([]string{
			barcode, nameZh, nameEn, program, trackMode, category, spec, unit,
			strconv.Itoa(minStock), locCode, strconv.Itoa(quantity), note,
		})
	}
	cw.Flush()
}

// GET /api/data/export/template.csv
func handleDownloadTemplate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=items_import_template.csv")
	w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	cw.Write([]string{"name_zh", "name_en", "program", "track_mode", "category", "spec", "unit", "min_stock", "location_code", "quantity_initial", "note", "barcode"})
	cw.Write([]string{"NEO 电机", "NEO Motor", "FRC", "SNP", "BEAR", "2-1/16 in", "pcs", "2", "A-01-01", "1", "", ""})
	cw.Flush()
}

// POST /api/data/import
func handleImport(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, 400, "file too large")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "file required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Strip BOM if present
	if len(content) >= 3 && content[0] == 0xEF && content[1] == 0xBB && content[2] == 0xBF {
		content = content[3:]
	}

	filename := header.Filename
	var records [][]string

	if strings.HasSuffix(strings.ToLower(filename), ".csv") {
		reader := csv.NewReader(strings.NewReader(string(content)))
		// A hand-edited file often has a short row. Letting the reader enforce
		// one width made a single missing comma reject the whole file, losing
		// every valid row with it; missing cells now read as empty.
		reader.FieldsPerRecord = -1
		reader.TrimLeadingSpace = true
		records, err = reader.ReadAll()
		if err != nil {
			writeError(w, 400, "invalid CSV: "+err.Error())
			return
		}
	} else {
		writeError(w, 400, "Upload .csv file")
		return
	}

	if len(records) < 2 {
		writeError(w, 400, "empty file")
		return
	}

	headers := records[0]
	headerIdx := map[string]int{}
	for i, h := range headers {
		headerIdx[strings.TrimSpace(h)] = i
	}

	getField := func(row []string, name string) string {
		if idx, ok := headerIdx[name]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	created := 0
	errs := []string{}

	tx, err := db.Begin()
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer tx.Rollback()

	for i := 1; i < len(records); i++ {
		row := records[i]
		rowNum := i + 1 // 1-based, header is row 1

		nameZh := getField(row, "name_zh")
		nameEn := getField(row, "name_en")
		if nameZh == "" && nameEn == "" {
			continue
		}
		if nameZh == "" {
			nameZh = nameEn
		}
		if nameEn == "" {
			nameEn = nameZh
		}
		if msg := maxLenError("name_zh", nameZh, MaxNameLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}
		if msg := maxLenError("name_en", nameEn, MaxNameLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}

		program := strings.ToUpper(getField(row, "program"))
		if program == "" || (program != ProgramFRC && program != ProgramFTC && program != ProgramBOTH) {
			program = ProgramBOTH
		}

		trackMode := strings.ToUpper(getField(row, "track_mode"))
		if trackMode != TrackSNP {
			trackMode = TrackBLK
		}

		barcode := getField(row, "barcode")
		if msg := maxLenError("barcode", barcode, MaxBarcodeLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}
		if barcode == "" {
			// Query inside the transaction: sees rows inserted by this import
			// and does not deadlock the single-connection pool.
			barcode = nextBarcode(tx, program, trackMode)
		}

		// Check duplicate
		var exists int
		tx.QueryRow("SELECT COUNT(*) FROM items WHERE barcode = ?", barcode).Scan(&exists)
		if exists > 0 {
			errs = append(errs, fmt.Sprintf("Row %d: duplicate barcode %s", rowNum, barcode))
			continue
		}

		category := getField(row, "category")
		if category == "" {
			category = "MISC"
		}
		spec := getField(row, "spec")
		unit := getField(row, "unit")
		if unit == "" {
			unit = "pcs"
		}
		if msg := maxLenError("category", category, MaxCategoryLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}
		if msg := maxLenError("spec", spec, MaxSpecLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}
		if msg := maxLenError("unit", unit, MaxUnitLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}

		// Reject "2.5" or "abc" instead of silently storing 0: a rounded-down
		// min_stock quietly disables that item's low-stock alert.
		minStock, err := parseCount(getField(row, "min_stock"))
		if err != nil {
			errs = append(errs, fmt.Sprintf("Row %d: min_stock %v", rowNum, err))
			continue
		}
		qty, err := parseCount(getField(row, "quantity_initial"))
		if err != nil {
			errs = append(errs, fmt.Sprintf("Row %d: quantity_initial %v", rowNum, err))
			continue
		}

		note := getField(row, "note")
		if msg := maxLenError("note", note, MaxNoteLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}

		// Resolve location
		var locationID *int
		locCode := getField(row, "location_code")
		if msg := maxLenError("location_code", locCode, MaxCodeLen); msg != "" {
			errs = append(errs, fmt.Sprintf("Row %d: %s", rowNum, msg))
			continue
		}
		if locCode != "" {
			var lid int
			err := tx.QueryRow("SELECT id FROM locations WHERE code = ? COLLATE NOCASE", locCode).Scan(&lid)
			if err != nil {
				res, err := tx.Exec("INSERT INTO locations (code, name_zh, name_en) VALUES (?, ?, ?)", locCode, locCode, locCode)
				if err != nil {
					errs = append(errs, fmt.Sprintf("Row %d: location error: %v", rowNum, err))
					continue
				}
				newID, _ := res.LastInsertId()
				lid = int(newID)
			}
			locationID = &lid
		}

		result, err := tx.Exec(`
			INSERT INTO items (barcode, program, track_mode, name_zh, name_en, category, spec, unit, min_stock, location_id, note, active, created_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, barcode, program, trackMode, nameZh, nameEn, category, spec, unit, minStock, locationID, note, 1, nowSQL())
		if err != nil {
			errs = append(errs, fmt.Sprintf("Row %d: %v", rowNum, err))
			continue
		}

		itemID, _ := result.LastInsertId()
		tx.Exec("INSERT OR IGNORE INTO stock (item_id, quantity) VALUES (?, ?)", itemID, qty)
		created++
	}

	if err := tx.Commit(); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	writeJSON(w, 200, ImportResult{Created: created, Errors: errs})
}

// POST /api/data/backup
func handleBackup(w http.ResponseWriter, r *http.Request) {
	dataDir := filepath.Dir(dbPath())
	backupDir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// Millisecond precision avoids overwriting a backup created in the same second.
	stamp := time.Now().Format("20060102_150405.000")
	destName := "inventory_" + stamp + ".db"
	dest := filepath.Join(backupDir, destName)

	// VACUUM INTO produces a consistent snapshot (including WAL contents) and
	// cannot race with concurrent writes the way a plain file copy could.
	escaped := strings.ReplaceAll(dest, "'", "''")
	if _, err := db.Exec("VACUUM INTO '" + escaped + "'"); err != nil {
		writeError(w, 500, "backup failed: "+err.Error())
		return
	}

	writeJSON(w, 200, BackupResult{Path: dest, Filename: destName})
}

// helper to get database file path for backup
func dbPath() string {
	return filepath.Join(dataDirectory, "inventory.db")
}
