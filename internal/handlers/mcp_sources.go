package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (h *MCPHandler) addSourceTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_data_sources",
		Title: "Data sources",
		Description: "Where a vehicle's records come from and how fresh they are: what each origin wrote (TeslaMate, Home Assistant or a script, a CSV import, manual entries) with the date of its latest record, " +
			"the state of the TeslaMate synchronization and how many pushed charging sessions still wait for a vehicle. Use it when asked why recent data is missing.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpVehicleInput) (*mcp.CallToolResult, IntegrationDataSources, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationDataSources{}, mcpFailure(ctx, "get_data_sources", err, "Could not read the data sources")
		}
		out, err := h.read.DataSources(ctx, mcpUserID(req), vehicleID)
		if err != nil {
			return nil, IntegrationDataSources{}, mcpFailure(ctx, "get_data_sources", err, "Could not read the data sources")
		}
		return nil, *out, nil
	})
}
