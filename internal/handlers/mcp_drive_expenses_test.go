package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/models"
)

// expenseFixture records two drives on the account's vehicle, the first with an estimated toll of 14.60.
func (f *mcpFixture) expenseFixture(t *testing.T) (first, second *models.Drive) {
	t.Helper()
	ctx := context.Background()
	km := 120.0
	day := time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC)
	record := func(start time.Time) *models.Drive {
		d := &models.Drive{VehicleID: f.mine.ID, StartTime: start, EndTime: start.Add(time.Hour), DistanceKm: &km, DurationMin: 60, StartAddress: strPtr("Lyon"), EndAddress: strPtr("Annecy")}
		if err := f.repo.CreateManualDrive(ctx, d); err != nil {
			t.Fatal(err)
		}
		return d
	}
	first, second = record(day), record(day.AddDate(0, 0, 1))
	toll := &models.DriveExpense{VehicleID: f.mine.ID, DriveID: &first.ID, Type: "TOLL", Amount: 1460, Currency: f.mine.Currency, Date: day, Source: models.ExpenseSourceAutoToll}
	if err := f.repo.SaveDriveExpense(ctx, toll, nil, ""); err != nil {
		t.Fatal(err)
	}
	return first, second
}

// tollOf is what the trip lists as travel_expenses: its tolls and its parking together.
func (f *mcpFixture) tollOf(t *testing.T, driveID string) string {
	t.Helper()
	got, err := f.repo.GetTollExpensesForDrives(context.Background(), f.mine.ID, []string{driveID})
	if err != nil {
		t.Fatal(err)
	}
	return got[driveID].String()
}

func TestMCPUpdateDriveExpensesChangesTheTollsAllOrNothing(t *testing.T) {
	f := newMCPFixture(t)
	first, second := f.expenseFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(token string, args map[string]any) (*mcp.CallToolResult, IntegrationDriveExpensesUpdated) {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: "update_drive_expenses", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		var out IntegrationDriveExpensesUpdated
		if !res.IsError {
			raw, _ := json.Marshal(res.StructuredContent)
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatal(err)
			}
		}
		return res, out
	}
	reason := func(res *mcp.CallToolResult) string { return res.Content[0].(*mcp.TextContent).Text }
	items := []map[string]any{
		{"drive_id": first.ID, "tolls": "12.41", "expected_tolls": "14.60"},
		{"drive_id": second.ID, "tolls": "7.00", "parking": "3.50", "expected_tolls": "0"},
	}
	args := map[string]any{"vehicle_id": f.mine.ID, "items": items}

	if res, _ := call(f.readTok, args); !res.IsError || !strings.Contains(reason(res), "can only read") {
		t.Fatalf("a read connection must not change expenses: %+v", res)
	}

	args["dry_run"] = true
	res, preview := call(writeTok, args)
	if res.IsError || preview.Saved || len(preview.Drives) != 2 {
		t.Fatalf("a dry run answers and saves nothing: %+v %+v", res, preview)
	}
	if c := preview.Drives[0].Changes[0]; c.Kind != "tolls" || c.Action != "updated" || c.Before.Amount != "14.60" || c.After.Amount != "12.41" {
		t.Errorf("the old and the new amount: %+v", c)
	}
	if c := preview.Drives[1].Changes; len(c) != 2 || c[0].Action != "created" || c[1].Kind != "parking" || c[1].After.Amount != "3.50" {
		t.Errorf("a trip without expense gets them: %+v", c)
	}
	if f.tollOf(t, first.ID) != "14.60" || f.tollOf(t, second.ID) != "0.00" {
		t.Fatalf("a dry run changed the records: %s %s", f.tollOf(t, first.ID), f.tollOf(t, second.ID))
	}

	delete(args, "dry_run")
	if res, out := call(writeTok, args); res.IsError || !out.Saved {
		t.Fatalf("a write connection saves: %+v", res)
	}
	if f.tollOf(t, first.ID) != "12.41" || f.tollOf(t, second.ID) != "10.50" {
		t.Errorf("the amounts replace what was held: %s %s", f.tollOf(t, first.ID), f.tollOf(t, second.ID))
	}
	manual, err := f.repo.ListDriveExpenses(ctx, f.mine.ID, "en")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range manual {
		if e.Source != models.ExpenseSourceManual {
			t.Errorf("an estimate overwritten is entered by the owner: %+v", e)
		}
	}

	// Another amount than the one read refuses the whole call: nothing of it is applied
	stale := map[string]any{"vehicle_id": f.mine.ID, "items": []map[string]any{
		{"drive_id": second.ID, "tolls": "9.00"},
		{"drive_id": first.ID, "tolls": "1.00", "expected_tolls": "14.60"},
	}}
	if res, _ := call(writeTok, stale); !res.IsError || !strings.Contains(reason(res), "12.41") || !strings.Contains(reason(res), "nothing was changed") {
		t.Fatalf("a changed trip refuses the call: %+v", res)
	}
	if f.tollOf(t, second.ID) != "10.50" {
		t.Errorf("a refused batch applied its first item: %s", f.tollOf(t, second.ID))
	}

	// Zero removes
	if res, out := call(writeTok, map[string]any{"vehicle_id": f.mine.ID, "items": []map[string]any{{"drive_id": first.ID, "tolls": "0"}}}); res.IsError || out.Drives[0].Changes[0].Action != "deleted" {
		t.Fatalf("zero removes the tolls: %+v", res)
	}
	if f.tollOf(t, first.ID) != "0.00" {
		t.Errorf("tolls removed: %s", f.tollOf(t, first.ID))
	}
	// The same amount changes nothing
	if _, out := call(writeTok, map[string]any{"vehicle_id": f.mine.ID, "items": []map[string]any{{"drive_id": second.ID, "tolls": "7.00"}}}); out.Drives[0].Changes[0].Action != "unchanged" {
		t.Errorf("same amount: %+v", out)
	}
}

