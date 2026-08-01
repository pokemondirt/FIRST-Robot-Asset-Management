package main

import (
	"database/sql"
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
	dbPath := filepath.Join(dataDir, "inventory.db")

	var err error
	db, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on&_time_format=sqlite")
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	db.SetMaxOpenConns(1)

	migrate()
	seed()
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
	// Default settings
	settings := map[string]string{
		"label_width_mm":  "40",
		"label_height_mm": "30",
	}
	for k, v := range settings {
		_, err := db.Exec(
			"INSERT OR IGNORE INTO app_settings (key, value) VALUES (?, ?)",
			k, v,
		)
		if err != nil {
			log.Printf("seed setting %s: %v", k, err)
		}
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
