package product

import (
	"errors"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"stockflow/backend/internal/apierr"
)

func TestDecodeStockInputValidation(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantDetails []string
	}{
		{
			name:        "all fields invalid",
			body:        `{"type":"sideways","qty":0}`,
			wantDetails: []string{"type must be 'in' or 'out'", "qty must be an integer > 0"},
		},
		{
			name:        "negative qty",
			body:        `{"type":"out","qty":-3}`,
			wantDetails: []string{"qty must be an integer > 0"},
		},
		{
			name:        "unknown field",
			body:        `{"type":"in","qty":1,"unit_price":10}`,
			wantDetails: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.body))
			_, err := decodeStockInput(httptest.NewRecorder(), req)
			if err == nil {
				t.Fatal("expected an error")
			}
			var ae *apierr.Error
			if !errors.As(err, &ae) {
				t.Fatalf("error = %T, want *apierr.Error", err)
			}
			if tt.wantDetails == nil {
				return
			}
			details, _ := ae.Fields["details"].([]string)
			if !reflect.DeepEqual(details, tt.wantDetails) {
				t.Errorf("details = %v, want %v", details, tt.wantDetails)
			}
		})
	}
}

func TestDecodeStockInputValid(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"type":"out","qty":7,"note":"damaged"}`))
	in, err := decodeStockInput(httptest.NewRecorder(), req)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if in.Type != "out" || in.Qty != 7 || in.Note != "damaged" {
		t.Errorf("input = %+v", in)
	}
}
