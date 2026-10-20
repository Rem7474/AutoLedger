package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpEnergyInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
}

func (h *MCPHandler) addEnergyTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_energy_efficiency",
		Title: "Energy efficiency",
		Description: "How a vehicle uses and pays for its energy. Electric: consumption per 100 km, price per kWh, energy cost per 100 km, charging efficiency, cost of a full charge, " +
			"the mix of slow, AC and DC charging with the price of each, the effect of cold weather, the usable capacity and the battery state of health. " +
			"Fuel: litres per 100 km, price per litre, fuel cost per 100 km. The basis field says which electric figures are measured and which are derived.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpEnergyInput) (*mcp.CallToolResult, IntegrationEnergyEfficiency, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationEnergyEfficiency{}, mcpFailure(ctx, "get_energy_efficiency", err, "Could not compute the energy efficiency")
		}
		out, err := h.read.EnergyEfficiency(ctx, mcpUserID(req), vehicleID)
		if err != nil {
			return nil, IntegrationEnergyEfficiency{}, mcpFailure(ctx, "get_energy_efficiency", err, "Could not compute the energy efficiency")
		}
		return nil, *out, nil
	})
}
