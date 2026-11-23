package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

func TestMCPDeleteRecordShowsThenDeletesAtTheVersionRead(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	session := f.connect(t, writeTok)
	call := func(token, name string, args map[string]any) (*mcp.CallToolResult, map[string]any) {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		var out map[string]any
		if !res.IsError {
			raw, _ := json.Marshal(res.StructuredContent)
			_ = json.Unmarshal(raw, &out)
		}
		return res, out
	}
	_ = session
	reason := func(res *mcp.CallToolResult) string { return res.Content[0].(*mcp.TextContent).Text }

	ice := &models.Vehicle{UserID: f.owner.ID, Name: "Diesel", Currency: "GBP", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone}
	if err := f.repo.CreateVehicle(ctx, ice); err != nil {
		t.Fatal(err)
	}
	res, expense := call(writeTok, "create_expense", map[string]any{"vehicle_id": f.mine.ID, "category": "tax", "amount": "40.00", "date": "2026-09-01", "description": "Road tax"})
	if res.IsError {
		t.Fatal(reason(res))
	}
	res, fill := call(writeTok, "add_fuel_fill", map[string]any{"vehicle_id": ice.ID, "date": "2026-09-02", "amount": "50.00", "liters": 35.0})
	if res.IsError {
		t.Fatal(reason(res))
	}
	res, charge := call(writeTok, "add_charge", map[string]any{"vehicle_id": f.mine.ID, "date": "2026-09-03", "kwh_added": 20.0, "cost": "6.00"})
	if res.IsError {
		t.Fatal(reason(res))
	}
	expenseID := expense["expense_id"].(string)
	fillID := fill["fill"].(map[string]any)["id"].(string)
	chargeID := charge["id"].(string)

	cases := []struct{ name, vehicle, kind, id string }{
		{"expense", f.mine.ID, "expense", expenseID},
		{"fill-up", ice.ID, "fuel_fill_up", fillID},
		{"charge", f.mine.ID, "charge", chargeID},
	}
	for _, c := range cases {
		args := func(extra map[string]any) map[string]any {
			a := map[string]any{"vehicle_id": c.vehicle, "kind": c.kind, "id": c.id}
			for k, v := range extra {
				a[k] = v
			}
			return a
		}
		if res, _ := call(f.readTok, "delete_record", args(map[string]any{"dry_run": true})); !res.IsError || !strings.Contains(reason(res), "can only read") {
			t.Fatalf("%s: a read connection must not delete: %+v", c.name, res)
		}
		res, preview := call(writeTok, "delete_record", args(map[string]any{"dry_run": true}))
		if res.IsError || preview["deleted"] != false {
			t.Fatalf("%s: a dry run deletes nothing: %+v %v", c.name, res, preview)
		}
		version := preview["record"].(map[string]any)["version"].(float64)
		if res, _ := call(writeTok, "delete_record", args(nil)); !res.IsError || !strings.Contains(reason(res), "expected_version") {
			t.Fatalf("%s: a deletion needs the version: %+v", c.name, res)
		}
		if res, _ := call(writeTok, "delete_record", args(map[string]any{"expected_version": version + 1})); !res.IsError || !strings.Contains(reason(res), "changed since") {
			t.Fatalf("%s: a wrong version is refused: %+v", c.name, res)
		}
		if _, err := f.repo.FindRecord(ctx, c.kind, c.vehicle, c.id); err != nil {
			t.Fatalf("%s: a refused call deleted the entry: %v", c.name, err)
		}
		res, done := call(writeTok, "delete_record", args(map[string]any{"expected_version": version}))
		if res.IsError || done["deleted"] != true {
			t.Fatalf("%s: the call deletes: %+v %v", c.name, res, done)
		}
		if _, err := f.repo.FindRecord(ctx, c.kind, c.vehicle, c.id); !errors.Is(err, database.ErrNotFound) {
			t.Fatalf("%s: the entry is still there: %v", c.name, err)
		}
		if res, _ := call(writeTok, "delete_record", args(map[string]any{"expected_version": version})); !res.IsError || !strings.Contains(reason(res), "no such entry") {
			t.Fatalf("%s: deleting twice is refused: %+v", c.name, res)
		}
	}

	// The repository itself refuses a stale version, and a kind or an entry it does not know
	res, again := call(writeTok, "create_expense", map[string]any{"vehicle_id": f.mine.ID, "category": "tax", "amount": "41.00", "date": "2026-09-01", "description": "Other tax"})
	if res.IsError {
		t.Fatal(reason(res))
	}
	id := again["expense_id"].(string)
	if err := f.repo.DeleteRecordAtVersion(ctx, database.RecordExpense, f.mine.ID, id, 99); !errors.Is(err, database.ErrStaleWrite) {
		t.Fatalf("a stale version: %v", err)
	}
	if err := f.repo.DeleteRecordAtVersion(ctx, "nothing", f.mine.ID, id, 1); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("an unknown kind: %v", err)
	}
	if res, _ := call(writeTok, "delete_record", map[string]any{"vehicle_id": f.mine.ID, "kind": "drive", "id": id, "dry_run": true}); !res.IsError {
		t.Fatal("a kind that cannot be deleted must be refused")
	}
	if res, _ := call(writeTok, "delete_record", map[string]any{"vehicle_id": f.theirs.ID, "kind": "expense", "id": id, "dry_run": true}); !res.IsError {
		t.Fatal("another account's vehicle must be refused")
	}
}
