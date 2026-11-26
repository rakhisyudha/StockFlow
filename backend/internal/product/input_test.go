package product

import (
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"stockflow/backend/internal/apierr"
)

func TestDecodeInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantDetails []string
	}{
		{
			name:        "all fields invalid",
			body:        `{"sku":"  ","name":"","price":-1}`,
			wantDetails: []string{"sku is required", "name is required", "price must be an integer >= 0"},
		},
		{
			name:        "negative price only",
			body:        `{"sku":"A","name":"B","price":-1}`,
			wantDetails: []string{"price must be an integer >= 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			_, err := decodeInput(httptest.NewRecorder(), req)

			var ae *apierr.Error
			if !errors.As(err, &ae) {
				t.Fatalf("error = %v, want *apierr.Error", err)
			}
			details, ok := ae.Fields["details"].([]string)
			if !ok {
				t.Fatalf("details = %T, want []string", ae.Fields["details"])
			}
			if !reflect.DeepEqual(details, tt.wantDetails) {
				t.Errorf("details = %v, want %v", details, tt.wantDetails)
			}
		})
	}
}

func TestDecodeInputTrimsStrings(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"sku":"  KOP-1 ","name":"  Kopi  ","price":0}`))
	in, err := decodeInput(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if in.SKU != "KOP-1" || in.Name != "Kopi" {
		t.Errorf("input = %+v, want trimmed strings", in)
	}
	if in.IsActive != nil {
		t.Errorf("is_active = %v, want nil when omitted", *in.IsActive)
	}
}
