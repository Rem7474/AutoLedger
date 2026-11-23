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

func TestMCPAddChargeChecksRecordsAndReplays(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(token string, args map[string]any) (*mcp.CallToolResult, IntegrationChargeRecorded) {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: "add_charge", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		var out IntegrationChargeRecorded
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
		_, total, err := f.repo.ListCharges(ctx, f.mine.ID, false, 100, 0)
		if err != nil {
			t.Fatal(err)
		}
		return total
	}
	args := func(extra map[string]any) map[string]any {
		a := map[string]any{"vehicle_id": f.mine.ID, "date": "2026-09-01", "kwh_added": 30.5, "cost": "7.80", "notes": " Public terminal ", "idempotency_key": "chg-1"}
		for k, v := range extra {
			a[k] = v
		}
		return a
	}
	start := count()

	if res, _ := call(f.readTok, args(nil)); !res.IsError || !strings.Contains(reason(res), "can only read") {
		t.Fatalf("a read connection must not record: %+v", res)
	}

	res, preview := call(writeTok, args(map[string]any{"dry_run": true}))
	if res.IsError || preview.Recorded || preview.ID != "" || preview.Cost.Amount != "7.80" || preview.Cost.Currency != "GBP" || preview.Notes == nil || *preview.Notes != "Public terminal" {
		t.Fatalf("a dry run shows the charge and records nothing: %+v %+v", res, preview)
	}
	if count() != start {
		t.Fatal("a dry run recorded a charge")
	}

	res, first := call(writeTok, args(nil))
	if res.IsError || !first.Recorded || first.ID == "" || count() != start+1 {
		t.Fatalf("the call records: %+v %+v", res, first)
	}
	charges, _, err := f.repo.ListCharges(ctx, f.mine.ID, false, 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	var stored bool
	for _, c := range charges {
		if c.ID == first.ID && c.IsManual && c.KwhAdded == 30.5 && c.Cost != nil && c.Cost.String() == "7.80" {
			stored = true
		}
	}
	if !stored {
		t.Fatalf("stored charges: %+v", charges)
	}
	if res, again := call(writeTok, args(nil)); res.IsError || again.ID != first.ID || count() != start+1 {
		t.Fatalf("the retry returns the first answer: %+v %+v", res, again)
	}

	for name, bad := range map[string]map[string]any{
		"no cost":           {"cost": "", "idempotency_key": "chg-2"},
		"no energy":         {"kwh_added": 0.0, "idempotency_key": "chg-3"},
		"not a day":         {"date": "01/09/2026", "idempotency_key": "chg-4"},
		"battery goes down": {"start_battery_level": 80, "end_battery_level": 20, "idempotency_key": "chg-5"},
		"negative cost":     {"cost": "-1.00", "idempotency_key": "chg-6"},
	} {
		if res, _ := call(writeTok, args(bad)); !res.IsError {
			t.Errorf("%s must be refused", name)
		}
	}
	if count() != start+1 {
		t.Fatalf("a refused call recorded a charge: %d", count())
	}

	// A combustion vehicle does not charge
	ice := &models.Vehicle{UserID: f.owner.ID, Name: "Diesel", Currency: "GBP", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone}
	if err := f.repo.CreateVehicle(ctx, ice); err != nil {
		t.Fatal(err)
	}
	if res, _ := call(writeTok, args(map[string]any{"vehicle_id": ice.ID, "idempotency_key": "chg-7"})); !res.IsError || !strings.Contains(reason(res), "does not charge") {
		t.Fatalf("a combustion vehicle has no charge: %+v", res)
	}
}
