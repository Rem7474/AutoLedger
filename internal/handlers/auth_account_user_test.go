package handlers

import (
	"encoding/json"
	"testing"

	"github.com/teslacost/teslacost/internal/config"
	"github.com/teslacost/teslacost/internal/models"
)

func TestAccountUserCarriesReportingTimezone(t *testing.T) {
	cases := []struct{ name, configured, want string }{
		{"configured zone", "Europe/Paris", "Europe/Paris"},
		{"another zone", "America/Los_Angeles", "America/Los_Angeles"},
		{"unknown zone falls back to UTC like the reporting services", "Not/AZone", "UTC"},
		{"empty falls back to UTC", "", "UTC"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := NewAuthHandler(nil, &config.Config{ReportingTimezone: c.configured}, nil)
			raw, err := json.Marshal(h.accountUser(&models.User{ID: "u1", Email: "a@b.c", DistanceUnit: "km"}))
			if err != nil {
				t.Fatal(err)
			}
			var got map[string]any
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatal(err)
			}
			if got["reporting_timezone"] != c.want {
				t.Fatalf("reporting_timezone = %v, want %s", got["reporting_timezone"], c.want)
			}
			if got["email"] != "a@b.c" || got["distance_unit"] != "km" {
				t.Fatalf("stored user fields lost: %v", got)
			}
		})
	}
}
