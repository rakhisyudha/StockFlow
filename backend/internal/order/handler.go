package order

import (
	"net/http"

	"stockflow/backend/internal/api"
)

type Handler struct{ store *Store }

func NewHandler(store *Store) *Handler { return &Handler{store: store} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	in, err := decodeCreateInput(w, r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	o, err := h.store.Create(r.Context(), in)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusCreated, o)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseID(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	o, err := h.store.Get(r.Context(), id)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, o)
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := api.ParseID(r)
	if err != nil {
		api.WriteError(w, err)
		return
	}

	o, err := h.store.Cancel(r.Context(), id)
	if err != nil {
		api.WriteError(w, err)
		return
	}
	api.WriteJSON(w, http.StatusOK, o)
}
