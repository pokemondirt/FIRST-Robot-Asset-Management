package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func resetDB(t *testing.T) {
	t.Helper()
	initDB(t.TempDir())
	t.Cleanup(func() { db.Close() })
}

func testMux() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/items", handleCreateItem)
	mux.HandleFunc("PATCH /api/items/{id}", handleUpdateItem)
	mux.HandleFunc("DELETE /api/items/{id}/purge", handlePurgeItem)
	mux.HandleFunc("GET /api/items/lookup/{code}", handleLookupCode)
	mux.HandleFunc("POST /api/transactions", handleCreateTransaction)
	mux.HandleFunc("GET /api/dashboard", handleDashboard)
	mux.HandleFunc("GET /api/settings/stats", handleStats)
	mux.HandleFunc("GET /api/settings", handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", handleUpdateSettings)
	mux.HandleFunc("POST /api/data/import", handleImport)
	mux.HandleFunc("POST /api/data/backup", handleBackup)
	mux.HandleFunc("POST /api/operators", handleCreateOperator)
	return mux
}

func postJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	rr := httptest.NewRecorder()
	testMux().ServeHTTP(rr, httptest.NewRequest("POST", path, bytes.NewReader(b)))
	return rr
}

func patchJSON(t *testing.T, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", path, bytes.NewReader(b))
	testMux().ServeHTTP(rr, req)
	return rr
}

func getStock(t *testing.T, itemID int) int {
	t.Helper()
	var q int
	if err := db.QueryRow("SELECT quantity FROM stock WHERE item_id = ?", itemID).Scan(&q); err != nil {
		t.Fatalf("get stock: %v", err)
	}
	return q
}

func createItem(t *testing.T, body map[string]any) int {
	t.Helper()
	rr := postJSON(t, "/api/items", body)
	if rr.Code != 200 {
		t.Fatalf("create item: %d %s", rr.Code, rr.Body.String())
	}
	var it Item
	json.Unmarshal(rr.Body.Bytes(), &it)
	return it.ID
}

func TestAdjustToZero(t *testing.T) {
	resetDB(t)
	createItem(t, map[string]any{"barcode": "T-ADJ-1", "name_zh": "x", "quantity_initial": 5})
	rr := postJSON(t, "/api/transactions", map[string]any{"barcode": "T-ADJ-1", "type": "ADJUST", "quantity": 0})
	if rr.Code != 200 {
		t.Fatalf("adjust: %d %s", rr.Code, rr.Body.String())
	}
	var itemID int
	db.QueryRow("SELECT id FROM items WHERE barcode='T-ADJ-1'").Scan(&itemID)
	if got := getStock(t, itemID); got != 0 {
		t.Errorf("ADJUST to 0: stock=%d, want 0", got)
	}
}

func TestClearLocation(t *testing.T) {
	resetDB(t)
	db.Exec("INSERT INTO locations (code) VALUES ('A-01')")
	var locID int
	db.QueryRow("SELECT id FROM locations WHERE code='A-01'").Scan(&locID)
	id := createItem(t, map[string]any{"barcode": "T-LOC-1", "name_zh": "x", "location_id": locID})

	rr := patchJSON(t, fmt.Sprintf("/api/items/%d", id), map[string]any{"location_id": nil})
	if rr.Code != 200 {
		t.Fatalf("patch: %d %s", rr.Code, rr.Body.String())
	}
	var loc *int
	db.QueryRow("SELECT location_id FROM items WHERE id=?", id).Scan(&loc)
	if loc != nil {
		t.Errorf("location_id after PATCH null = %v, want nil", *loc)
	}

	// non-null value should still work
	rr = patchJSON(t, fmt.Sprintf("/api/items/%d", id), map[string]any{"location_id": locID})
	db.QueryRow("SELECT location_id FROM items WHERE id=?", id).Scan(&loc)
	if loc == nil || *loc != locID {
		t.Errorf("location_id after PATCH id = %v, want %d", loc, locID)
	}
}

