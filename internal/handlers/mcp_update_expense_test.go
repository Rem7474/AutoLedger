package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
)

func TestMCPUpdateExpenseChangesWhatIsNamedAtTheVersionRead(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
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
	reason := func(res *mcp.CallToolResult) string { return res.Content[0].(*mcp.TextContent).Text }

	res, created := call(writeTok, "create_expense", map[string]any{"vehicle_id": f.mine.ID, "category": "insurance", "amount": "300.00", "date": "2026-03-01", "description": "Insurance", "repeats_every_months": 12})
	if res.IsError {
		t.Fatal(reason(res))
	}
	id := created["expense_id"].(string)
	if created["expense"].(map[string]any)["id"] != id {
		t.Fatalf("the created entry carries its id: %v", created)
	}
	args := func(extra map[string]any) map[string]any {
		a := map[string]any{"vehicle_id": f.mine.ID, "expense_id": id}
		for k, v := range extra {
			a[k] = v
		}
		return a
	}

	if res, _ := call(f.readTok, "update_expense", args(map[string]any{"amount": "320.00", "dry_run": true})); !res.IsError || !strings.Contains(reason(res), "can only read") {
		t.Fatalf("a read connection must not change: %+v", res)
	}
	if res, _ := call(writeTok, "update_expense", args(map[string]any{"dry_run": true})); !res.IsError || !strings.Contains(reason(res), "nothing to change") {
		t.Fatalf("an empty change is refused: %+v", res)
	}
	res, preview := call(writeTok, "update_expense", args(map[string]any{"amount": "320.00", "description": " Insurance 2026 ", "dry_run": true}))
	if res.IsError || preview["updated"] != false {
		t.Fatalf("a dry run changes nothing: %+v %v", res, preview)
	}
	version := preview["version"].(float64)
	after := preview["after"].(map[string]any)
	if after["amount"].(map[string]any)["amount"] != "320.00" || after["note"] != "Insurance 2026" || after["recurring"] != true || after["date"] != "2026-03-01" {
		t.Fatalf("the preview applies the change and keeps the rest: %v", after)
	}
	if res, _ := call(writeTok, "update_expense", args(map[string]any{"amount": "320.00"})); !res.IsError || !strings.Contains(reason(res), "expected_version") {
		t.Fatalf("saving needs the version: %+v", res)
	}
	if res, _ := call(writeTok, "update_expense", args(map[string]any{"amount": "320.00", "expected_version": version + 1})); !res.IsError || !strings.Contains(reason(res), "changed since") {
		t.Fatalf("a wrong version is refused: %+v", res)
	}
	if m, err := f.repo.GetMaintenanceExpense(ctx, f.mine.ID, id); err != nil || m.Amount.String() != "300.00" {
		t.Fatalf("a refused call changed the expense: %+v %v", m, err)
	}

	res, saved := call(writeTok, "update_expense", args(map[string]any{"amount": "320.00", "description": " Insurance 2026 ", "repeats_until": "2030-03-01", "expected_version": version}))
	if res.IsError || saved["updated"] != true || saved["version"].(float64) != version+1 {
		t.Fatalf("the call saves: %+v %v", res, saved)
	}
	m, err := f.repo.GetMaintenanceExpense(ctx, f.mine.ID, id)
	if err != nil || m.Amount.String() != "320.00" || m.Description != "Insurance 2026" || !m.IsRecurring || m.RecurrenceEndDate == nil || m.Date.Format("2006-01-02") != "2026-03-01" || m.Category != "INSURANCE" {
		t.Fatalf("stored: %+v %v", m, err)
	}
	// The same version is now stale
	if res, _ := call(writeTok, "update_expense", args(map[string]any{"amount": "330.00", "expected_version": version})); !res.IsError || !strings.Contains(reason(res), "changed since") {
		t.Fatalf("a replay is refused: %+v", res)
	}

	// A single expense again, and what an expense cannot become
	res, single := call(writeTok, "update_expense", args(map[string]any{"repeats_every_months": 0, "expected_version": version + 1}))
	if res.IsError || single["after"].(map[string]any)["recurring"] != false {
		t.Fatalf("0 makes it a single expense: %+v %v", res, single)
	}
	for name, bad := range map[string]map[string]any{
		"category of a trip": {"category": "toll"},
		"zero amount":        {"amount": "0"},
		"empty description":  {"description": " "},
		"not a day":          {"date": "01/03/2026"},
		"end of a single":    {"repeats_until": "2030-01-01"},
	} {
		bad["dry_run"] = true
		if res, _ := call(writeTok, "update_expense", args(bad)); !res.IsError {
			t.Errorf("%s must be refused", name)
		}
	}
	if res, _ := call(writeTok, "update_expense", map[string]any{"vehicle_id": f.mine.ID, "expense_id": "00000000-0000-4000-8000-000000000001", "amount": "1.00", "dry_run": true}); !res.IsError || !strings.Contains(reason(res), "no such expense") {
		t.Fatalf("an unknown expense: %+v", res)
	}
}
