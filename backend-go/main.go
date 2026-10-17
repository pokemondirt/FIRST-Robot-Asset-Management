package main

import (
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"first-inventory/webembed"
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

	// Static UI: a dist folder next to the binary (release override) or in the
	// dev layout wins, otherwise the copy embedded in the executable is served,
	// so shipping a single .exe is enough.
	if h, ok := resolveFrontend(execDir); ok {
		mux.Handle("/", h)
	} else {
		log.Println("no frontend found; run build-frontend.cmd to build it, serving the API only")
	}

	// Apply middleware
	handler := loggingMiddleware(guardMiddleware(mux))

	// Listening on every interface is deliberate: other computers on the LAN
	// are meant to open the app from their own browser. Only 127.0.0.1 is
	// printed by http.ListenAndServe's own messages, so log the real addresses.
	const port = ":8000"
	log.Printf("Server starting on http://127.0.0.1%s", port)
	for _, ip := range localIPv4s() {
		log.Printf("  On this network: http://%s%s/#/checkout", ip, port)
	}
	if err := http.ListenAndServe(port, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// localIPv4s returns the machine's non-loopback IPv4 addresses, so the startup
// log can tell the operator which URL to share on the local network.
func localIPv4s() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	var out []string
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		ip4 := ipNet.IP.To4()
		// Skip IPv6 and the 169.254.x.x self-assigned addresses, which a
		// disconnected adapter reports and nobody can connect to.
		if ip4 == nil || ip4.IsLinkLocalUnicast() {
			continue
		}
		out = append(out, ip4.String())
	}
	return out
}

// resolveFrontend returns a handler for the built UI. A dist directory on disk
// takes precedence, so a development build can be swapped in without
// recompiling; otherwise the copy embedded in the executable is used.
func resolveFrontend(execDir string) (spaHandler, bool) {
	candidates := []string{
		filepath.Join(filepath.Dir(execDir), "dist"),
		filepath.Join(filepath.Dir(execDir), "..", "frontend", "dist"),
		filepath.Join("..", "frontend", "dist"),
	}
	for _, dir := range candidates {
		if info, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !info.IsDir() {
			return spaHandler{fsys: os.DirFS(dir)}, true
		}
	}

	embedded, err := fs.Sub(webembed.Dist, "dist")
	if err != nil {
		return spaHandler{}, false
	}
	if _, err := fs.Stat(embedded, "index.html"); err != nil {
		return spaHandler{}, false
	}
	return spaHandler{fsys: embedded}, true
}
