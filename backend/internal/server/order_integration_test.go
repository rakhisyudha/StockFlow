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

	"github.com/jackc/pgx/v5/pgxpool"

	"stockflow/backend/internal/order"
)

func seedOrderProducts(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `INSERT INTO products (sku, name, price, stock) VALUES
		('KOP-001', 'Kopi Arabika', 25000, 10),
		('TEH-001', 'Teh Hijau', 15000, 5),
		('GULA-001', 'Gula Pasir', 10000, 3)`); err != nil {
		t.Fatalf("seed products: %v", err)
	}
}

func TestCreateOrderHappyPath(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPost, "/api/orders",
		`{"customer":"Andi","items":[{"sku":"KOP-001","qty":2},{"sku":"TEH-001","qty":1}]}`)
	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var got order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID < 1 || got.Customer != "Andi" || got.Status != "CREATED" || got.CancelledAt != nil {
		t.Errorf("order = %+v", got)
	}
	if got.Total != 65000 {
		t.Errorf("total = %d, want 65000", got.Total)
	}
	if len(got.Items) != 2 || got.Items[0].UnitPrice != 25000 || got.Items[1].UnitPrice != 15000 {
		t.Errorf("items = %+v, want server-computed unit prices", got.Items)
	}
	if got.Items[0].Qty != 2 || got.Items[0].SKU != "KOP-001" {
		t.Errorf("items = %+v", got.Items)
	}

	ctx := context.Background()
	var kopStock, tehStock, orders int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'KOP-001'`).Scan(&kopStock); err != nil {
		t.Fatalf("read kop stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'TEH-001'`).Scan(&tehStock); err != nil {
		t.Fatalf("read teh stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sales_orders`).Scan(&orders); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if kopStock != 8 || tehStock != 4 || orders != 1 {
		t.Errorf("db state = %d/%d stock, %d orders", kopStock, tehStock, orders)
	}
}

func TestCreateOrderValidation(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)

	tests := []struct {
		name     string
		body     string
		contains string
	}{
		{name: "empty customer", body: `{"customer":"","items":[{"sku":"KOP-001","qty":1}]}`, contains: "customer is required"},
		{name: "empty items", body: `{"customer":"Andi","items":[]}`, contains: "items must contain at least one item"},
		{name: "qty zero", body: `{"customer":"Andi","items":[{"sku":"KOP-001","qty":0}]}`, contains: "qty must be an integer"},
		{name: "duplicate sku", body: `{"customer":"Andi","items":[{"sku":"KOP-001","qty":1},{"sku":"KOP-001","qty":2}]}`, contains: "duplicate sku"},
		{name: "unknown sku", body: `{"customer":"Andi","items":[{"sku":"NOPE-001","qty":1}]}`, contains: "unknown sku NOPE-001"},
		{name: "client price", body: `{"customer":"Andi","items":[{"sku":"KOP-001","qty":1,"unit_price":1}]}`, contains: "invalid JSON body"},
		{name: "malformed", body: `{`, contains: "invalid JSON body"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doRequest(t, handler, http.MethodPost, "/api/orders", tt.body)
			if rec.Code != 400 {
				t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tt.contains) {
				t.Errorf("body = %s, want it to contain %q", rec.Body.String(), tt.contains)
			}
		})
	}

	ctx := context.Background()
	var stock, orders int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'KOP-001'`).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sales_orders`).Scan(&orders); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if stock != 10 || orders != 0 {
		t.Errorf("validation wrote state: stock %d, %d orders", stock, orders)
	}
}

func TestCreateOrderInsufficientWritesNothing(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPost, "/api/orders",
		`{"customer":"Andi","items":[{"sku":"KOP-001","qty":2},{"sku":"GULA-001","qty":5}]}`)
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			SKU       string `json:"sku"`
			Available int    `json:"available"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Error.Code != "INSUFFICIENT_STOCK" || body.Error.SKU != "GULA-001" || body.Error.Available != 3 {
		t.Errorf("error = %+v, want INSUFFICIENT_STOCK for GULA-001 with available 3", body.Error)
	}

	ctx := context.Background()
	var kopStock, gulaStock, orders, items, movements int
	for query, dest := range map[string]*int{
		`SELECT stock FROM products WHERE sku = 'KOP-001'`:  &kopStock,
		`SELECT stock FROM products WHERE sku = 'GULA-001'`: &gulaStock,
		`SELECT count(*) FROM sales_orders`:                 &orders,
		`SELECT count(*) FROM sales_order_items`:            &items,
		`SELECT count(*) FROM stock_movements`:              &movements,
	} {
		if err := pool.QueryRow(ctx, query).Scan(dest); err != nil {
			t.Fatalf("query %s: %v", query, err)
		}
	}
	if kopStock != 10 || gulaStock != 3 || orders != 0 || items != 0 || movements != 0 {
		t.Errorf("failed order changed state: %d/%d stock, %d orders, %d items, %d movements",
			kopStock, gulaStock, orders, items, movements)
	}
}

func TestConcurrentOrdersDoNotOversell(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)

	const attempts = 20
	var succeeded, conflicted, unexpected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/api/orders",
				strings.NewReader(`{"customer":"Andi","items":[{"sku":"KOP-001","qty":1}]}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			switch rec.Code {
			case 201:
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

	ctx := context.Background()
	var stock, orders, items int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'KOP-001'`).Scan(&stock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sales_orders`).Scan(&orders); err != nil {
		t.Fatalf("count orders: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM sales_order_items`).Scan(&items); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if stock != 0 || orders != 10 || items != 10 {
		t.Errorf("db state = stock %d, %d orders, %d items", stock, orders, items)
	}
}
