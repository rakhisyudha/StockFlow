package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"stockflow/backend/internal/product"
)

func TestCreateProduct(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price) VALUES ('EXIST-001', 'Existing', 1000)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := testHandler(pool)

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "valid", body: `{"sku":"NEW-001","name":"Kopi Robusta","price":18000}`, wantStatus: 201},
		{name: "explicit inactive", body: `{"sku":"NEW-002","name":"Teh Tarik","price":12000,"is_active":false}`, wantStatus: 201},
		{name: "duplicate sku", body: `{"sku":"EXIST-001","name":"Dup","price":1}`, wantStatus: 409, wantCode: "SKU_ALREADY_EXISTS"},
		{name: "empty sku", body: `{"sku":"  ","name":"Bad","price":1}`, wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "negative price", body: `{"sku":"BAD-001","name":"Bad","price":-1}`, wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "unknown field", body: `{"sku":"BAD-002","name":"Bad","price":1,"stock":99}`, wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "malformed json", body: `{`, wantStatus: 400, wantCode: "BAD_REQUEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, handler, http.MethodPost, "/api/products", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}

			var body struct {
				product.Product
				Error struct {
					Code    string   `json:"code"`
					Details []string `json:"details"`
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
			if body.ID < 1 || body.Stock != 0 || body.LastMovementAt != nil {
				t.Errorf("unexpected created product: %+v", body.Product)
			}
			if tt.name == "valid" && (!body.IsActive || body.Price != 18000) {
				t.Errorf("created product = %+v, want active with price 18000", body.Product)
			}
			if tt.name == "explicit inactive" && body.IsActive {
				t.Errorf("created product should be inactive")
			}
		})
	}
}

func TestUpdateProduct(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price) VALUES ('OLD-001', 'Old Name', 1000)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := testHandler(pool)

	tests := []struct {
		name       string
		path       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{name: "update", path: "/api/products/1", body: `{"sku":"NEW-001","name":"New Name","price":2000,"is_active":false}`, wantStatus: 200},
		{name: "missing id", path: "/api/products/9999", body: `{"sku":"X","name":"X","price":1}`, wantStatus: 404, wantCode: "PRODUCT_NOT_FOUND"},
		{name: "invalid id", path: "/api/products/abc", body: `{"sku":"X","name":"X","price":1}`, wantStatus: 400, wantCode: "BAD_REQUEST"},
		{name: "negative price", path: "/api/products/1", body: `{"sku":"X","name":"X","price":-5}`, wantStatus: 400, wantCode: "BAD_REQUEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, handler, http.MethodPut, tt.path, tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
			if tt.wantCode != "" {
				var body struct {
					Error struct {
						Code string `json:"code"`
					} `json:"error"`
				}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("decode body: %v", err)
				}
				if body.Error.Code != tt.wantCode {
					t.Errorf("error.code = %q, want %q", body.Error.Code, tt.wantCode)
				}
				return
			}
			var got product.Product
			if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.SKU != "NEW-001" || got.Name != "New Name" || got.Price != 2000 || got.IsActive {
				t.Errorf("updated product = %+v", got)
			}
		})
	}
}

func TestUpdatePreservesStatusWhenOmitted(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price) VALUES ('KEEP-001', 'Keep', 1000)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPut, "/api/products/1", `{"sku":"KEEP-001","name":"Renamed","price":1500}`)
	if rec.Code != 200 {
		t.Fatalf("status = %d; body = %s", rec.Code, rec.Body.String())
	}
	var got product.Product
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if !got.IsActive || got.Name != "Renamed" {
		t.Errorf("product = %+v, want active and renamed", got)
	}
}

func TestDeleteProduct(t *testing.T) {
	t.Run("history is protected", func(t *testing.T) {
		pool := testPool(t)
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price) VALUES ('HIST-001', 'Has History', 1000)`); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO stock_movements (product_id, type, qty) VALUES (1, 'in', 10)`); err != nil {
			t.Fatalf("seed movement: %v", err)
		}
		handler := testHandler(pool)

		rec := doRequest(t, handler, http.MethodDelete, "/api/products/1", "")
		if rec.Code != 409 {
			t.Fatalf("status = %d, want 409; body = %s", rec.Code, rec.Body.String())
		}
		var body struct {
			Error struct {
				Code string `json:"code"`
			} `json:"error"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if body.Error.Code != "PRODUCT_HAS_HISTORY" {
			t.Errorf("error.code = %q", body.Error.Code)
		}

		var count int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM products WHERE id = 1`).Scan(&count); err != nil {
			t.Fatalf("count: %v", err)
		}
		if count != 1 {
			t.Errorf("product was deleted despite history")
		}
	})

	t.Run("deletes without history", func(t *testing.T) {
		pool := testPool(t)
		ctx := context.Background()
		if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price) VALUES ('FREE-001', 'No History', 1000)`); err != nil {
			t.Fatalf("seed: %v", err)
		}
		handler := testHandler(pool)

		rec := doRequest(t, handler, http.MethodDelete, "/api/products/1", "")
		if rec.Code != 204 {
			t.Fatalf("status = %d, want 204; body = %s", rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("204 response should have no body")
		}
	})

	t.Run("missing product", func(t *testing.T) {
		pool := testPool(t)
		handler := testHandler(pool)

		rec := doRequest(t, handler, http.MethodDelete, "/api/products/9999", "")
		if rec.Code != 404 {
			t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
		}
	})
}
