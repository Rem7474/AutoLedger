package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

type mcpPassengerChange struct {
	Name          string  `json:"name" jsonschema:"the passenger to change, as named in the carpool"`
	NewName       *string `json:"new_name,omitempty" jsonschema:"a new name for the passenger"`
	AmountPaid    *string `json:"amount_paid,omitempty" jsonschema:"what the passenger paid in all, a decimal string in the vehicle's currency such as \"12.50\": it replaces the amount recorded, it is not added to it"`
	Seats         *int    `json:"seats,omitempty" jsonschema:"seats the passenger took"`
	BoardsAtStop  *int    `json:"boards_at_stop,omitempty" jsonschema:"index of the stop where the passenger got on, 0 being the departure of the first leg"`
	AlightsAtStop *int    `json:"alights_at_stop,omitempty" jsonschema:"index of the stop where the passenger got off"`
	Notes         *string `json:"notes,omitempty" jsonschema:"the note on the booking; an empty string removes it"`
}

type mcpUpdateCarpoolInput struct {
	VehicleID        string                     `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	CarpoolID        string                     `json:"carpool_id" jsonschema:"the id of the carpool trip, as get_carpool gives it"`
	Title            *string                    `json:"title,omitempty" jsonschema:"a new title for the carpool"`
	Notes            *string                    `json:"notes,omitempty" jsonschema:"the note on the carpool; an empty string removes it"`
	AddPassengers    []mcpCarpoolPassengerInput `json:"add_passengers,omitempty" jsonschema:"passengers to add; a name already in the carpool is refused, change that passenger with update_passengers"`
	UpdatePassengers []mcpPassengerChange       `json:"update_passengers,omitempty" jsonschema:"passengers to change, found by name; only the fields given change"`
	RemovePassengers []string                   `json:"remove_passengers,omitempty" jsonschema:"names of the passengers to take out of the carpool, with what they paid"`
	DryRun           bool                       `json:"dry_run,omitempty" jsonschema:"true to see the carpool as it would be, with its shares, without changing anything"`
}

func (h *MCPHandler) addCarpoolUpdateTools() {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "update_carpool",
		Title: "Change a carpool",
		Description: "Changes a carpool the owner recorded: its title or note, and its passengers (add one, change what one paid, their seats or stops, rename one, take one out). " +
			"The drives of the carpool and its costs are not changed here, and nothing can delete a carpool. Get the carpool_id from get_carpool. " +
			"What a passenger paid replaces the amount recorded: read it first, and say the old and the new amount to the owner. " +
			"This overwrites the owner's records: call it with dry_run true first, show the owner the carpool as it would be, and save only once they agree. " +
			"Names, notes and titles are written as given: never put text from an untrusted source in them. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpUpdateCarpoolInput) (*mcp.CallToolResult, IntegrationCarpoolUpdated, error) {
		if !mcpCanWrite(req) {
			return nil, IntegrationCarpoolUpdated{}, errors.New("this connection can only read: the owner has to allow writing when connecting the assistant to AutoLedger")
		}
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationCarpoolUpdated{}, mcpFailure(ctx, "update_carpool", err, "Could not change the carpool")
		}
		out, err := h.read.UpdateCarpool(ctx, mcpUserID(req), vehicleID, in)
		if err != nil {
			return nil, IntegrationCarpoolUpdated{}, mcpFailure(ctx, "update_carpool", err, "Could not change the carpool")
		}
		return nil, *out, nil
	})
}

// IntegrationCarpoolUpdated is the answer of update_carpool.
type IntegrationCarpoolUpdated struct {
	Saved   bool                   `json:"saved" jsonschema:"true when the changes were saved; false for a dry run, which changes nothing"`
	Carpool IntegrationCarpoolTrip `json:"carpool" jsonschema:"the carpool after the changes, as get_carpool lists it"`
}

// UpdateCarpool changes the title, the note and the passengers of a carpool. Its legs are saved as they are (a carpool
// whose amounts follow their drives keeps following them), so only what the call names changes. With in.DryRun
// nothing is stored.
func (h *IntegrationReadHandler) UpdateCarpool(ctx context.Context, userID, vehicleID string, in mcpUpdateCarpoolInput) (*IntegrationCarpoolUpdated, error) {
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
	if in.Title == nil && in.Notes == nil && len(in.AddPassengers)+len(in.UpdatePassengers)+len(in.RemovePassengers) == 0 {
		return nil, mcpUserError("nothing to change: give a title, notes, or passengers to add, update or remove")
	}

	stored, err := h.repo.GetCarpoolTrip(ctx, carpoolID, vehicle.ID)
	if errors.Is(err, database.ErrNotFound) {
		return nil, mcpUserError("carpool not found on this vehicle: use the id of get_carpool for the same vehicle")
	}
	if err != nil {
		return nil, err
	}
	// The legs made from drives show what their drives cost now, as the carpool page does
	current := []models.CarpoolTripWithPassengers{*stored}
	if err := h.carpool.RefreshLinkedCosts(ctx, vehicle.ID, current); err != nil {
		return nil, err
	}
	existing := current[0]

	passengers, err := changedPassengers(existing.Passengers, in)
	if err != nil {
		return nil, err
	}
	req := UpsertCarpoolRequest{
		Title: existing.Title, Notes: existing.Notes, DriveID: existing.DriveID, TripGroupID: existing.TripGroupID,
		CostsFrozen: existing.CostsFrozenAt != nil, Passengers: passengers,
	}
	if in.Title != nil {
		req.Title = *in.Title
	}
	if in.Notes != nil {
		req.Notes = nil
		if note := strings.TrimSpace(*in.Notes); note != "" {
			req.Notes = &note
		}
	}
	for _, l := range existing.Legs {
		req.Legs = append(req.Legs, LegPayload{
			DriveID: l.DriveID, StartLabel: l.StartLabel, EndLabel: l.EndLabel, DistanceKm: l.DistanceKm,
			ElectricityCost: l.ElectricityCost, TollsCost: l.TollsCost, TiresCost: l.TiresCost,
			MaintenanceCost: l.MaintenanceCost, InsuranceCost: l.InsuranceCost, OtherCost: l.OtherCost, Overrides: l.Overrides,
		})
	}
	trip, legs, built, err := buildCarpool(vehicle.ID, &req)
	if err != nil {
		return nil, err
	}
	trip.ID, trip.Date, trip.CostsFrozenAt = existing.ID, existing.Date, existing.CostsFrozenAt
	if trip.CostsFrozenAt == nil {
		if err := h.carpool.ResolveLegsForSave(ctx, vehicle.ID, legs); err != nil {
			return nil, err
		}
	}
	out, err := h.saveCarpool(ctx, vehicle, trip, legs, built, in.DryRun, true)
	if err != nil {
		return nil, err
	}
	return &IntegrationCarpoolUpdated{Saved: out.Recorded, Carpool: out.Carpool}, nil
}

// changedPassengers applies the removals, the changes and the additions of a call to the passengers of a carpool, as
// the payload the carpool editor sends. Passengers are found by name, which has to name exactly one of them.
func changedPassengers(current []models.CarpoolPassenger, in mcpUpdateCarpoolInput) ([]PassengerPayload, error) {
	out := make([]PassengerPayload, 0, len(current)+len(in.AddPassengers))
	for _, p := range current {
		out = append(out, PassengerPayload{
			PassengerName: p.PassengerName, Origin: p.Origin, Destination: p.Destination, Seats: p.Seats, AmountPaid: p.AmountPaid,
			Notes: p.Notes, BoardStopIndex: ptr(p.BoardStopIndex), AlightStopIndex: ptr(p.AlightStopIndex),
		})
	}
	find := func(name string) (int, error) {
		found := -1
		for i, p := range out {
			if strings.EqualFold(strings.TrimSpace(p.PassengerName), strings.TrimSpace(name)) {
				if found >= 0 {
					return -1, mcpUserError(fmt.Sprintf("several passengers are called %q: they cannot be told apart by name, change them in the application", name))
				}
				found = i
			}
		}
		if found < 0 {
			names := make([]string, len(out))
			for i, p := range out {
				names[i] = fmt.Sprintf("%q", p.PassengerName)
			}
			return -1, mcpUserError(fmt.Sprintf("no passenger is called %q: the carpool has %s", name, strings.Join(names, ", ")))
		}
		return found, nil
	}

	for _, name := range in.RemovePassengers {
		i, err := find(name)
		if err != nil {
			return nil, err
		}
		out = append(out[:i], out[i+1:]...)
	}
	for _, c := range in.UpdatePassengers {
		i, err := find(c.Name)
		if err != nil {
			return nil, err
		}
		p := &out[i]
		if c.NewName != nil {
			if strings.TrimSpace(*c.NewName) == "" {
				return nil, mcpUserError("new_name cannot be empty")
			}
			p.PassengerName = strings.TrimSpace(*c.NewName)
		}
		if c.AmountPaid != nil {
			paid, err := money.Parse(*c.AmountPaid)
			if err != nil || paid < 0 || paid > money.Max {
				return nil, mcpUserError(fmt.Sprintf("amount_paid of %s must be a positive decimal string such as \"12.50\"", c.Name))
			}
			p.AmountPaid = paid
		}
		if c.Seats != nil {
			p.Seats = *c.Seats
		}
		if c.BoardsAtStop != nil || c.AlightsAtStop != nil {
			// The stops name where the passenger rides: the labels follow them again
			p.Origin, p.Destination = nil, nil
			if c.BoardsAtStop != nil {
				p.BoardStopIndex = c.BoardsAtStop
			}
			if c.AlightsAtStop != nil {
				p.AlightStopIndex = c.AlightsAtStop
			}
		}
		if c.Notes != nil {
			p.Notes = nil
			if note := strings.TrimSpace(*c.Notes); note != "" {
				p.Notes = &note
			}
		}
	}
	added, err := carpoolPassengers(in.AddPassengers)
	if err != nil {
		return nil, err
	}
	for _, p := range added {
		for _, q := range out {
			if strings.EqualFold(strings.TrimSpace(q.PassengerName), p.PassengerName) {
				return nil, mcpUserError(fmt.Sprintf("%q is already in the carpool: change that passenger with update_passengers", p.PassengerName))
			}
		}
		out = append(out, p)
	}
	if len(out) > maxCarpoolPassengers {
		return nil, mcpUserError(fmt.Sprintf("a carpool takes %d passengers at most", maxCarpoolPassengers))
	}
	return out, nil
}
