package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// fixedMux mirrors the route table in main.go, wrapped in the same middleware
// chain, so these tests exercise the real request pipeline.
func fixedMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/items", handleListItems)
	mux.HandleFunc("POST /api/items", handleCreateItem)
	mux.HandleFunc("GET /api/items/lookup/{code}", handleLookupCode)
	mux.HandleFunc("GET /api/items/{id}", handleGetItem)
	mux.HandleFunc("PATCH /api/items/{id}", handleUpdateItem)
	mux.HandleFunc("DELETE /api/items/{id}", handleDeleteItem)
	mux.HandleFunc("DELETE /api/items/{id}/purge", handlePurgeItem)
	mux.HandleFunc("POST /api/transactions", handleCreateTransaction)
	mux.HandleFunc("POST /api/transactions/batch", handleBatchTransactions)
	mux.HandleFunc("GET /api/transactions", handleListTransactions)
	mux.HandleFunc("GET /api/locations", handleListLocations)
	mux.HandleFunc("POST /api/locations", handleCreateLocation)
	mux.HandleFunc("PATCH /api/locations/{id}", handleUpdateLocation)
	mux.HandleFunc("DELETE /api/locations/{id}", handleDeleteLocation)
	mux.HandleFunc("GET /api/categories", handleListCategories)
	mux.HandleFunc("POST /api/categories", handleCreateCategory)
	mux.HandleFunc("PATCH /api/categories/{id}", handleUpdateCategory)
	mux.HandleFunc("POST /api/operators", handleCreateOperator)
	mux.HandleFunc("DELETE /api/operators/{id}", handleDeleteOperator)
	mux.HandleFunc("GET /api/settings", handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", handleUpdateSettings)
	mux.HandleFunc("GET /api/settings/stats", handleStats)
	mux.HandleFunc("GET /api/dashboard", handleDashboard)
	mux.HandleFunc("GET /api/data/export/csv", handleExportCSV)
	mux.HandleFunc("POST /api/data/import", handleImport)
	return loggingMiddleware(guardMiddleware(mux))
}

func call(t *testing.T, method, path string, body any, contentType string) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	switch v := body.(type) {
	case nil:
		payload = nil
	case []byte:
		payload = v
	default:
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		payload = b
	}
	req := httptest.NewRequest(method, path, bytes.NewReader(payload))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	rr := httptest.NewRecorder()
	fixedMux().ServeHTTP(rr, req)
	return rr
}

func callJSON(t *testing.T, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return call(t, method, path, body, "application/json")
}

func mustStatus(t *testing.T, rr *httptest.ResponseRecorder, want int, what string) {
	t.Helper()
	if rr.Code != want {
		t.Fatalf("%s: got %d, want %d (%s)", what, rr.Code, want, strings.TrimSpace(rr.Body.String()))
	}
}

func itemIDOf(t *testing.T, barcode string) int {
	t.Helper()
	var id int
	if err := db.QueryRow("SELECT id FROM items WHERE barcode = ?", barcode).Scan(&id); err != nil {
		t.Fatalf("item %s: %v", barcode, err)
	}
	return id
}

func uploadCSV(t *testing.T, content string) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "import.csv")
	fw.Write([]byte(content))
	mw.Close()
	req := httptest.NewRequest("POST", "/api/data/import", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rr := httptest.NewRecorder()
	fixedMux().ServeHTTP(rr, req)
	return rr
}

// The DSN used to request WAL and foreign keys with parameters this driver
// silently ignores.
func TestPragmasAreApplied(t *testing.T) {
	resetDB(t)

	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}
	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatalf("foreign_keys: %v", err)
	}
	if foreignKeys != 1 {
		t.Errorf("foreign_keys = %d, want 1", foreignKeys)
	}
}

