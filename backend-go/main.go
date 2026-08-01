package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	// Determine data directory
	execDir, _ := os.Executable()
	dataDir := filepath.Join(filepath.Dir(execDir), "data")
	// If running with `go run`, use current directory
	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		dataDir = "data"
	}

	initDB(dataDir)
	log.Println("Database initialized")

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]string{"status": "ok"})
	})

	// Items
	mux.HandleFunc("GET /api/items", handleListItems)
	mux.HandleFunc("POST /api/items", handleCreateItem)
	mux.HandleFunc("GET /api/items/lookup/{code}", handleLookupCode)
	mux.HandleFunc("POST /api/items/next-barcode", handleNextBarcode)
	mux.HandleFunc("GET /api/items/{id}", handleGetItem)
	mux.HandleFunc("PATCH /api/items/{id}", handleUpdateItem)
	mux.HandleFunc("DELETE /api/items/{id}", handleDeleteItem)
	mux.HandleFunc("DELETE /api/items/{id}/purge", handlePurgeItem)

	// Transactions
	mux.HandleFunc("POST /api/transactions", handleCreateTransaction)
	mux.HandleFunc("POST /api/transactions/batch", handleBatchTransactions)
	mux.HandleFunc("GET /api/transactions", handleListTransactions)

	// Locations
	mux.HandleFunc("GET /api/locations", handleListLocations)
	mux.HandleFunc("POST /api/locations", handleCreateLocation)
	mux.HandleFunc("PATCH /api/locations/{id}", handleUpdateLocation)
	mux.HandleFunc("DELETE /api/locations/{id}", handleDeleteLocation)

	// Categories
	mux.HandleFunc("GET /api/categories", handleListCategories)
	mux.HandleFunc("POST /api/categories", handleCreateCategory)
	mux.HandleFunc("PATCH /api/categories/{id}", handleUpdateCategory)
	mux.HandleFunc("DELETE /api/categories/{id}", handleDeleteCategory)

	// Operators
	mux.HandleFunc("GET /api/operators", handleListOperators)
	mux.HandleFunc("POST /api/operators", handleCreateOperator)
	mux.HandleFunc("DELETE /api/operators/{id}", handleDeleteOperator)

	// Settings
	mux.HandleFunc("GET /api/settings", handleGetSettings)
	mux.HandleFunc("PATCH /api/settings", handleUpdateSettings)
	mux.HandleFunc("GET /api/settings/stats", handleStats)

	// Dashboard
	mux.HandleFunc("GET /api/dashboard", handleDashboard)

	// Data
	mux.HandleFunc("GET /api/data/export/csv", handleExportCSV)
	mux.HandleFunc("GET /api/data/export/template.csv", handleDownloadTemplate)
	mux.HandleFunc("POST /api/data/import", handleImport)
	mux.HandleFunc("POST /api/data/backup", handleBackup)

	// Static UI (optional): run build-frontend.ps1 first
	frontendDist := filepath.Join(filepath.Dir(execDir), "..", "frontend", "dist")
	if _, err := os.Stat(frontendDist); os.IsNotExist(err) {
		frontendDist = filepath.Join("..", "frontend", "dist")
	}
	if indexExists(frontendDist) {
		mux.Handle("/", spaHandler{distDir: frontendDist})
	}

	// Apply middleware
	handler := loggingMiddleware(corsMiddleware(mux))

	port := ":8000"
	log.Printf("Server starting on http://127.0.0.1%s", port)
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func indexExists(dist string) bool {
	_, err := os.Stat(filepath.Join(dist, "index.html"))
	return err == nil
}
