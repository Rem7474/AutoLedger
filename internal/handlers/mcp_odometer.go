package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpOdometerInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Date      string `json:"date,omitempty" jsonschema:"a day, YYYY-MM-DD, to get the odometer on that day from the records (a reading, or spread between the readings around it)"`
}

func (h *MCPHandler) addOdometerTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_odometer",
		Title: "Odometer",
		Description: "The current odometer of a vehicle, how many dated readings the records hold and the period they cover. " +
			"With a date it also gives the odometer on that day, and says whether it is a reading, spread between two readings or the nearest one.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpOdometerInput) (*mcp.CallToolResult, IntegrationOdometer, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationOdometer{}, mcpFailure(ctx, "get_odometer", err, "Could not read the odometer")
		}
		out, err := h.read.Odometer(ctx, mcpUserID(req), vehicleID, in.Date)
		if err != nil {
			return nil, IntegrationOdometer{}, mcpFailure(ctx, "get_odometer", err, "Could not read the odometer")
		}
		return nil, *out, nil
	})
}
