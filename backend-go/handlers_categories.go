package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
)

// GET /api/categories
func handleListCategories(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT id, name_zh, name_en, sort_order FROM categories ORDER BY sort_order, id")
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}
	defer rows.Close()

	cats := []Category{}
	for rows.Next() {
		var c Category
		if err := rows.Scan(&c.ID, &c.NameZh, &c.NameEn, &c.SortOrder); err != nil {
			continue
		}
		cats = append(cats, c)
	}
	if cats == nil {
		cats = []Category{}
	}
	writeJSON(w, 200, cats)
}

// POST /api/categories
func handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}

	req.NameZh = strings.TrimSpace(req.NameZh)
	if req.NameZh == "" {
		writeError(w, 400, "name_zh required")
		return
	}
	req.NameEn = strings.TrimSpace(req.NameEn)
	if !checkLengths(w,
		fieldLimit{"name_zh", req.NameZh, MaxCategoryLen},
		fieldLimit{"name_en", req.NameEn, MaxCategoryLen},
	) {
		return
	}

	// Check duplicate
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM categories WHERE name_zh = ?", req.NameZh).Scan(&exists)
	if exists > 0 {
		writeError(w, 400, "Category already exists")
		return
	}

	result, err := db.Exec(
		"INSERT INTO categories (name_zh, name_en) VALUES (?, ?)",
		req.NameZh, req.NameEn,
	)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	id, _ := result.LastInsertId()
	writeJSON(w, 200, Category{
		ID:     int(id),
		NameZh: req.NameZh,
		NameEn: req.NameEn,
	})
}

// DELETE /api/categories/{id}
func handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	// Check if any items reference this category
	var nameZh string
	if err := db.QueryRow("SELECT name_zh FROM categories WHERE id = ?", id).Scan(&nameZh); err != nil {
		writeError(w, 404, "Category not found")
		return
	}

	var refCount int
	db.QueryRow("SELECT COUNT(*) FROM items WHERE category = ?", nameZh).Scan(&refCount)
	if refCount > 0 {
		writeError(w, 400, map[string]any{
			"code":    "CATEGORY_IN_USE",
			"message": "Category in use by " + strconv.Itoa(refCount) + " item(s)",
			"count":   refCount,
		})
		return
	}

	result, err := db.Exec("DELETE FROM categories WHERE id = ?", id)
	if err != nil {
		writeError(w, 500, err.Error())
		return
	}

	affected, _ := result.RowsAffected()
	if affected == 0 {
		writeError(w, 404, "Category not found")
		return
	}

	writeJSON(w, 200, OKResponse{OK: true})
}

// PATCH /api/categories/{id}
func handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		writeError(w, 400, "invalid id")
		return
	}

	var oldNameZh string
	if err := db.QueryRow("SELECT name_zh FROM categories WHERE id = ?", id).Scan(&oldNameZh); err != nil {
		writeError(w, 404, "Category not found")
		return
	}

	var req CategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "invalid JSON")
		return
	}
	req.NameZh = strings.TrimSpace(req.NameZh)
	req.NameEn = strings.TrimSpace(req.NameEn)

	if req.NameZh == "" {
		writeError(w, 400, "name_zh required")
		return
	}
	if !checkLengths(w,
		fieldLimit{"name_zh", req.NameZh, MaxCategoryLen},
		fieldLimit{"name_en", req.NameEn, MaxCategoryLen},
	) {
		return
	}

	// Check duplicate (exclude self)
	var exists int
	db.QueryRow("SELECT COUNT(*) FROM categories WHERE name_zh = ? AND id != ?", req.NameZh, id).Scan(&exists)
	if exists > 0 {
		writeError(w, 400, "Category already exists")
		return
	}

	if _, err := db.Exec("UPDATE categories SET name_zh = ?, name_en = ? WHERE id = ?",
		req.NameZh, req.NameEn, id); err != nil {
		writeError(w, 500, err.Error())
		return
	}

	// If name_zh changed, update items referencing old name
	if oldNameZh != req.NameZh {
		db.Exec("UPDATE items SET category = ? WHERE category = ?", req.NameZh, oldNameZh)
	}

	writeJSON(w, 200, Category{ID: id, NameZh: req.NameZh, NameEn: req.NameEn})
}
