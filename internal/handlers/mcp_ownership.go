package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpOwnershipInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
}

func (h *MCPHandler) addOwnershipTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_vehicle_ownership",
		Title: "Vehicle ownership cost",
		Description: "What owning a vehicle really costs once the purchase counts, which get_vehicle_costs leaves out: the full cost and its cost per kilometre (running costs with the tires amortized, plus the depreciation), " +
			"the lease or loan with its mileage allowance used and the projected excess cost, and the estimated resale value today. " +
			"Check depreciation_known before comparing the full cost of two vehicles: when false, the depreciation is missing and the full cost is understated.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpOwnershipInput) (*mcp.CallToolResult, IntegrationOwnership, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationOwnership{}, mcpFailure(ctx, "get_vehicle_ownership", err, "Could not compute the ownership figures")
		}
		out, err := h.read.Ownership(ctx, mcpUserID(req), vehicleID)
		if err != nil {
			return nil, IntegrationOwnership{}, mcpFailure(ctx, "get_vehicle_ownership", err, "Could not compute the ownership figures")
		}
		return nil, *out, nil
	})
}
