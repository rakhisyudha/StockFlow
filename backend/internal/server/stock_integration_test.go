package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"stockflow/backend/internal/product"
)

func TestAdjustStockHappyPath(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price, stock) VALUES ('STK-001', 'Stock Item', 1000, 10)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPost, "/api/products/1/stock",
		`{"type":"in","qty":5,"note":"restock"}`)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var in struct {
		Product  product.Product  `json:"product"`
		Movement product.Movement `json:"movement"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &in); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if in.Product.Stock != 15 || in.Product.LastMovementAt == nil {
		t.Errorf("product = %+v, want stock 15 with last movement", in.Product)
	}
	if in.Movement.Type != "in" || in.Movement.Qty != 5 || in.Movement.Note == nil || *in.Movement.Note != "restock" {
		t.Errorf("movement = %+v", in.Movement)
	}

	rec = doRequest(t, handler, http.MethodPost, "/api/products/1/stock", `{"type":"out","qty":4}`)
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var out struct {
		Product  product.Product  `json:"product"`
		Movement product.Movement `json:"movement"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Product.Stock != 11 || out.Movement.Note != nil {
		t.Errorf("product/movement = %+v / %+v", out.Product, out.Movement)
	}

	var stock, movements int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE product_id = 1`).Scan(&movements); err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if stock != 11 || movements != 2 {
		t.Errorf("db state = stock %d, %d movements; want 11 and 2", stock, movements)
	}
}

func TestAdjustStockInsufficientWritesNothing(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price, stock) VALUES ('STK-002', 'Low Stock', 1000, 3)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPost, "/api/products/1/stock", `{"type":"out","qty":5,"note":"overdraw"}`)
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			Available int    `json:"available"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != "INSUFFICIENT_STOCK" || body.Error.Available != 3 {
		t.Errorf("error = %+v, want INSUFFICIENT_STOCK with available 3", body.Error)
	}

	var stock, movements int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE product_id = 1`).Scan(&movements); err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if stock != 3 || movements != 0 {
		t.Errorf("failed adjustment changed state: stock %d, %d movements", stock, movements)
	}
}

func TestAdjustStockMissingProduct(t *testing.T) {
	pool := testPool(t)
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPost, "/api/products/9999/stock", `{"type":"in","qty":1}`)
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
}

func TestMovementsNewestFirst(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price) VALUES ('STK-003', 'History', 1000)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO stock_movements (product_id, type, qty, created_at) VALUES
		(1, 'in', 1, '2026-01-01T00:00:00Z'),
		(1, 'out', 1, '2026-01-03T00:00:00Z'),
		(1, 'in', 1, '2026-01-02T00:00:00Z')`); err != nil {
		t.Fatalf("seed movements: %v", err)
	}
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodGet, "/api/products/1/movements", "")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var body product.MovementListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Data) != 3 {
		t.Fatalf("len(data) = %d, want 3", len(body.Data))
	}
	for i := 1; i < len(body.Data); i++ {
		if body.Data[i-1].CreatedAt.Before(body.Data[i].CreatedAt) {
			t.Fatalf("movements not newest first: %v", body.Data)
		}
	}
}

func TestConcurrentStockDecrements(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price, stock) VALUES ('RACE-001', 'Race', 1000, 10)`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	handler := testHandler(pool)

	const attempts = 20
	var succeeded, conflicted, unexpected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/products/1/stock",
				strings.NewReader(`{"type":"out","qty":1}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			switch rec.Code {
			case 200:
				succeeded.Add(1)
			case 409:
				conflicted.Add(1)
			default:
				unexpected.Add(1)
			}
		}()
	}
	wg.Wait()

	if unexpected.Load() != 0 {
		t.Errorf("unexpected statuses: %d", unexpected.Load())
	}
	if succeeded.Load() != 10 || conflicted.Load() != 10 {
		t.Errorf("succeeded = %d, conflicted = %d; want 10 and 10", succeeded.Load(), conflicted.Load())
	}

	var stock, movements int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE id = 1`).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM stock_movements WHERE product_id = 1`).Scan(&movements); err != nil {
		t.Fatalf("count movements: %v", err)
	}
	if stock != 0 || movements != 10 {
		t.Errorf("db state = stock %d, %d movements; want 0 and 10", stock, movements)
	}
}
