package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"stockflow/backend/internal/apierr"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("encode response: %v", err)
	}
}

func WriteError(w http.ResponseWriter, err error) {
	var ae *apierr.Error
	if !errors.As(err, &ae) {
		log.Printf("unexpected error: %v", err)
		ae = apierr.Internal()
	}

	body := make(map[string]any, len(ae.Fields)+2)
	body["code"] = ae.Code
	body["message"] = ae.Message
	for k, v := range ae.Fields {
		body[k] = v
	}
	WriteJSON(w, ae.Status, map[string]any{"error": body})
}
