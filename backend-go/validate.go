package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

// maxPageSize caps list endpoints. A larger request is clamped rather than
// silently replaced by the default, which used to turn "give me 500 rows" into
// 50 rows with no indication that it happened.
const maxPageSize = 200

// Field length limits. SQLite stores TEXT without any length constraint, so
// without these a single request could persist megabytes into one field.
// The values are deliberately generous for real inventory data.
const (
	MaxBarcodeLen      = 64
	MaxNameLen         = 120
	MaxSpecLen         = 120
	MaxUnitLen         = 16
	MaxNoteLen         = 500
	MaxCodeLen         = 32
	MaxLocationNameLen = 60
	MaxCategoryLen     = 40
	MaxOperatorLen     = 40
)

// Purge password bounds.
const (
	MinPurgePasswordLen = 4
	MaxPurgePasswordLen = 128
)

// queryRower is satisfied by both *sql.DB and *sql.Tx. Handlers that already
// hold a transaction must pass the transaction: the pool is capped at one
// connection (db.SetMaxOpenConns(1)), so querying db while a tx is open would
// block forever.
type queryRower interface {
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// fieldLimit pairs a request field with its maximum length for checkLengths.
type fieldLimit struct {
	field string
	value string
	max   int
}

// maxLenError returns a human readable message when value is longer than max
// runes, or "" when it fits.
func maxLenError(field, value string, max int) string {
	if n := utf8.RuneCountInString(value); n > max {
		return fmt.Sprintf("%s too long (%d characters, max %d)", field, n, max)
	}
	return ""
}

// checkLengths writes a 400 and reports false as soon as one field is too long.
func checkLengths(w http.ResponseWriter, limits ...fieldLimit) bool {
	for _, l := range limits {
		if msg := maxLenError(l.field, l.value, l.max); msg != "" {
			writeError(w, 400, msg)
			return false
		}
	}
	return true
}

// likePattern escapes the LIKE metacharacters so a search for "%" matches a
// literal percent sign instead of every row. Use with `LIKE ? ESCAPE '\'`.
func likePattern(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 4)
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteRune('\\')
		}
		b.WriteRune(r)
	}
	return "%" + b.String() + "%"
}

// checkLocationExists returns "" when id refers to a stored location. It is
// checked explicitly so a bad location_id is a friendly 400 instead of a
// foreign-key failure surfacing as a 500.
func checkLocationExists(q queryRower, id int) string {
	var one int
	if err := q.QueryRow("SELECT 1 FROM locations WHERE id = ?", id).Scan(&one); err != nil {
		return "location not found"
	}
	return ""
}

// checkOperatorActive returns "" when id refers to an existing, active
// operator. Transactions used to accept ids of deleted or never-existing
// operators and silently record an anonymous movement.
func checkOperatorActive(q queryRower, id int) string {
	var active int
	if err := q.QueryRow("SELECT active FROM operators WHERE id = ?", id).Scan(&active); err != nil {
		return "operator not found"
	}
	if active == 0 {
		return "operator is inactive"
	}
	return ""
}

// parseCount reads a non-negative whole number from a CSV cell. An empty cell
// means zero; anything else must parse, so "2.5" is reported instead of being
// silently rounded down to 0.
func parseCount(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("must be a whole number, got %q", raw)
	}
	if n < 0 {
		return 0, fmt.Errorf("cannot be negative, got %d", n)
	}
	return n, nil
}
