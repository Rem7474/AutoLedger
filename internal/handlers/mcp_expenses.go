package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpExpensesInput struct {
	VehicleID  string   `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	From       string   `json:"from,omitempty" jsonschema:"first day to include, YYYY-MM-DD; no lower bound when omitted"`
	To         string   `json:"to,omitempty" jsonschema:"last day to include, YYYY-MM-DD; no upper bound when omitted"`
	Categories []string `json:"categories,omitempty" jsonschema:"keep only these categories: energy, toll, parking, travel_other, maintenance, repair, insurance, subscription, tax, financing, tires, accessory, other; every running cost when omitted"`
	MinAmount  string   `json:"min_amount,omitempty" jsonschema:"keep only the entries of at least this amount, a decimal string such as 50.00, in the vehicle's currency"`
	Sort       string   `json:"sort,omitempty" jsonschema:"date (default) or amount"`
	Ascending  bool     `json:"ascending,omitempty" jsonschema:"oldest or smallest first instead of newest or largest first"`
	Limit      int      `json:"limit,omitempty" jsonschema:"entries per page, 50 by default, 200 at most"`
	Cursor     string   `json:"cursor,omitempty" jsonschema:"the next_cursor of the previous page"`
}

func (h *MCPHandler) addExpenseTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "list_expenses",
		Title: "List expenses",
		Description: "The individual entries behind a vehicle's costs, one per charge, fuel fill-up, toll, maintenance or insurance expense, tire purchase or loan or lease instalment, " +
			"with their date, category, amount and, for a charge, the place and the energy. Filter by period, category and minimum amount, sort by date or amount. " +
			"The total and the per-category sums cover every matching entry, not only the page, so never add up the page yourself. The purchase of the vehicle is not an entry. " +
			"Use it to answer which expense was the largest, what was spent on a category in a period or where the vehicle was charged.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpExpensesInput) (*mcp.CallToolResult, IntegrationExpenses, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationExpenses{}, mcpFailure(ctx, "list_expenses", err, "Could not list the expenses")
		}
		out, err := h.read.Expenses(ctx, mcpUserID(req), vehicleID, IntegrationExpenseQuery{
			From: in.From, To: in.To, Categories: in.Categories, MinAmount: in.MinAmount, Sort: in.Sort, Ascending: in.Ascending, Limit: in.Limit, Cursor: in.Cursor,
		})
		if err != nil {
			return nil, IntegrationExpenses{}, mcpFailure(ctx, "list_expenses", err, "Could not list the expenses")
		}
		return nil, *out, nil
	})
}
