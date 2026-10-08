package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB
var dataDirectory string

// timeFormat is the SQLite-compatible datetime format.
const timeFormat = "2006-01-02 15:04:05"

// nowSQL returns the current time formatted for SQLite.
func nowSQL() string {
	return time.Now().UTC().Format(timeFormat)
}

// parseTime parses a SQLite datetime string.
func parseTime(s string) time.Time {
	t, _ := time.Parse(timeFormat, s)
	return t
}

func initDB(dataDir string) {
	dataDirectory = dataDir
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	// DSN parameters are driver specific. modernc.org/sqlite only understands
	// _pragma/_time_format/_txlock, so the mattn/go-sqlite3 style
	// "?_journal_mode=WAL&_foreign_keys=on" used to be dropped silently,
	// leaving the database in rollback-journal mode with foreign keys OFF.
	dbPath := filepath.Join(dataDir, "inventory.db")

	var err error
	db, err = sql.Open("sqlite", dbPath+
		"?_pragma=journal_mode(WAL)"+
		"&_pragma=foreign_keys(1)"+
		"&_pragma=busy_timeout(5000)")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := verifyPragmas(); err != nil {
		log.Fatalf("db pragmas: %v", err)
	}

	migrate()
	seed()
}

// verifyPragmas fails fast when the connection parameters did not take effect.
// Both settings used to be requested with parameters this driver ignores, and
// the silent downgrade went unnoticed until an orphan location_id showed up.
func verifyPragmas() error {
	var journalMode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		return err
	}
	if journalMode != "wal" {
		return fmt.Errorf("journal_mode is %q, want wal", journalMode)
	}
	var foreignKeys int
	if err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		return err
	}
	if foreignKeys != 1 {
		return fmt.Errorf("foreign_keys is %d, want 1", foreignKeys)
	}
	return nil
}

func migrate() {
	schema := `
	CREATE TABLE IF NOT EXISTS locations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT NOT NULL UNIQUE,
		name_zh TEXT DEFAULT '',
		name_en TEXT DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS operators (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		display_name TEXT NOT NULL UNIQUE,
		active INTEGER NOT NULL DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		barcode TEXT NOT NULL UNIQUE,
		program TEXT NOT NULL DEFAULT 'BOTH',
		track_mode TEXT NOT NULL DEFAULT 'BLK',
		name_zh TEXT NOT NULL,
		name_en TEXT NOT NULL,
		category TEXT NOT NULL DEFAULT 'MISC',
		spec TEXT NOT NULL DEFAULT '',
		unit TEXT NOT NULL DEFAULT 'pcs',
		min_stock INTEGER NOT NULL DEFAULT 0,
		location_id INTEGER,
		note TEXT NOT NULL DEFAULT '',
		active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (location_id) REFERENCES locations(id)
	);

	CREATE TABLE IF NOT EXISTS stock (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		item_id INTEGER UNIQUE NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 0,
		FOREIGN KEY (item_id) REFERENCES items(id)
	);

	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		item_id INTEGER NOT NULL,
		quantity INTEGER NOT NULL DEFAULT 1,
		operator_id INTEGER,
		note TEXT NOT NULL DEFAULT '',
		created_at TEXT NOT NULL DEFAULT (datetime('now')),
		FOREIGN KEY (item_id) REFERENCES items(id),
		FOREIGN KEY (operator_id) REFERENCES operators(id)
	);

	CREATE TABLE IF NOT EXISTS app_settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name_zh TEXT NOT NULL,
		name_en TEXT NOT NULL DEFAULT '',
		sort_order INTEGER NOT NULL DEFAULT 0,
		UNIQUE(name_zh)
	);

	CREATE INDEX IF NOT EXISTS idx_items_barcode ON items(barcode);
	CREATE INDEX IF NOT EXISTS idx_items_program ON items(program);
	CREATE INDEX IF NOT EXISTS idx_items_category ON items(category);
	CREATE INDEX IF NOT EXISTS idx_items_track_mode ON items(track_mode);
	CREATE INDEX IF NOT EXISTS idx_transactions_created ON transactions(created_at);
	CREATE INDEX IF NOT EXISTS idx_transactions_item ON transactions(item_id);
	`
	if _, err := db.Exec(schema); err != nil {
		log.Fatalf("migrate: %v", err)
	}
}

func seed() {
	// The label size settings were dropped along with the label printer
	// integration, so clean up the rows older databases still carry.
	if _, err := db.Exec("DELETE FROM app_settings WHERE key IN ('label_width_mm', 'label_height_mm')"); err != nil {
		log.Printf("drop legacy label settings: %v", err)
	}

	// Default categories
	var catCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&catCount); err == nil && catCount == 0 {
		defaultCategories := []struct {
			nameZh string
			nameEn string
			order  int
		}{
			{"电控", "Electronics", 1},
			{"电机", "Motor", 2},
			{"传动件", "Power Transmission", 3},
			{"轮子", "Wheels", 4},
			{"皮带/链条", "Belts & Chains", 5},
			{"气动", "Pneumatics", 6},
			{"结构件", "Structure", 7},
			{"工具", "Tools", 8},
			{"耗材", "Consumables", 9},
			{"其他", "MISC", 99},
		}
		for _, c := range defaultCategories {
			db.Exec(
				"INSERT OR IGNORE INTO categories (name_zh, name_en, sort_order) VALUES (?, ?, ?)",
				c.nameZh, c.nameEn, c.order,
			)
		}
	}

	// Default operators
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM operators").Scan(&count); err != nil {
		log.Printf("seed operators count: %v", err)
		return
	}
	if count == 0 {
		for _, name := range []string{"库管", "Warehouse"} {
			if _, err := db.Exec(
				"INSERT OR IGNORE INTO operators (display_name, active) VALUES (?, 1)",
				name,
			); err != nil {
				log.Printf("seed operator %s: %v", name, err)
			}
		}
	}
}
