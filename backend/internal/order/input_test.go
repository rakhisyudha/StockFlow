package order

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"stockflow/backend/internal/apierr"
)

func TestDecodeCreateInput(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantErr  bool
		contains string
	}{
		{
			name: "valid",
			body: `{"customer":"Andi","items":[{"sku":"KOP-001","qty":2},{"sku":"TEH-001","qty":1}]}`,
		},
		{
			name:     "empty customer",
			body:     `{"customer":"  ","items":[{"sku":"KOP-001","qty":1}]}`,
			wantErr:  true,
			contains: "customer is required",
		},
		{
			name:     "empty items",
			body:     `{"customer":"Andi","items":[]}`,
			wantErr:  true,
			contains: "items must contain at least one item",
		},
		{
			name:     "qty zero",
			body:     `{"customer":"Andi","items":[{"sku":"KOP-001","qty":0}]}`,
			wantErr:  true,
			contains: "items[0].qty must be an integer > 0",
		},
		{
			name:     "duplicate sku",
			body:     `{"customer":"Andi","items":[{"sku":"KOP-001","qty":1},{"sku":"KOP-001","qty":2}]}`,
			wantErr:  true,
			contains: "duplicate sku KOP-001",
		},
		{
			name:     "client supplied price",
			body:     `{"customer":"Andi","items":[{"sku":"KOP-001","qty":1,"unit_price":1}]}`,
			wantErr:  true,
			contains: "invalid JSON body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			in, err := decodeCreateInput(httptest.NewRecorder(), req)
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if in.Customer != "Andi" || len(in.Items) == 0 {
					t.Errorf("input = %+v", in)
				}
				return
			}
			var ae *apierr.Error
			if !errors.As(err, &ae) {
				t.Fatalf("error = %T, want *apierr.Error", err)
			}
			haystack := err.Error()
			if details, ok := ae.Fields["details"].([]string); ok {
				haystack += " " + strings.Join(details, " ")
			}
			if !strings.Contains(haystack, tt.contains) {
				t.Errorf("error = %q, want it to contain %q", haystack, tt.contains)
			}
		})
	}
}