func TestDanglingLocationRejected(t *testing.T) {
	resetDB(t)

	rr := callJSON(t, "POST", "/api/items", map[string]any{
		"barcode": "FIX-LOC-1", "name_zh": "幽灵库位", "location_id": 99999,
	})
	mustStatus(t, rr, 400, "create with unknown location")

	id := createItem(t, map[string]any{"barcode": "FIX-LOC-2", "name_zh": "x"})
	rr = callJSON(t, "PATCH", fmt.Sprintf("/api/items/%d", id), map[string]any{"location_id": 424242})
	mustStatus(t, rr, 400, "patch with unknown location")

	// a real location still works, and null still clears
	rr = callJSON(t, "POST", "/api/locations", map[string]any{"code": "FIX-A1"})
	mustStatus(t, rr, 200, "create location")
	var loc Location
	json.Unmarshal(rr.Body.Bytes(), &loc)

	rr = callJSON(t, "PATCH", fmt.Sprintf("/api/items/%d", id), map[string]any{"location_id": loc.ID})
	mustStatus(t, rr, 200, "patch with real location")
	rr = callJSON(t, "PATCH", fmt.Sprintf("/api/items/%d", id), map[string]any{"location_id": nil})
	mustStatus(t, rr, 200, "clear location")
	var locID *int
	db.QueryRow("SELECT location_id FROM items WHERE id = ?", id).Scan(&locID)
	if locID != nil {
		t.Errorf("location_id after null = %v, want nil", *locID)
	}
}

func TestLocationNameCanBeCleared(t *testing.T) {
	resetDB(t)

	rr := callJSON(t, "POST", "/api/locations", map[string]any{
		"code": "FIX-L1", "name_zh": "旧名", "name_en": "Old",
	})
	mustStatus(t, rr, 200, "create location")
	var loc Location
	json.Unmarshal(rr.Body.Bytes(), &loc)

	rr = callJSON(t, "PATCH", fmt.Sprintf("/api/locations/%d", loc.ID),
		map[string]any{"name_zh": "", "name_en": ""})
	mustStatus(t, rr, 200, "clear location names")
	var zh, en string
	db.QueryRow("SELECT name_zh, name_en FROM locations WHERE id = ?", loc.ID).Scan(&zh, &en)
	if zh != "" || en != "" {
		t.Errorf("names not cleared: name_zh=%q name_en=%q", zh, en)
	}

	// the code itself may not be blanked
	rr = callJSON(t, "PATCH", fmt.Sprintf("/api/locations/%d", loc.ID), map[string]any{"code": "  "})
	mustStatus(t, rr, 400, "blank location code")

	// location codes are unique regardless of case
	rr = callJSON(t, "PATCH", fmt.Sprintf("/api/locations/%d", loc.ID), map[string]any{"code": "FIX-L1"})
	mustStatus(t, rr, 200, "rename to same code, different case should be allowed for self")
}

func TestLocationCodeCaseInsensitiveDuplicate(t *testing.T) {
	resetDB(t)
	mustStatus(t, callJSON(t, "POST", "/api/locations", map[string]any{"code": "A-01"}), 200, "create A-01")
	rr := callJSON(t, "POST", "/api/locations", map[string]any{"code": "a-01"})
	mustStatus(t, rr, 400, "create a-01 (same shelf, different case)")
}

func TestPatchMissingItemIs404(t *testing.T) {
	resetDB(t)
	mustStatus(t, callJSON(t, "PATCH", "/api/items/999999", map[string]any{"name_zh": "x"}), 404, "patch missing")
	mustStatus(t, callJSON(t, "PATCH", "/api/items/999999", map[string]any{}), 404, "patch missing with empty body")
}

func TestBarcodeIsTrimmed(t *testing.T) {
	resetDB(t)

	var it Item
	rr := callJSON(t, "POST", "/api/items", map[string]any{"barcode": "  FIX-WS-1  ", "name_zh": "padded"})
	mustStatus(t, rr, 200, "create with padded barcode")
	json.Unmarshal(rr.Body.Bytes(), &it)
	if it.Barcode != "FIX-WS-1" {
		t.Errorf("stored barcode = %q, want FIX-WS-1", it.Barcode)
	}
	rr = call(t, "GET", "/api/items/lookup/FIX-WS-1", nil, "")
	var res ScanLookupResponse
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res.Kind != "item" {
		t.Errorf("scanning the trimmed code returned kind=%q, want item", res.Kind)
	}

	// whitespace-only falls back to an auto-generated barcode instead of being stored
	rr = callJSON(t, "POST", "/api/items", map[string]any{"barcode": "   ", "name_zh": "blank-code"})
	mustStatus(t, rr, 200, "create with blank barcode")
	json.Unmarshal(rr.Body.Bytes(), &it)
	if strings.TrimSpace(it.Barcode) == "" {
		t.Errorf("blank barcode was stored verbatim")
	}
}