func TestBarcodeMatchesProgram(t *testing.T) {
	resetDB(t)
	// case A: program change only -> barcode regenerated with new program
	idA := createItem(t, map[string]any{"name_zh": "autoA", "program": "FRC", "track_mode": "BLK"})
	var barcode, program string
	db.QueryRow("SELECT barcode, program FROM items WHERE id=?", idA).Scan(&barcode, &program)
	if !strings.HasPrefix(barcode, "FRC-") {
		t.Fatalf("setup: barcode=%s", barcode)
	}
	rr := patchJSON(t, fmt.Sprintf("/api/items/%d", idA), map[string]any{"program": "BOTH"})
	if rr.Code != 200 {
		t.Fatalf("patch A: %d %s", rr.Code, rr.Body.String())
	}
	db.QueryRow("SELECT barcode, program FROM items WHERE id=?", idA).Scan(&barcode, &program)
	if !strings.HasPrefix(barcode, "FIRST-") {
		t.Errorf("program->BOTH: barcode=%s, want FIRST- prefix", barcode)
	}

	// case B: program + track_mode together -> barcode uses NEW program
	idB := createItem(t, map[string]any{"name_zh": "autoB", "program": "FRC", "track_mode": "BLK"})
	rr = patchJSON(t, fmt.Sprintf("/api/items/%d", idB), map[string]any{"program": "FTC", "track_mode": "SNP"})
	if rr.Code != 200 {
		t.Fatalf("patch B: %d %s", rr.Code, rr.Body.String())
	}
	db.QueryRow("SELECT barcode, program FROM items WHERE id=?", idB).Scan(&barcode, &program)
	if !strings.HasPrefix(barcode, "FTC-SNP-") {
		t.Errorf("program->FTC, mode->SNP: barcode=%s, want FTC-SNP- prefix", barcode)
	}

	// no-op edit (same values) must NOT regenerate the barcode
	idC := createItem(t, map[string]any{"name_zh": "autoC", "program": "FRC", "track_mode": "BLK"})
	db.QueryRow("SELECT barcode FROM items WHERE id=?", idC).Scan(&barcode)
	rr = patchJSON(t, fmt.Sprintf("/api/items/%d", idC), map[string]any{"program": "FRC", "track_mode": "BLK", "name_zh": "renamed"})
	if rr.Code != 200 {
		t.Fatalf("patch C: %d %s", rr.Code, rr.Body.String())
	}
	var barcode2 string
	db.QueryRow("SELECT barcode FROM items WHERE id=?", idC).Scan(&barcode2)
	if barcode2 != barcode {
		t.Errorf("no-op edit regenerated barcode: %s -> %s", barcode, barcode2)
	}
}

