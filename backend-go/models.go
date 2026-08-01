package main

import (
	"encoding/json"
	"time"
)

// --- Enums ---

const (
	ProgramFRC  = "FRC"
	ProgramFTC  = "FTC"
	ProgramBOTH = "BOTH"

	TrackSNP = "SNP"
	TrackBLK = "BLK"

	TxIn     = "IN"
	TxOut    = "OUT"
	TxReturn = "RETURN"
	TxAdjust = "ADJUST"
)

// --- DB Models ---

type Location struct {
	ID     int    `json:"id"`
	Code   string `json:"code"`
	NameZh string `json:"name_zh"`
	NameEn string `json:"name_en"`
}

type Operator struct {
	ID          int    `json:"id"`
	DisplayName string `json:"display_name"`
	Active      bool   `json:"active"`
}

type Item struct {
	ID         int       `json:"id"`
	Barcode    string    `json:"barcode"`
	Program    string    `json:"program"`
	TrackMode  string    `json:"track_mode"`
	NameZh     string    `json:"name_zh"`
	NameEn     string    `json:"name_en"`
	Category   string    `json:"category"`
	Spec       string    `json:"spec"`
	Unit       string    `json:"unit"`
	MinStock   int       `json:"min_stock"`
	LocationID *int      `json:"location_id"`
	Note       string    `json:"note"`
	Active     bool      `json:"active"`
	CreatedAt  time.Time `json:"created_at"`
	// joined
	Quantity     int     `json:"quantity"`
	LocationCode *string `json:"location_code"`
}

type Stock struct {
	ID       int `json:"-"`
	ItemID   int `json:"item_id"`
	Quantity int `json:"quantity"`
}

type Transaction struct {
	ID         int       `json:"id"`
	Type       string    `json:"type"`
	ItemID     int       `json:"item_id"`
	Quantity   int       `json:"quantity"`
	OperatorID *int      `json:"-"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"created_at"`
	// joined
	Barcode      string  `json:"barcode"`
	NameZh       string  `json:"name_zh"`
	NameEn       string  `json:"name_en"`
	OperatorName *string `json:"operator_name"`
}

type AppSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// --- Request / Response ---

type ItemCreateRequest struct {
	Barcode        string `json:"barcode"`
	Program        string `json:"program"`
	TrackMode      string `json:"track_mode"`
	NameZh         string `json:"name_zh"`
	NameEn         string `json:"name_en"`
	Category       string `json:"category"`
	Spec           string `json:"spec"`
	Unit           string `json:"unit"`
	MinStock       int    `json:"min_stock"`
	LocationID     *int   `json:"location_id"`
	Note           string `json:"note"`
	Active         *bool  `json:"active"`
	QuantityInit   int    `json:"quantity_initial"`
}

// OptionalInt distinguishes "field absent" from an explicit null/zero in
// PATCH requests, so callers can clear a value (e.g. location_id -> NULL).
type OptionalInt struct {
	Set   bool
	Value int
}

func (o *OptionalInt) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		o.Set = true
		o.Value = 0
		return nil
	}
	var v int
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	o.Set = true
	o.Value = v
	return nil
}

// NullValue reports whether the field was explicitly set to null.
func (o *OptionalInt) NullValue() bool { return o.Set && o.Value == 0 }

type ItemUpdateRequest struct {
	Program    *string      `json:"program"`
	TrackMode  *string      `json:"track_mode"`
	NameZh     *string      `json:"name_zh"`
	NameEn     *string      `json:"name_en"`
	Category   *string      `json:"category"`
	Spec       *string      `json:"spec"`
	Unit       *string      `json:"unit"`
	MinStock   *int         `json:"min_stock"`
	LocationID OptionalInt  `json:"location_id"`
	Note       *string      `json:"note"`
	Active     *bool        `json:"active"`
}

