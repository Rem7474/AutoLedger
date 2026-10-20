package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpFleetInput struct{}

func (h *MCPHandler) addFleetTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_fleet_summary",
		Title: "Fleet summary",
		Description: "Every vehicle of the account side by side: what each cost this month, its running and full cost per kilometre, its annual cost, its consumption, which ones are cheapest to own among those that can be compared, " +
			"the household's monthly budget and where the month stands against it, the last 12 months of household cost, and who drove how much. " +
			"Amounts are never converted between currencies: totals are given per currency and the monthly series covers the vehicles in the fleet currency only. " +
			"Do not rank a vehicle whose comparable flag is false.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ mcpFleetInput) (*mcp.CallToolResult, IntegrationFleet, error) {
		out, err := h.read.FleetSummary(ctx, mcpUserID(req))
		if err != nil {
			return nil, IntegrationFleet{}, mcpFailure(ctx, "get_fleet_summary", err, "Could not compute the fleet summary")
		}
		return nil, *out, nil
	})
}
