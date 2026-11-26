package server

import (
	"net/http"

	"stockflow/backend/internal/api"
	"stockflow/backend/internal/order"
	"stockflow/backend/internal/product"
)

func New(products *product.Handler, orders *order.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /api/products", products.List)
	mux.HandleFunc("POST /api/products", products.Create)
	mux.HandleFunc("PUT /api/products/{id}", products.Update)
	mux.HandleFunc("DELETE /api/products/{id}", products.Delete)
	mux.HandleFunc("POST /api/products/{id}/stock", products.AdjustStock)
	mux.HandleFunc("GET /api/products/{id}/movements", products.Movements)
	mux.HandleFunc("POST /api/orders", orders.Create)
	mux.HandleFunc("GET /api/orders/{id}", orders.Get)
	mux.HandleFunc("POST /api/orders/{id}/cancel", orders.Cancel)
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
