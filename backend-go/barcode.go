package main

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// rowQuerier is satisfied by both *sql.DB and *sql.Tx so barcode generation
// can run inside a transaction (import) without deadlocking the single
// connection pool (db.SetMaxOpenConns(1)).
type rowQuerier interface {
	Query(query string, args ...any) (*sql.Rows, error)
}

func barcodePrefix(program, trackMode string) string {
	prog := program
	if prog != ProgramFRC && prog != ProgramFTC {
		prog = "FIRST"
	}
	mode := TrackBLK
	if trackMode == TrackSNP {
		mode = TrackSNP
	}
	return prog + "-" + mode
}

func maxSerial(q rowQuerier, prefix string) int {
	pattern := prefix + "-%"
	rows, err := q.Query("SELECT barcode FROM items WHERE barcode LIKE ?", pattern)
	if err != nil {
		return 0
	}
	defer rows.Close()

	maxNum := 0
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		parts := strings.Split(code, "-")
		if len(parts) < 1 {
			continue
		}
		if n, err := strconv.Atoi(parts[len(parts)-1]); err == nil && n > maxNum {
			maxNum = n
		}
	}
	return maxNum
}

func nextBarcode(q rowQuerier, program, trackMode string) string {
	prefix := barcodePrefix(program, trackMode)
	return fmt.Sprintf("%s-%05d", prefix, maxSerial(q, prefix)+1)
}
