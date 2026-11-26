package product

import (
	"net/url"
	"testing"
)

func TestParseListParams(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    ListParams
		wantErr bool
	}{
		{
			name:  "defaults",
			query: "",
			want:  ListParams{Search: "", Page: 1, Limit: defaultLimit},
		},
		{
			name:  "search and paging",
			query: "search=kopi&page=3&limit=5",
			want:  ListParams{Search: "kopi", Page: 3, Limit: 5},
		},
		{
			name:    "page zero",
			query:   "page=0",
			wantErr: true,
		},
		{
			name:    "limit above max",
			query:   "limit=101",
			wantErr: true,
		},
		{
			name:    "non integer",
			query:   "page=abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("parse query: %v", err)
			}
			got, err := ParseListParams(q)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseListParams() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseListParams() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	got := escapeLike(`50%_off\now`)
	want := `50\%\_off\\now`
	if got != want {
		t.Errorf("escapeLike() = %q, want %q", got, want)
	}
}
