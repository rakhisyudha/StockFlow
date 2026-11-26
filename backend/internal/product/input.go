package product

import (
	"net/http"
	"strconv"
	"strings"

	"stockflow/backend/internal/api"
	"stockflow/backend/internal/apierr"
)

type Input struct {
	SKU      string `json:"sku"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	IsActive *bool  `json:"is_active"`
}

type StockInput struct {
	Type string `json:"type"`
	Qty  int    `json:"qty"`
	Note string `json:"note"`
}

func decodeInput(w http.ResponseWriter, r *http.Request) (Input, error) {
	var in Input
	if err := api.DecodeJSON(w, r, &in); err != nil {
		return in, err
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

func decodeStockInput(w http.ResponseWriter, r *http.Request) (StockInput, error) {
	var in StockInput
	if err := api.DecodeJSON(w, r, &in); err != nil {
		return in, err
	}

	var details []string
	if in.Type != "in" && in.Type != "out" {
		details = append(details, "type must be 'in' or 'out'")
	}
	if in.Qty <= 0 {
		details = append(details, "qty must be an integer > 0")
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
