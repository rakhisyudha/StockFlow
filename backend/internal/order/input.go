package order

import (
	"fmt"
	"net/http"
	"strings"

	"stockflow/backend/internal/api"
	"stockflow/backend/internal/apierr"
)

func decodeCreateInput(w http.ResponseWriter, r *http.Request) (CreateInput, error) {
	var in CreateInput
	if err := api.DecodeJSON(w, r, &in); err != nil {
		return in, err
	}
	in.Customer = strings.TrimSpace(in.Customer)

	var details []string
	if in.Customer == "" {
		details = append(details, "customer is required")
	}
	if len(in.Items) == 0 {
		details = append(details, "items must contain at least one item")
	}

	seen := make(map[string]bool, len(in.Items))
	for i := range in.Items {
		item := &in.Items[i]
		item.SKU = strings.TrimSpace(item.SKU)
		if item.SKU == "" {
			details = append(details, fmt.Sprintf("items[%d].sku is required", i))
		}
		if item.Qty <= 0 {
			details = append(details, fmt.Sprintf("items[%d].qty must be an integer > 0", i))
		}
		if item.SKU != "" {
			if seen[item.SKU] {
				details = append(details, fmt.Sprintf("duplicate sku %s", item.SKU))
			}
			seen[item.SKU] = true
		}
	}

	if len(details) > 0 {
		return in, apierr.BadRequest("validation failed", map[string]any{"details": details})
	}
	return in, nil
}
