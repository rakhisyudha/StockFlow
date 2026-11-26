package product

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"stockflow/backend/internal/apierr"
)

type Input struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	IsActive *bool  `json:"is_active"`
}

func decodeInput(w http.ResponseWriter, r *http.Request) (Input, error) {
	var in Input
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&in); err != nil {
		return in, apierr.BadRequest("invalid JSON body: "+err.Error(), nil)
	}
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return in, apierr.BadRequest("request body must contain a single JSON object", nil)
	}

	in.SKU = strings.TrimSpace(in.SKU)
	in.Name = strings.TrimSpace(in.Name)
	var details []string
	if in.SKU == "" {
		details = append(details, "sku is required")
	}
	if in.Name == "" {
		details = append(details, "name is required")
	}
	if in.Price < 0 {
		details = append(details, "price must be an integer >= 0")
	}
	if len(details) > 0 {
		return in, apierr.BadRequest("validation failed", map[string]any{"details": details})
	}
	return in, nil
}

func parseID(r *http.Request) (int, error) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		return 0, apierr.BadRequest("id must be a positive integer", nil)
	}
	return id, nil
}
