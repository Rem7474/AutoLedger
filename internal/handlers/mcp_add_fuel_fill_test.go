package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/models"
)

func TestMCPAddFuelFillWorksOutChecksRecordsAndReplays(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	ice := &models.Vehicle{UserID: f.owner.ID, Name: "Diesel", Currency: "GBP", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone}
	if err := f.repo.CreateVehicle(ctx, ice); err != nil {
		t.Fatal(err)
	}
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(token string, args map[string]any) (*mcp.CallToolResult, IntegrationFuelFillRecorded) {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: "add_fuel_fill", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		var out IntegrationFuelFillRecorded
		if !res.IsError {
			raw, _ := json.Marshal(res.StructuredContent)
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatal(err)
			}
		}
		return res, out
	}
	reason := func(res *mcp.CallToolResult) string { return res.Content[0].(*mcp.TextContent).Text }
	count := func() int {
		list, err := f.repo.ListFuelLogs(ctx, ice.ID)
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}
	args := func(extra map[string]any) map[string]any {
		a := map[string]any{"vehicle_id": ice.ID, "date": "2026-09-01", "liters": 40.0, "price_per_liter": 1.5, "odometer_km": 50000.0, "fuel_type": "DIESEL", "idempotency_key": "fuel-1"}
		for k, v := range extra {
			a[k] = v
		}
		return a
	}

	if res, _ := call(f.readTok, args(nil)); !res.IsError || !strings.Contains(reason(res), "can only read") {
		t.Fatalf("a read connection must not record: %+v", res)
	}

	res, preview := call(writeTok, args(map[string]any{"dry_run": true}))
	if res.IsError || preview.Recorded || preview.Fill.ID != "" || preview.Fill.Amount.Amount != "60.00" || !preview.Fill.IsFullTank {
		t.Fatalf("a dry run works the amount out and records nothing: %+v %+v", res, preview)
	}
	if count() != 0 {
		t.Fatal("a dry run recorded a fill-up")
	}

	res, first := call(writeTok, args(nil))
	if res.IsError || !first.Recorded || first.Fill.ID == "" || count() != 1 || first.Fill.Odometer == nil || first.Fill.Odometer.Value != 50000 {
		t.Fatalf("the call records: %+v %+v", res, first)
	}
	if res, again := call(writeTok, args(nil)); res.IsError || again.Fill.ID != first.Fill.ID || count() != 1 {
		t.Fatalf("the retry returns the first answer: %+v %+v", res, again)
	}

	for name, bad := range map[string]map[string]any{
		"lower mileage on a later day": {"date": "2026-09-10", "odometer_km": 40000.0, "idempotency_key": "fuel-2"},
		"unknown fuel":                 {"fuel_type": "WATER", "idempotency_key": "fuel-3"},
		"no amount at all":             {"liters": nil, "price_per_liter": nil, "idempotency_key": "fuel-4"},
		"not a day":                    {"date": "01/09/2026", "idempotency_key": "fuel-5"},
	} {
		if res, _ := call(writeTok, args(bad)); !res.IsError {
			t.Errorf("%s must be refused", name)
		}
	}
	if count() != 1 {
		t.Fatalf("a refused call recorded a fill-up: %d", count())
	}

	// An electric vehicle is refused
	if res, _ := call(writeTok, args(map[string]any{"vehicle_id": f.mine.ID, "idempotency_key": "fuel-6"})); f.mine.Powertrain != models.PowertrainICE && (!res.IsError || !strings.Contains(reason(res), "electric")) {
		t.Fatalf("an electric vehicle has no fill-up: %+v", res)
	}
}