func TestSearchEscapesLikeMetacharacters(t *testing.T) {
	resetDB(t)
	createItem(t, map[string]any{"barcode": "FIX-LK-1", "name_zh": "齿轮"})
	createItem(t, map[string]any{"barcode": "FIX-LK-2", "name_zh": "电机"})

	count := func(q string) int {
		rr := call(t, "GET", "/api/items?q="+url.QueryEscape(q), nil, "")
		var res ItemListResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return res.Total
	}
	if got := count("%"); got != 0 {
		t.Errorf("search for %% matched %d rows, want 0", got)
	}
	if got := count("_"); got != 0 {
		t.Errorf("search for _ matched %d rows, want 0", got)
	}
	if got := count("齿轮"); got != 1 {
		t.Errorf("search for 齿轮 matched %d rows, want 1", got)
	}
}

func TestPageSizeIsClampedNotReset(t *testing.T) {
	resetDB(t)
	rr := call(t, "GET", "/api/items?page_size=100000", nil, "")
	var res ItemListResponse
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res.PageSize != maxPageSize {
		t.Errorf("page_size = %d, want %d", res.PageSize, maxPageSize)
	}
	rr = call(t, "GET", "/api/transactions?page_size=100000", nil, "")
	var txRes TransactionListResponse
	json.Unmarshal(rr.Body.Bytes(), &txRes)
	if txRes.PageSize != maxPageSize {
		t.Errorf("transaction page_size = %d, want %d", txRes.PageSize, maxPageSize)
	}
}

func TestFieldLengthLimits(t *testing.T) {
	resetDB(t)
	long := strings.Repeat("长", MaxNameLen+1)

	mustStatus(t, callJSON(t, "POST", "/api/items", map[string]any{"name_zh": long}), 400, "long item name")
	mustStatus(t, callJSON(t, "POST", "/api/operators", map[string]any{"display_name": long}), 400, "long operator name")
	mustStatus(t, callJSON(t, "POST", "/api/categories", map[string]any{"name_zh": long}), 400, "long category name")
	mustStatus(t, callJSON(t, "POST", "/api/locations", map[string]any{
		"code": "FIX-LONG", "name_zh": strings.Repeat("x", MaxLocationNameLen+1),
	}), 400, "long location name")

	// a name at the limit is still accepted
	mustStatus(t, callJSON(t, "POST", "/api/items", map[string]any{
		"name_zh": strings.Repeat("长", MaxNameLen),
	}), 200, "name at the limit")
}

func TestNegativeMinStockRejected(t *testing.T) {
	resetDB(t)
	mustStatus(t, callJSON(t, "POST", "/api/items", map[string]any{"name_zh": "x", "min_stock": -1}), 400, "negative min_stock on create")
	id := createItem(t, map[string]any{"name_zh": "y"})
	mustStatus(t, callJSON(t, "PATCH", fmt.Sprintf("/api/items/%d", id), map[string]any{"min_stock": -5}), 400, "negative min_stock on patch")
}

// The purge password is the only setting left after the label printer
// integration was removed. A rejected value must not be stored at all, and the
// salt/hash pair must land together.
func TestSettingsPasswordRules(t *testing.T) {
	resetDB(t)

	mustStatus(t, callJSON(t, "PATCH", "/api/settings", map[string]any{"purge_password": "abc"}), 400, "too short")
	if purgePasswordConfigured() {
		t.Error("a rejected password left the protection configured")
	}
	var rows int
	db.QueryRow("SELECT COUNT(*) FROM app_settings WHERE key LIKE 'purge_password%'").Scan(&rows)
	if rows != 0 {
		t.Errorf("rejected password wrote %d settings row(s)", rows)
	}

	mustStatus(t, callJSON(t, "PATCH", "/api/settings", map[string]any{
		"purge_password": strings.Repeat("x", MaxPurgePasswordLen+1),
	}), 400, "too long")
	if purgePasswordConfigured() {
		t.Error("an over-long password was stored")
	}

	rr := callJSON(t, "PATCH", "/api/settings", map[string]any{"purge_password": "good-pass"})
	mustStatus(t, rr, 200, "valid password")
	var sr SettingsResponse
	json.Unmarshal(rr.Body.Bytes(), &sr)
	if !sr.HasPurgePassword {
		t.Error("has_purge_password = false after setting a password")
	}
	if !verifyPurgePassword("good-pass") || verifyPurgePassword("wrong") {
		t.Error("password verification is wrong")
	}

	// a body without the field must leave the stored password alone
	mustStatus(t, callJSON(t, "PATCH", "/api/settings", map[string]any{}), 200, "empty patch")
	if !verifyPurgePassword("good-pass") {
		t.Error("empty patch changed the stored password")
	}
}

