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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	in, err := decodeInput(w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	p, err := h.store.Create(r.Context(), in)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	in, err := decodeInput(w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	p, err := h.store.Update(r.Context(), id, in)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, p)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	if err := h.store.Delete(r.Context(), id); err != nil {
		api.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AdjustStock(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	in, err := decodeStockInput(w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	result, err := h.store.AdjustStock(r.Context(), id, in)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, result)
}

func (h *Handler) Movements(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	data, err := h.store.Movements(r.Context(), id)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, MovementListResponse{Data: data})
}
