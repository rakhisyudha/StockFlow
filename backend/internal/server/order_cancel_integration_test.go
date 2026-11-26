package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"stockflow/backend/internal/order"
)

func createTestOrder(t *testing.T, h http.Handler) order.Order {
	t.Helper()
	rec := doRequest(t, h, http.MethodPost, "/api/orders",
		`{"customer":"Andi","items":[{"sku":"KOP-001","qty":2},{"sku":"TEH-001","qty":1}]}`)
	if rec.Code != 201 {
		t.Fatalf("create order status = %d; body = %s", rec.Code, rec.Body.String())
	}
	var got order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	return got
}

func orderStatus(t *testing.T, h http.Handler, id int) int {
	t.Helper()
	return doRequest(t, h, http.MethodGet, "/api/orders/"+strconv.Itoa(id), "").Code
}

func TestGetOrder(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)
	created := createTestOrder(t, handler)

	rec := doRequest(t, handler, http.MethodGet, "/api/orders/"+strconv.Itoa(created.ID), "")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.ID != created.ID || got.Customer != "Andi" || got.Status != "CREATED" || got.Total != 65000 {
		t.Errorf("order = %+v", got)
	}
	if len(got.Items) != 2 || got.Items[0].UnitPrice != 25000 {
		t.Errorf("items = %+v", got.Items)
	}

	if orderStatus(t, handler, 9999) != 404 {
		t.Error("missing order should return 404")
	}
	if doRequest(t, handler, http.MethodGet, "/api/orders/abc", "").Code != 400 {
		t.Error("invalid id should return 400")
	}
}

func TestCancelOrderRestoresStock(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)
	created := createTestOrder(t, handler)

	rec := doRequest(t, handler, http.MethodPost, "/api/orders/"+strconv.Itoa(created.ID)+"/cancel", "")
	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got order.Order
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if got.Status != "CANCELLED" || got.CancelledAt == nil || got.Total != 65000 {
		t.Errorf("cancelled order = %+v", got)
	}

	ctx := context.Background()
	var kopStock, tehStock int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'KOP-001'`).Scan(&kopStock); err != nil {
		t.Fatalf("read kop stock: %v", err)
	}
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'TEH-001'`).Scan(&tehStock); err != nil {
		t.Fatalf("read teh stock: %v", err)
	}
	if kopStock != 10 || tehStock != 5 {
		t.Errorf("stock after cancel = %d/%d, want 10/5", kopStock, tehStock)
	}
}

func TestCancelTwiceRejected(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)
	created := createTestOrder(t, handler)

	path := "/api/orders/" + strconv.Itoa(created.ID) + "/cancel"
	if rec := doRequest(t, handler, http.MethodPost, path, ""); rec.Code != 200 {
		t.Fatalf("first cancel status = %d", rec.Code)
	}

	rec := doRequest(t, handler, http.MethodPost, path, "")
	if rec.Code != 409 {
		t.Fatalf("second cancel status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ORDER_ALREADY_CANCELLED") {
		t.Errorf("body = %s", rec.Body.String())
	}

	ctx := context.Background()
	var kopStock int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'KOP-001'`).Scan(&kopStock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if kopStock != 10 {
		t.Errorf("stock restored twice: %d", kopStock)
	}
}

func TestCancelMissingOrder(t *testing.T) {
	pool := testPool(t)
	handler := testHandler(pool)

	rec := doRequest(t, handler, http.MethodPost, "/api/orders/9999/cancel", "")
	if rec.Code != 404 {
		t.Fatalf("status = %d, want 404; body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "ORDER_NOT_FOUND") {
		t.Errorf("body = %s", rec.Body.String())
	}
}

func TestConcurrentCancelRestoresOnce(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)
	created := createTestOrder(t, handler)

	path := "/api/orders/" + strconv.Itoa(created.ID) + "/cancel"
	var succeeded, conflicted, unexpected atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
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

	if unexpected.Load() != 0 || succeeded.Load() != 1 || conflicted.Load() != 19 {
		t.Errorf("succeeded = %d, conflicted = %d, unexpected = %d", succeeded.Load(), conflicted.Load(), unexpected.Load())
	}

	ctx := context.Background()
	var kopStock int
	if err := pool.QueryRow(ctx, `SELECT stock FROM products WHERE sku = 'KOP-001'`).Scan(&kopStock); err != nil {
		t.Fatalf("read stock: %v", err)
	}
	if kopStock != 10 {
		t.Errorf("stock = %d, want 10 after a single restore", kopStock)
	}
}

func TestDeleteProductWithOrderHistory(t *testing.T) {
	pool := testPool(t)
	seedOrderProducts(t, pool)
	handler := testHandler(pool)
	createTestOrder(t, handler)

	rec := doRequest(t, handler, http.MethodDelete, "/api/products/1", "")
	if rec.Code != 409 {
		t.Fatalf("status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "PRODUCT_HAS_HISTORY") {
		t.Errorf("body = %s", rec.Body.String())
	}
}