type ItemListResponse struct {
	Items    []Item `json:"items"`
	Total    int    `json:"total"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type ScanLookupResponse struct {
	Kind     string    `json:"kind"`
	Item     *Item     `json:"item"`
	Location *Location `json:"location"`
}

type TransactionCreateRequest struct {
	Barcode    string `json:"barcode"`
	Type       string `json:"type"`
	Quantity   int    `json:"quantity"`
	OperatorID *int   `json:"operator_id"`
	Note       string `json:"note"`
}

type BatchTransactionLine struct {
	Barcode  string `json:"barcode"`
	Type     string `json:"type"`
	Quantity int    `json:"quantity"`
}

type BatchTransactionRequest struct {
	OperatorID *int                   `json:"operator_id"`
	Note       string                 `json:"note"`
	Lines      []BatchTransactionLine `json:"lines"`
}

type BatchTransactionResult struct {
	Success      bool                   `json:"success"`
	Transactions []Transaction          `json:"transactions"`
	Errors       []BatchTransactionLine `json:"errors,omitempty"`
}

type TransactionListResponse struct {
	Transactions []Transaction `json:"transactions"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	PageSize     int           `json:"page_size"`
}

type LocationRequest struct {
	Code   string `json:"code"`
	NameZh string `json:"name_zh"`
	NameEn string `json:"name_en"`
}

type OperatorRequest struct {
	DisplayName string `json:"display_name"`
}

type SettingsResponse struct {
	LabelWidthMM     float64 `json:"label_width_mm"`
	LabelHeightMM    float64 `json:"label_height_mm"`
	HasPurgePassword bool    `json:"has_purge_password"`
}

type SettingsUpdateRequest struct {
	LabelWidthMM  *float64 `json:"label_width_mm"`
	LabelHeightMM *float64 `json:"label_height_mm"`
	// Empty string clears the protection; <4 chars is rejected.
	PurgePassword *string `json:"purge_password"`
}

type PurgeRequest struct {
	Password string `json:"password"`
}

type StatsResponse struct {
	TotalItems    int `json:"total_items"`
	LowStockCount int `json:"low_stock_count"`
	TotalQuantity int `json:"total_quantity"`
}

type InboundSummary struct {
	TodayTransactions int `json:"today_transactions"`
	TodayQuantity     int `json:"today_quantity"`
	WeekQuantity      int `json:"week_quantity"`
}

type RecentInboundLine struct {
	ID           int       `json:"id"`
	Barcode      string    `json:"barcode"`
	NameZh       string    `json:"name_zh"`
	NameEn       string    `json:"name_en"`
	Quantity     int       `json:"quantity"`
	OperatorName *string   `json:"operator_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type LowStockAlert struct {
	ID       int    `json:"id"`
	Barcode  string `json:"barcode"`
	NameZh   string `json:"name_zh"`
	NameEn   string `json:"name_en"`
	Program  string `json:"program"`
	Category string `json:"category"`
	Quantity int    `json:"quantity"`
	MinStock int    `json:"min_stock"`
	Shortage int    `json:"shortage"`
	Unit     string `json:"unit"`
	Level    string `json:"level"`
}

type DashboardResponse struct {
	GeneratedAt    time.Time           `json:"generated_at"`
	Inbound        InboundSummary      `json:"inbound"`
	RecentInbound  []RecentInboundLine `json:"recent_inbound"`
	LowStockAlerts []LowStockAlert     `json:"low_stock_alerts"`
	AlertCount     int                 `json:"alert_count"`
}

type ImportResult struct {
	Created int      `json:"created"`
	Errors  []string `json:"errors"`
}

type BackupResult struct {
	Path     string `json:"path"`
	Filename string `json:"filename"`
}

type NextBarcodeResponse struct {
	Barcode string `json:"barcode"`
}

type OKResponse struct {
	OK bool `json:"ok"`
}

// --- Categories ---

type Category struct {
	ID       int    `json:"id"`
	NameZh   string `json:"name_zh"`
	NameEn   string `json:"name_en"`
	SortOrder int   `json:"sort_order"`
}

type CategoryRequest struct {
	NameZh string `json:"name_zh"`
	NameEn string `json:"name_en"`
}