func TestMCPUpdateDriveExpensesRefusesWhatItCannotSetAsOneAmount(t *testing.T) {
	f := newMCPFixture(t)
	first, second := f.expenseFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(items ...map[string]any) string {
		t.Helper()
		res, err := f.connect(t, writeTok).CallTool(ctx, &mcp.CallToolParams{Name: "update_drive_expenses", Arguments: map[string]any{"vehicle_id": f.mine.ID, "items": items}})
		if err != nil {
			t.Fatal(err)
		}
		if !res.IsError {
			return ""
		}
		return res.Content[0].(*mcp.TextContent).Text
	}

	// A cost shared by a trip group is one amount for several drives
	shared := &models.DriveExpense{VehicleID: f.mine.ID, Type: "PARKING", Amount: 900, Currency: f.mine.Currency, Date: first.StartTime}
	if err := f.repo.SaveDriveExpense(ctx, shared, []string{first.ID, second.ID}, "Weekend"); err != nil {
		t.Fatal(err)
	}
	if msg := call(map[string]any{"drive_id": first.ID, "parking": "4.00"}); !strings.Contains(msg, "shared by a trip group") {
		t.Errorf("a shared cost: %q", msg)
	}

	// Two tolls on a trip cannot be set as one
	third := &models.DriveExpense{VehicleID: f.mine.ID, DriveID: &first.ID, Type: "TOLL", Amount: 300, Currency: f.mine.Currency, Date: first.StartTime}
	if err := f.repo.SaveDriveExpense(ctx, third, nil, ""); err != nil {
		t.Fatal(err)
	}
	if msg := call(map[string]any{"drive_id": first.ID, "tolls": "5.00"}); !strings.Contains(msg, "several tolls expenses") {
		t.Errorf("several expenses: %q", msg)
	}

	// A foreign currency is not overwritten as if it were the vehicle's
	rate := 1.1
	foreign := &models.DriveExpense{VehicleID: f.mine.ID, DriveID: &second.ID, Type: "TOLL", Amount: 500, Currency: "USD", FxRate: &rate, Date: second.StartTime}
	if f.mine.Currency == "USD" {
		foreign.Currency = "CHF"
	}
	if err := f.repo.SaveDriveExpense(ctx, foreign, nil, ""); err != nil {
		t.Fatal(err)
	}
	if msg := call(map[string]any{"drive_id": second.ID, "tolls": "5.00"}); !strings.Contains(msg, "another currency") {
		t.Errorf("a foreign currency: %q", msg)
	}

	for name, item := range map[string]map[string]any{
		"not a drive id":    {"drive_id": "nope", "tolls": "1"},
		"nothing to change": {"drive_id": first.ID},
		"negative":          {"drive_id": first.ID, "tolls": "-1"},
		"not a number":      {"drive_id": first.ID, "tolls": "a lot"},
		"expected alone":    {"drive_id": first.ID, "expected_tolls": "1"},
		"a stranger":        {"drive_id": f.theirs.ID, "tolls": "1"},
	} {
		if call(item) == "" {
			t.Errorf("%s must be refused", name)
		}
	}
	if call(map[string]any{"drive_id": first.ID, "tolls": "1"}, map[string]any{"drive_id": first.ID, "parking": "1"}) == "" {
		t.Error("a trip given twice must be refused")
	}
}
