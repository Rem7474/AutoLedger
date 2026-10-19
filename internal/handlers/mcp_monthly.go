package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpMonthlyInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	From      string `json:"from,omitempty" jsonschema:"first month to include, YYYY-MM; without from and to the last 12 months are returned"`
	To        string `json:"to,omitempty" jsonschema:"last month to include, YYYY-MM; the month in progress when omitted"`
}

func (h *MCPHandler) addMonthlyTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_monthly_costs",
		Title: "Monthly costs",
		Description: "What a vehicle cost month by month, oldest first: for each month the cost by category (energy, tolls, maintenance, insurance, financing, tires, other), " +
			"the distance and the cost per kilometre, with the listed months added up. Use it to compare periods or to see which month was expensive and in which category. " +
			"Without from and to it returns the last 12 months. A month in progress is flagged: its figures are not final.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpMonthlyInput) (*mcp.CallToolResult, IntegrationMonthlyCosts, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationMonthlyCosts{}, mcpFailure(ctx, "get_monthly_costs", err, "Could not compute the monthly costs")
		}
		out, err := h.read.MonthlyCosts(ctx, mcpUserID(req), vehicleID, IntegrationMonthlyQuery{From: in.From, To: in.To})
		if err != nil {
			return nil, IntegrationMonthlyCosts{}, mcpFailure(ctx, "get_monthly_costs", err, "Could not compute the monthly costs")
		}
		return nil, *out, nil
	})
}
