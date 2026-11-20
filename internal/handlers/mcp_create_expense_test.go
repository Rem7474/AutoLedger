package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
)

func TestMCPCreateExpenseRecordsOneExpense(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(token string, args map[string]any) (*mcp.CallToolResult, IntegrationExpenseCreated) {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: "create_expense", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		var out IntegrationExpenseCreated
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
		list, err := f.repo.ListMaintenanceExpenses(ctx, f.mine.ID)
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}
	args := map[string]any{
		"vehicle_id": f.mine.ID, "category": "Subscription", "amount": "6.00", "date": "2026-09-01",
		"description": " Monthly toll badge fee ", "repeats_every_months": 1,
	}
	start := count()

	if res, _ := call(f.readTok, args); !res.IsError || !strings.Contains(reason(res), "can only read") {
		t.Fatalf("a read connection must not record: %+v", res)
	}

	args["dry_run"] = true
	res, preview := call(writeTok, args)
	if res.IsError || preview.Recorded || preview.ExpenseID != "" || preview.Expense.Amount.Amount != "6.00" || !preview.Expense.Recurring || preview.Expense.Category != "subscription" {
		t.Fatalf("a dry run shows the entry and records nothing: %+v %+v", res, preview)
	}
	if count() != start {
		t.Fatal("a dry run recorded an expense")
	}

	delete(args, "dry_run")
	res, out := call(writeTok, args)
	if res.IsError || !out.Recorded || out.ExpenseID == "" {
		t.Fatalf("a write connection records: %+v", res)
	}
	stored, err := f.repo.GetMaintenanceExpense(ctx, f.mine.ID, out.ExpenseID)
	if err != nil || stored.Category != "SUBSCRIPTION" || stored.Amount != 600 || stored.Description != "Monthly toll badge fee" || !stored.IsRecurring || stored.RecurrenceIntervalMonths == nil || *stored.RecurrenceIntervalMonths != 1 || stored.Currency != f.mine.Currency {
		t.Fatalf("stored expense: %+v %v", stored, err)
	}

	// The same call again records nothing a second time
	if res, _ := call(writeTok, args); !res.IsError || !strings.Contains(reason(res), "already recorded") || !strings.Contains(reason(res), out.ExpenseID) {
		t.Fatalf("a repeated call is refused: %+v", res)
	}
	if count() != start+1 {
		t.Fatalf("expenses: %d", count())
	}
}

func TestMCPCreateExpenseRefusesWhatItCannotRecord(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	base := func() map[string]any {
		return map[string]any{"vehicle_id": f.mine.ID, "category": "tax", "amount": "40", "date": "2026-02-01", "description": "Road tax"}
	}
	with := func(key string, value any) map[string]any {
		m := base()
		if value == nil {
			delete(m, key)
		} else {
			m[key] = value
		}
		return m
	}
	cases := map[string]struct {
		args map[string]any
		want string
	}{
		"a toll":                {with("category", "toll"), "not one create_expense records"},
		"financing":             {with("category", "financing"), "not one create_expense records"},
		"zero":                  {with("amount", "0"), "more than zero"},
		"negative":              {with("amount", "-3"), "positive decimal string"},
		"not an amount":         {with("amount", "six"), "positive decimal string"},
		"no description":        {with("description", "  "), "description is required"},
		"long description":      {with("description", strings.Repeat("x", 201)), "200 characters at most"},
		"not a date":            {with("date", "soon"), "Invalid date"},
		"until without repeat":  {with("repeats_until", "2027-01-01"), "needs repeats_every_months"},
		"interval out of range": {with("repeats_every_months", 500), "frequency in months"},
		"ends before it starts": {func() map[string]any {
			m := with("repeats_every_months", 12)
			m["repeats_until"] = "2025-01-01"
			return m
		}(), "before the first due date"},
	}
	for name, tc := range cases {
		res, err := f.connect(t, writeTok).CallTool(ctx, &mcp.CallToolParams{Name: "create_expense", Arguments: tc.args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !res.IsError || !strings.Contains(res.Content[0].(*mcp.TextContent).Text, tc.want) {
			t.Errorf("%s: %+v, want %q", name, res.Content, tc.want)
		}
	}
	list, err := f.repo.ListMaintenanceExpenses(ctx, f.mine.ID)
	if err != nil || len(list) != 0 {
		t.Errorf("a refused call recorded something: %d %v", len(list), err)
	}
}
