package server

import (
	"net/http"

	"stockflow/backend/internal/api"
	"stockflow/backend/internal/product"
)

func New(products *product.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthz)
	mux.HandleFunc("GET /api/products", products.List)
	return mux
}

func healthz(w http.ResponseWriter, _ *http.Request) {
	api.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
