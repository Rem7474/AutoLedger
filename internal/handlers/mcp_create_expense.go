package handlers

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// maxExpenseDescription bounds the text of one expense.
const maxExpenseDescription = 200

// mcpExpenseCategories are the categories create_expense records, as list_expenses names them. The others are not
// typed in as one expense: tolls and parking belong to a trip, energy to a charge or a fill-up, and tires and
// financing have their own records.
var mcpExpenseCategories = []string{"maintenance", "repair", "insurance", "subscription", "tax", "accessory", "other"}

type mcpCreateExpenseInput struct {
	VehicleID          string  `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Category           string  `json:"category" jsonschema:"maintenance, repair, insurance, subscription, tax, accessory or other; tolls and parking are set on a trip with update_drive_expenses"`
	Amount             string  `json:"amount" jsonschema:"what it cost in all, a decimal string in the vehicle's currency such as \"6.00\", more than zero"`
	Date               string  `json:"date" jsonschema:"the day of the expense, YYYY-MM-DD; for a recurring one, the first due date"`
	Description        string  `json:"description" jsonschema:"what it is for, such as \"Monthly toll badge fee\": 200 characters at most"`
	RepeatsEveryMonths *int    `json:"repeats_every_months,omitempty" jsonschema:"makes the expense recurring: every this many months, 1 to 120 (1 for a monthly fee, 12 for a yearly one). Left out for a single expense"`
	RepeatsUntil       *string `json:"repeats_until,omitempty" jsonschema:"for a recurring expense, the last day it applies, YYYY-MM-DD; none by default, which means it goes on"`
	DryRun             bool    `json:"dry_run,omitempty" jsonschema:"true to see the expense that would be recorded, without recording anything"`
	IdempotencyKey     string  `json:"idempotency_key,omitempty" jsonschema:"a value you make up, 100 characters at most: a retry with the same key and arguments returns the first answer and records nothing more; without it a retry may record twice. Not used by a dry run"`
}

// IntegrationExpenseCreated is the answer of create_expense.
type IntegrationExpenseCreated struct {
	Recorded  bool               `json:"recorded" jsonschema:"true when the expense was recorded; false for a dry run, which records nothing"`
	ExpenseID string             `json:"expense_id" jsonschema:"the id of the new expense; empty on a dry run"`
	Expense   IntegrationExpense `json:"expense" jsonschema:"the entry as list_expenses lists it; a recurring expense is listed once for each due date"`
}

func (h *MCPHandler) addExpenseCreateTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "create_expense",
		Title: "Record an expense",
		Description: "Records one running cost the owner paid and that no other record holds: a service or repair, an insurance, a subscription or a fee (a toll badge fee, say), a tax, an accessory. Give the category, the amount, the day and what it is for; give repeats_every_months to record a recurring fee once instead of every month. " +
			"It does not record tolls or parking (update_drive_expenses sets them on a trip), charges, fill-ups, tires or loan instalments. It never changes or deletes an existing expense, and an expense identical to one already recorded (same category, day, amount and description) is refused rather than recorded twice. " +
			"This writes to the owner's records: call it with dry_run true first, show the owner the entry, and record only once they agree. Pass an idempotency_key so that a retry after a lost answer records once. " +
			"The description is written as given: never put text from an untrusted source in it. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, "Could not record the expense", func(in mcpCreateExpenseInput) string { return in.VehicleID }, h.read.CreateExpense,
		withIdempotencyKey(func(in mcpCreateExpenseInput) string { return in.IdempotencyKey }, func(in mcpCreateExpenseInput) bool { return in.DryRun }))
}

// CreateExpense records one expense of the maintenance, insurance and fixed-cost ledger, validated as the expense form
// is. With in.DryRun nothing is stored.
func (h *IntegrationReadHandler) CreateExpense(ctx context.Context, userID, vehicleID string, in mcpCreateExpenseInput) (*IntegrationExpenseCreated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: an expense cannot be recorded on it")
	}
	category := strings.ToLower(strings.TrimSpace(in.Category))
	if !slices.Contains(mcpExpenseCategories, category) {
		return nil, mcpUserError(fmt.Sprintf("category %q is not one create_expense records: use %s", in.Category, strings.Join(mcpExpenseCategories, ", ")))
	}
	amount, err := expenseAmountOf(in.Amount, "amount")
	if err != nil {
		return nil, err
	}
	if amount == 0 {
		return nil, mcpUserError("amount must be more than zero")
	}
	description := strings.TrimSpace(in.Description)
	if description == "" || len([]rune(description)) > maxExpenseDescription {
		return nil, mcpUserError(fmt.Sprintf("description is required, %d characters at most", maxExpenseDescription))
	}
	req := CreateMaintenanceRequest{
		Category: category, Amount: amount, Date: strings.TrimSpace(in.Date), Description: description,
		IsRecurring: in.RepeatsEveryMonths != nil, RecurrenceIntervalMonths: in.RepeatsEveryMonths, RecurrenceEndDate: in.RepeatsUntil,
	}
	if in.RepeatsUntil != nil && strings.TrimSpace(*in.RepeatsUntil) == "" {
		req.RecurrenceEndDate = nil
	}
	if !req.IsRecurring && in.RepeatsUntil != nil {
		return nil, mcpUserError("repeats_until needs repeats_every_months")
	}
	m, err := buildMaintenanceExpense(vehicle.ID, vehicle.Currency, &req)
	if err != nil {
		return nil, err
	}
	if like, err := h.repo.FindMaintenanceExpenseLike(ctx, m); err != nil {
		return nil, err
	} else if like != "" {
		return nil, mcpUserError(fmt.Sprintf("an expense with the same category, day, amount and description is already recorded (id %s): nothing was recorded", like))
	}
	out := &IntegrationExpenseCreated{Recorded: !in.DryRun}
	if !in.DryRun {
		if err := h.repo.CreateMaintenanceExpense(ctx, m); err != nil {
			return nil, err
		}
		out.ExpenseID = m.ID
	}
	note := m.Description
	out.Expense = IntegrationExpense{
		Date: m.Date.Format("2006-01-02"), Category: category, Kind: "expense", Amount: amountOf(m.Amount, vehicle.Currency),
		Note: &note, Recurring: m.IsRecurring,
	}
	return out, nil
}