// Rows left behind by versions that still stored a label size are cleaned up
// on startup.
func TestLegacyLabelSettingsAreRemoved(t *testing.T) {
	resetDB(t)

	if _, err := db.Exec(`INSERT OR REPLACE INTO app_settings (key, value)
		VALUES ('label_width_mm', '40'), ('label_height_mm', '30')`); err != nil {
		t.Fatalf("seed legacy rows: %v", err)
	}
	seed()

	var legacy int
	db.QueryRow("SELECT COUNT(*) FROM app_settings WHERE key IN ('label_width_mm', 'label_height_mm')").Scan(&legacy)
	if legacy != 0 {
		t.Errorf("legacy label settings still present: %d", legacy)
	}
}

func TestPurgePasswordTrimsWhitespace(t *testing.T) {
	resetDB(t)
	id := createItem(t, map[string]any{"name_zh": "x"})

	// whitespace-only means "clear", not "a password made of spaces"
	mustStatus(t, callJSON(t, "PATCH", "/api/settings", map[string]any{"purge_password": "   "}), 200, "clear with spaces")
	if purgePasswordConfigured() {
		t.Error("whitespace-only password should clear the protection")
	}
	mustStatus(t, doPurge(t, id, "   "), 400, "purge without protection")
}

func TestOperatorMustBeActive(t *testing.T) {
	resetDB(t)
	createItem(t, map[string]any{"barcode": "FIX-OP-1", "name_zh": "x", "quantity_initial": 5})

	rr := callJSON(t, "POST", "/api/operators", map[string]any{"display_name": "FIX 操作人"})
	mustStatus(t, rr, 200, "create operator")
	var op Operator
	json.Unmarshal(rr.Body.Bytes(), &op)

	body := map[string]any{"barcode": "FIX-OP-1", "type": "OUT", "quantity": 1}
	body["operator_id"] = op.ID
	mustStatus(t, callJSON(t, "POST", "/api/transactions", body), 200, "active operator")

	body["operator_id"] = 987654
	mustStatus(t, callJSON(t, "POST", "/api/transactions", body), 400, "unknown operator")

	mustStatus(t, callJSON(t, "DELETE", fmt.Sprintf("/api/operators/%d", op.ID), nil), 200, "deactivate operator")
	body["operator_id"] = op.ID
	mustStatus(t, callJSON(t, "POST", "/api/transactions", body), 400, "deactivated operator")

	// same rule inside a batch
	mustStatus(t, callJSON(t, "POST", "/api/transactions/batch", map[string]any{
		"operator_id": op.ID,
		"lines":       []map[string]any{{"barcode": "FIX-OP-1", "type": "IN", "quantity": 1}},
	}), 400, "batch with deactivated operator")

	// a zero id simply means "no operator"
	body["operator_id"] = 0
	mustStatus(t, callJSON(t, "POST", "/api/transactions", body), 200, "operator_id 0 means none")
}

func TestBatchRejectsMissingOperatorBeforeWriting(t *testing.T) {
	resetDB(t)
	createItem(t, map[string]any{"barcode": "FIX-BOP-1", "name_zh": "x", "quantity_initial": 1})
	before := getStock(t, itemIDOf(t, "FIX-BOP-1"))

	mustStatus(t, callJSON(t, "POST", "/api/transactions/batch", map[string]any{
		"operator_id": 424242,
		"lines":       []map[string]any{{"barcode": "FIX-BOP-1", "type": "IN", "quantity": 9}},
	}), 400, "batch with unknown operator")

	if after := getStock(t, itemIDOf(t, "FIX-BOP-1")); after != before {
		t.Errorf("stock changed (%d -> %d) although the batch was rejected", before, after)
	}
}

