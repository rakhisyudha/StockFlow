package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"stockflow/backend/internal/product"
)

func TestListProducts(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price, stock) VALUES
		('KOP-001', 'Kopi Arabika', 25000, 50),
		('TEH-001', 'Teh Hijau', 15000, 5)`); err != nil {
		t.Fatalf("seed products: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO stock_movements (product_id, type, qty) VALUES (1, 'in', 50)`); err != nil {
		t.Fatalf("seed movement: %v", err)
	}

	handler := testHandler(pool)

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantTotal  int
		wantCount  int
		wantCode   string
	}{
		{name: "all products", path: "/api/products", wantStatus: 200, wantTotal: 2, wantCount: 2},
		{name: "search is case insensitive", path: "/api/products?search=KOPI&page=1&limit=1", wantStatus: 200, wantTotal: 1, wantCount: 1},
		{name: "empty search result", path: "/api/products?search=nonexistent", wantStatus: 200, wantTotal: 0, wantCount: 0},
		{name: "invalid page", path: "/api/products?page=0", wantStatus: 400, wantCode: "BAD_REQUEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, handler, http.MethodGet, tt.path, "")
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var body struct {
				Data  []product.Product `json:"data"`
				Page  int               `json:"page"`
				Limit int               `json:"limit"`
				Total int               `json:"total"`
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if tt.wantCode != "" {
				if body.Error.Code != tt.wantCode {
					t.Errorf("error.code = %q, want %q", body.Error.Code, tt.wantCode)
				}
				return
			}
			if body.Total != tt.wantTotal {
				t.Errorf("total = %d, want %d", body.Total, tt.wantTotal)
			}
			if len(body.Data) != tt.wantCount {
				t.Fatalf("len(data) = %d, want %d", len(body.Data), tt.wantCount)
			}
			if body.Page < 1 || body.Limit < 1 {
				t.Errorf("page/limit = %d/%d, want positive", body.Page, body.Limit)
			}
			if tt.name == "all products" {
				first := body.Data[0]
				if first.SKU != "KOP-001" || first.LastMovementAt == nil {
					t.Errorf("first product = %+v, want KOP-001 with last_movement_at", first)
				}
				if body.Data[1].LastMovementAt != nil {
					t.Errorf("second product should have null last_movement_at")
				}
			}
		})
	}
}