func TestImportEmptyBarcodeNoHang(t *testing.T) {
	resetDB(t)
	csvData := "name_zh,name_en,program,track_mode,category,spec,unit,min_stock,location_code,quantity_initial,note,barcode\n" +
		"测试A,Test A,FRC,SNP,BEAR,,pcs,1,,1,,\n" +
		"测试B,Test B,FRC,SNP,BEAR,,pcs,1,,1,,\n"

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", "import.csv")
	fw.Write([]byte(csvData))
	mw.Close()

	done := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/data/import", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		testMux().ServeHTTP(rr, req)
		done <- rr
	}()

	select {
	case rr := <-done:
		if rr.Code != 200 {
			t.Fatalf("import: %d %s", rr.Code, rr.Body.String())
		}
		var res ImportResult
		json.Unmarshal(rr.Body.Bytes(), &res)
		if res.Created != 2 || len(res.Errors) != 0 {
			t.Errorf("import: created=%d errors=%v, want 2 created no errors", res.Created, res.Errors)
		}
		// both auto barcodes must be unique
		var n int
		db.QueryRow("SELECT COUNT(DISTINCT barcode) FROM items").Scan(&n)
		if n != 2 {
			t.Errorf("distinct barcodes=%d, want 2", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("import still hangs (deadlock not fixed)")
	}
}

func TestDashboardTodayTimezone(t *testing.T) {
	resetDB(t)
	localMidnight := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	storedUTC := localMidnight.Add(30 * time.Minute).In(time.UTC).Format(timeFormat)
	id := createItem(t, map[string]any{"barcode": "T-TZ-1", "name_zh": "z", "quantity_initial": 0})
	db.Exec(`INSERT INTO transactions (type, item_id, quantity, created_at) VALUES (?, ?, 1, ?)`, TxIn, id, storedUTC)

	todayStr := startOfToday().Format(timeFormat)
	if storedUTC < todayStr {
		t.Fatalf("test setup broken: stored %q < boundary %q", storedUTC, todayStr)
	}

	var count int
	db.QueryRow(`SELECT COUNT(*) FROM transactions WHERE type = ? AND created_at >= ?`, TxIn, todayStr).Scan(&count)
	if count != 1 {
		t.Errorf("today inbound count=%d, want 1 (early-morning tx misattributed)", count)
	}
}

func TestBackup(t *testing.T) {
	resetDB(t)
	createItem(t, map[string]any{"barcode": "T-BK-1", "name_zh": "x", "quantity_initial": 3})
	rr := postJSON(t, "/api/data/backup", nil)
	if rr.Code != 200 {
		t.Fatalf("backup: %d %s", rr.Code, rr.Body.String())
	}
	var res BackupResult
	json.Unmarshal(rr.Body.Bytes(), &res)
	if _, err := os.Stat(res.Path); err != nil {
		t.Fatalf("backup file missing: %v", err)
	}
	if filepath.Ext(res.Filename) != ".db" {
		t.Errorf("unexpected filename %s", res.Filename)
	}
}

func TestOperatorReactivate(t *testing.T) {
	resetDB(t)
	db.Exec("INSERT INTO operators (display_name, active) VALUES ('临时工', 0)")
	rr := postJSON(t, "/api/operators", map[string]any{"display_name": "临时工"})
	if rr.Code != 200 {
		t.Fatalf("create operator: %d %s", rr.Code, rr.Body.String())
	}
	var op Operator
	json.Unmarshal(rr.Body.Bytes(), &op)
	if !op.Active {
		t.Error("reactivated operator returned active=false")
	}
	var active int
	db.QueryRow("SELECT active FROM operators WHERE display_name='临时工'").Scan(&active)
	if active != 1 {
		t.Errorf("operator still inactive in db (active=%d)", active)
	}
}

func TestStatsAndLookupConsistency(t *testing.T) {
	resetDB(t)
	// zero-stock item with min_stock 0 -> counts as low in both stats and dashboard
	createItem(t, map[string]any{"barcode": "T-ST-1", "name_zh": "zero", "min_stock": 0, "quantity_initial": 0})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/settings/stats", nil)
	testMux().ServeHTTP(rr, req)
	var stats StatsResponse
	json.Unmarshal(rr.Body.Bytes(), &stats)
	if stats.LowStockCount != 1 {
		t.Errorf("stats low_stock_count=%d, want 1 (zero-stock with min_stock=0)", stats.LowStockCount)
	}

	// inactive item must not be found by lookup (same rule as transactions)
	rr = postJSON(t, "/api/items", map[string]any{"barcode": "T-ST-2", "name_zh": "off", "active": false})
	if rr.Code != 200 {
		t.Fatalf("create inactive: %d %s", rr.Code, rr.Body.String())
	}
	rr2 := httptest.NewRecorder()
	testMux().ServeHTTP(rr2, httptest.NewRequest("GET", "/api/items/lookup/T-ST-2", nil))
	var res ScanLookupResponse
	json.Unmarshal(rr2.Body.Bytes(), &res)
	if res.Kind == "item" {
		t.Error("lookup returned an inactive item")
	}
}

func TestProgramValidation(t *testing.T) {
	resetDB(t)
	rr := postJSON(t, "/api/items", map[string]any{"barcode": "T-V-1", "name_zh": "x", "program": "frc"})
	if rr.Code != 400 {
		t.Errorf("create with invalid program: code=%d, want 400", rr.Code)
	}
	id := createItem(t, map[string]any{"barcode": "T-V-2", "name_zh": "y"})
	rr = patchJSON(t, fmt.Sprintf("/api/items/%d", id), map[string]any{"track_mode": "BULK"})
	if rr.Code != 400 {
		t.Errorf("patch with invalid track_mode: code=%d, want 400", rr.Code)
	}
}

func setPurgePassword(t *testing.T, pw string) {
	t.Helper()
	rr := patchJSON(t, "/api/settings", map[string]any{"purge_password": pw})
	if rr.Code != 200 {
		t.Fatalf("set purge password: %d %s", rr.Code, rr.Body.String())
	}
}

func doPurge(t *testing.T, id int, pw string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(map[string]string{"password": pw})
	rr := httptest.NewRecorder()
	testMux().ServeHTTP(rr, httptest.NewRequest("DELETE", fmt.Sprintf("/api/items/%d/purge", id), bytes.NewReader(b)))
	return rr
}

func TestPurgeItem(t *testing.T) {
	resetDB(t)
	setPurgePassword(t, "test123")
	id := createItem(t, map[string]any{"barcode": "T-PG-1", "name_zh": "x", "quantity_initial": 3})
	rr := postJSON(t, "/api/transactions", map[string]any{"barcode": "T-PG-1", "type": "OUT", "quantity": 1})
	if rr.Code != 200 {
		t.Fatalf("create tx: %d %s", rr.Code, rr.Body.String())
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM transactions WHERE item_id = ?", id).Scan(&n)
	if n != 1 {
		t.Fatalf("setup: tx count=%d, want 1", n)
	}

	// wrong password -> 403, nothing deleted
	rr = doPurge(t, id, "wrong")
	if rr.Code != 403 {
		t.Fatalf("purge wrong pw: code=%d, want 403", rr.Code)
	}
	db.QueryRow("SELECT COUNT(*) FROM items WHERE id = ?", id).Scan(&n)
	if n != 1 {
		t.Fatal("item deleted despite wrong password")
	}

	rr = doPurge(t, id, "test123")
	if rr.Code != 200 {
		t.Fatalf("purge: %d %s", rr.Code, rr.Body.String())
	}
	db.QueryRow("SELECT COUNT(*) FROM items WHERE id = ?", id).Scan(&n)
	if n != 0 {
		t.Errorf("item still exists after purge (id=%d)", id)
	}
	for _, table := range []string{"stock", "transactions"} {
		db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE item_id = ?", table), id).Scan(&n)
		if n != 0 {
			t.Errorf("%s rows still exist after purge (item_id=%d)", table, id)
		}
	}

	// purging a missing item -> 404
	rr2 := doPurge(t, id, "test123")
	if rr2.Code != 404 {
		t.Errorf("purge missing: code=%d, want 404", rr2.Code)
	}
}

func TestPurgePasswordProtection(t *testing.T) {
	resetDB(t)
	id := createItem(t, map[string]any{"barcode": "T-PP-1", "name_zh": "x"})

	// no password configured -> reject
	rr := doPurge(t, id, "")
	if rr.Code != 400 {
		t.Errorf("purge without configured password: code=%d, want 400", rr.Code)
	}

	// set a password
	setPurgePassword(t, "secret1")

	// wrong password -> 403
	rr = doPurge(t, id, "nope")
	if rr.Code != 403 {
		t.Errorf("purge with wrong password: code=%d, want 403", rr.Code)
	}

	// correct password -> 200 and item gone
	rr = doPurge(t, id, "secret1")
	if rr.Code != 200 {
		t.Fatalf("purge with correct password: %d %s", rr.Code, rr.Body.String())
	}
	var n int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE id = ?", id).Scan(&n)
	if n != 0 {
		t.Error("item still exists after purge with correct password")
	}

	// clearing the password disables protection again
	setPurgePassword(t, "")
	rr = doPurge(t, id, "")
	if rr.Code != 400 {
		t.Errorf("purge after clearing password: code=%d, want 400 (protection off)", rr.Code)
	}
}
