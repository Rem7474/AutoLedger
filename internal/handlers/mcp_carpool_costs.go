package handlers

import (
	"context"
	"errors"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpCarpoolCostsInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	CarpoolID string `json:"carpool_id" jsonschema:"the id of the carpool trip, as get_carpool gives it"`
	Action    string `json:"action" jsonschema:"freeze to keep the carpool's amounts as they are, unfreeze to let them follow the cost of their drives again, recalculate to work them out again from the drives now"`
}

func (h *MCPHandler) addCarpoolCostTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "carpool_costs",
		Title: "Freeze, unfreeze or recalculate a carpool's amounts",
		Description: "Controls how the cost of one carpool is kept, as the carpool page does. freeze keeps its amounts as they are, so that later changes to its drives or to the tariffs do not move them; unfreeze lets the legs made from drives follow the cost of their drives again; recalculate works the amounts out again from the drives as they are now, which overwrites the frozen or stored ones. " +
			"Recalculate has no preview: tell the owner which carpool is concerned and call it only once they agree. The answer is the carpool as it is after the change. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, "Could not change the carpool's amounts", func(in mcpCarpoolCostsInput) string { return in.VehicleID }, h.read.CarpoolCosts)
}

// CarpoolCosts freezes, unfreezes or recalculates the amounts of one carpool.
func (h *IntegrationReadHandler) CarpoolCosts(ctx context.Context, userID, vehicleID string, in mcpCarpoolCostsInput) (*IntegrationCarpoolUpdated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a carpool cannot be changed on it")
	}
	if h.carpool == nil {
		return nil, errors.New("carpool service unavailable")
	}
	carpoolID := strings.TrimSpace(in.CarpoolID)
	if !vehicleIDPattern.MatchString(carpoolID) {
		return nil, mcpUserError("carpool_id is not a carpool id: use the id of get_carpool")
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	if action != "freeze" && action != "unfreeze" && action != "recalculate" {
		return nil, mcpUserError("action must be freeze, unfreeze or recalculate")
	}
	if _, err := h.repo.GetCarpoolTrip(ctx, carpoolID, vehicle.ID); errors.Is(err, database.ErrNotFound) {
		return nil, mcpUserError("carpool not found on this vehicle: use the id of get_carpool for the same vehicle")
	} else if err != nil {
		return nil, err
	}
	switch action {
	case "freeze", "unfreeze":
		err = h.repo.SetCarpoolCostsFrozen(ctx, carpoolID, vehicle.ID, action == "freeze")
	default:
		_, err = h.carpool.RecalculateTrip(ctx, vehicle.ID, carpoolID)
	}
	if err != nil {
		return nil, err
	}
	trip, err := h.repo.GetCarpoolTrip(ctx, carpoolID, vehicle.ID)
	if err != nil {
		return nil, err
	}
	list := []models.CarpoolTripWithPassengers{*trip}
	if err := h.carpool.RefreshLinkedCosts(ctx, vehicle.ID, list); err != nil {
		return nil, err
	}
	return &IntegrationCarpoolUpdated{Saved: true, Carpool: integrationCarpoolTrip(list[0], vehicle.Currency)}, nil
}