func TestTransactionPagingHasStableTieBreak(t *testing.T) {
	resetDB(t)
	id := createItem(t, map[string]any{"barcode": "FIX-PG-1", "name_zh": "page"})

	stamp := nowSQL()
	for i := 0; i < 60; i++ {
		if _, err := db.Exec(
			`INSERT INTO transactions (type, item_id, quantity, note, created_at) VALUES (?, ?, 1, ?, ?)`,
			TxIn, id, fmt.Sprintf("row-%02d", i), stamp,
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	seen := map[int]bool{}
	for page := 1; page <= 2; page++ {
		rr := call(t, "GET", fmt.Sprintf("/api/transactions?page=%d&page_size=50", page), nil, "")
		var res TransactionListResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		for _, tx := range res.Transactions {
			if seen[tx.ID] {
				t.Errorf("transaction %d appeared on both pages", tx.ID)
			}
			seen[tx.ID] = true
		}
	}
	if len(seen) != 60 {
		t.Errorf("pages covered %d of 60 transactions", len(seen))
	}
}

func TestSimpleContentTypesRejected(t *testing.T) {
	resetDB(t)
	body := `{"barcode":"FIX-CT-1","name_zh":"跨站"}`

	// the three content types a browser can send cross-origin without a
	// preflight must not reach the handlers
	for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded"} {
		rr := call(t, "POST", "/api/items", []byte(body), ct)
		mustStatus(t, rr, 415, "POST with Content-Type "+ct)
	}
	rr := call(t, "POST", "/api/items", []byte(body), "multipart/form-data")
	mustStatus(t, rr, 415, "multipart outside the import endpoint")

	var n int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE barcode = 'FIX-CT-1'").Scan(&n)
	if n != 0 {
		t.Errorf("cross-site style request created %d item(s)", n)
	}

	// the JSON client path is unaffected, and no permissive CORS header is sent
	rr = callJSON(t, "POST", "/api/items", map[string]any{"barcode": "FIX-CT-1", "name_zh": "正常"})
	mustStatus(t, rr, 200, "JSON POST")
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want it absent", got)
	}
	rr = call(t, "OPTIONS", "/api/items", nil, "")
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("preflight returned Access-Control-Allow-Origin %q, want it absent", got)
	}
}

func TestImportToleratesShortRows(t *testing.T) {
	resetDB(t)

	// row 3 is missing its trailing columns; the other rows must still import
	rr := uploadCSV(t, "name_zh,name_en,program,min_stock\n"+
		"短行A,Short A,FRC,2\n"+
		"短行B\n"+
		"短行C,Short C,FTC,1\n")
	mustStatus(t, rr, 200, "import with a short row")
	var res ImportResult
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res.Created != 3 {
		t.Errorf("created = %d, want 3 (errors: %v)", res.Created, res.Errors)
	}
	var minStock int
	db.QueryRow("SELECT min_stock FROM items WHERE name_zh = '短行A'").Scan(&minStock)
	if minStock != 2 {
		t.Errorf("min_stock = %d, want 2", minStock)
	}
}

func TestImportReportsBadNumbers(t *testing.T) {
	resetDB(t)

	rr := uploadCSV(t, "name_zh,min_stock,quantity_initial\n"+
		"小数,2.5,3.9\n"+
		"正常,4,5\n")
	mustStatus(t, rr, 200, "import with a decimal number")
	var res ImportResult
	json.Unmarshal(rr.Body.Bytes(), &res)
	if res.Created != 1 || len(res.Errors) != 1 {
		t.Fatalf("created = %d, errors = %v; want 1 created and 1 error", res.Created, res.Errors)
	}
	if !strings.Contains(res.Errors[0], "min_stock") {
		t.Errorf("error does not name the field: %q", res.Errors[0])
	}

	// nothing was stored for the rejected row, and the good row kept its value
	var q, m int
	db.QueryRow(`SELECT s.quantity, i.min_stock FROM items i JOIN stock s ON s.item_id = i.id
	             WHERE i.name_zh = '正常'`).Scan(&q, &m)
	if q != 5 || m != 4 {
		t.Errorf("good row stored quantity=%d min_stock=%d, want 5 and 4", q, m)
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE name_zh = '小数'").Scan(&n)
	if n != 0 {
		t.Errorf("rejected row was imported anyway")
	}
}
