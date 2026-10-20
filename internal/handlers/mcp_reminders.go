package handlers

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpRemindersInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Status    string `json:"status,omitempty" jsonschema:"keep only the reminders in this state: overdue, due_soon or ok; all of them when omitted"`
}

func (h *MCPHandler) addReminderTools(readOnly *mcp.ToolAnnotations) {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "get_reminders",
		Title: "Maintenance reminders",
		Description: "The maintenance reminders of a vehicle (service, tires, inspection...) and where each stands against the odometer and the calendar: " +
			"overdue, due soon or ok, with the distance and days left, the due date and the last service. Most urgent first. " +
			"Use it to answer what is due on a vehicle or when a service is next needed.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpRemindersInput) (*mcp.CallToolResult, IntegrationReminders, error) {
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationReminders{}, mcpFailure(ctx, "get_reminders", err, "Could not list the reminders")
		}
		out, err := h.read.Reminders(ctx, mcpUserID(req), vehicleID, in.Status)
		if err != nil {
			return nil, IntegrationReminders{}, mcpFailure(ctx, "get_reminders", err, "Could not list the reminders")
		}
		return nil, *out, nil
	})
}
