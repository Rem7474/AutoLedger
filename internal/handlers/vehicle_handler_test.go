package handlers

import "testing"

func TestNormalizeVehicleCurrency(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"rejects empty, there is no default currency", "", "", true},
		{"rejects blank", "  ", "", true},
		{"uppercases and trims", " usd ", "USD", false},
		{"already valid", "GBP", "GBP", false},
		{"rejects too short", "US", "", true},
		{"rejects too long", "USDD", "", true},
		{"rejects digits", "US1", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeVehicleCurrency(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for %q", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
