package product

import (
	"net/http"

	"stockflow/backend/internal/api"
	"stockflow/backend/internal/apierr"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	params, err := ParseListParams(r.URL.Query())
	if err != nil {
		api.WriteError(w, apierr.BadRequest(err.Error(), nil))
		return
	}

	result, err := h.store.List(r.Context(), params)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, result)
}
