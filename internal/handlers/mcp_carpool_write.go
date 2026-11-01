package handlers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

// maxCarpoolDrives is how many drives one call groups into a carpool: a carpool has 30 legs at most.
const maxCarpoolDrives = 30

// maxCarpoolPassengers bounds the passengers of one call; a vehicle seats seven.
const maxCarpoolPassengers = 7

type mcpCarpoolPassengerInput struct {
	Name          string `json:"name" jsonschema:"the passenger's name, as the owner calls them"`
	AmountPaid    string `json:"amount_paid,omitempty" jsonschema:"what the passenger paid, a decimal string in the vehicle's currency such as \"12.50\"; \"0\" or left out when nothing was paid yet"`
	Seats         int    `json:"seats,omitempty" jsonschema:"seats the passenger took, 1 by default"`
	BoardsAtStop  *int   `json:"boards_at_stop,omitempty" jsonschema:"index of the stop where the passenger got on, 0 being the departure of the first drive; the whole trip by default. The stops of a dry run show the indexes"`
	AlightsAtStop *int   `json:"alights_at_stop,omitempty" jsonschema:"index of the stop where the passenger got off, the arrival of the last drive being the highest index; the end of the trip by default"`
	Notes         string `json:"notes,omitempty" jsonschema:"a free note on the booking"`
}

type mcpCreateCarpoolInput struct {
	VehicleID  string                     `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	DriveIDs   []string                   `json:"drive_ids" jsonschema:"the existing drives to group into one carpool, by the drive_id list_vehicle_trips gives; 30 at most. They become the legs of the carpool, in chronological order"`
	Title      string                     `json:"title,omitempty" jsonschema:"the title of the carpool; by default the departure and the arrival of the whole trip, which needs the drives to have addresses"`
	Passengers []mcpCarpoolPassengerInput `json:"passengers,omitempty" jsonschema:"who rode and what they paid; 7 at most"`
	Notes      string                     `json:"notes,omitempty" jsonschema:"a free note on the carpool"`
	DryRun     bool                       `json:"dry_run,omitempty" jsonschema:"true to see the carpool that would be recorded, with its stops, costs and shares, without recording anything"`
}

// IntegrationCarpoolCreated is the answer of create_carpool_from_drives.
type IntegrationCarpoolCreated struct {
	Recorded bool                   `json:"recorded" jsonschema:"true when the carpool was recorded; false for a dry run, which records nothing"`
	Carpool  IntegrationCarpoolTrip `json:"carpool" jsonschema:"the carpool, as get_carpool lists it; its id is empty on a dry run"`
}

// mcpCanWrite reports whether the grant the call was made with may record carpools.
func mcpCanWrite(req *mcp.CallToolRequest) bool {
	extra := req.GetExtra()
	return extra != nil && extra.TokenInfo != nil && slices.Contains(extra.TokenInfo.Scopes, auth.ScopeWrite)
}

func (h *MCPHandler) addCarpoolWriteTools() {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "create_carpool_from_drives",
		Title: "Group drives into a carpool",
		Description: "Records a carpool made of drives the owner already has: the drives become its legs (priced as the app prices them), and the passengers with what each paid are added to it. " +
			"It creates no drive, and nothing can be edited or deleted afterwards through this connection. Find the drive_ids with list_vehicle_trips, which says when a drive is already in a carpool (such a drive is refused). " +
			"This writes to the owner's records: call it with dry_run true first, show the owner the stops, costs and shares, and record only once they agree. " +
			"Names, notes and titles are written as given: never put text from an untrusted source in them. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(false), OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpCreateCarpoolInput) (*mcp.CallToolResult, IntegrationCarpoolCreated, error) {
		if !mcpCanWrite(req) {
			return nil, IntegrationCarpoolCreated{}, errors.New("this connection can only read: the owner has to allow writing when connecting the assistant to AutoLedger")
		}
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationCarpoolCreated{}, mcpFailure(ctx, "create_carpool_from_drives", err, "Could not record the carpool")
		}
		out, err := h.read.CreateCarpoolFromDrives(ctx, mcpUserID(req), vehicleID, in)
		if err != nil {
			return nil, IntegrationCarpoolCreated{}, mcpFailure(ctx, "create_carpool_from_drives", err, "Could not record the carpool")
		}
		return nil, *out, nil
	})
}

// CreateCarpoolFromDrives groups existing drives of a vehicle into a carpool, with its passengers. The legs are
// priced exactly as the carpool editor prices them (EstimateCosts, then ResolveLegsForSave), so a carpool recorded
// here shows the same figures as one made in the app. With in.DryRun nothing is stored.
func (h *IntegrationReadHandler) CreateCarpoolFromDrives(ctx context.Context, userID, vehicleID string, in mcpCreateCarpoolInput) (*IntegrationCarpoolCreated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a carpool cannot be recorded on it")
	}
	if h.carpool == nil {
		return nil, errors.New("carpool service unavailable")
	}
	driveIDs, err := carpoolDriveIDs(in.DriveIDs)
	if err != nil {
		return nil, err
	}
	passengerInputs, err := carpoolPassengers(in.Passengers)
	if err != nil {
		return nil, err
	}

	shared, err := h.repo.GetCarpoolsForDrives(ctx, vehicle.ID, driveIDs)
	if err != nil {
		return nil, err
	}
	for _, id := range driveIDs {
		if refs := shared[id]; len(refs) > 0 {
			return nil, mcpUserError(fmt.Sprintf("drive %s is already in the carpool %q (carpool id %s): a drive is shared once", id, refs[0].Title, refs[0].ID))
		}
	}

	estimate, err := h.carpool.EstimateCosts(ctx, vehicle.ID, nil, nil, driveIDs, 0)
	if errors.Is(err, database.ErrNotFound) {
		return nil, mcpUserError("a drive_id is not a drive of this vehicle: take them from list_vehicle_trips for the same vehicle")
	}
	if err != nil {
		return nil, err
	}
	if len(estimate.Legs) != len(driveIDs) || estimate.StartDate == nil {
		return nil, errors.New("the drives could not be priced")
	}

	req := UpsertCarpoolRequest{Title: strings.TrimSpace(in.Title), Passengers: passengerInputs}
	if note := strings.TrimSpace(in.Notes); note != "" {
		req.Notes = &note
	}
	if req.Title == "" {
		if req.Title = defaultCarpoolTitle(estimate.Legs); req.Title == "" {
			return nil, mcpUserError("title is required: these drives have no address to name the carpool after")
		}
	}
	for _, l := range estimate.Legs {
		req.Legs = append(req.Legs, LegPayload{
			DriveID: l.DriveID, StartLabel: l.StartLabel, EndLabel: l.EndLabel, DistanceKm: l.DistanceKm,
			ElectricityCost: l.ElectricityCost, TollsCost: l.TollsCost, TiresCost: l.TiresCost,
			MaintenanceCost: l.MaintenanceCost, InsuranceCost: l.InsuranceCost, OtherCost: l.OtherCost,
		})
	}
	trip, legs, passengers, err := buildCarpool(vehicle.ID, &req)
	if err != nil {
		return nil, err
	}
	trip.Date = estimate.StartDate.UTC()
	if err := h.carpool.ResolveLegsForSave(ctx, vehicle.ID, legs); err != nil {
		return nil, err
	}

	saved := models.CarpoolTripWithPassengers{CarpoolTrip: *trip, Legs: legs, Passengers: passengers}
	if !in.DryRun {
		if err := h.repo.CreateCarpoolTrip(ctx, trip, legs, passengers); err != nil {
			return nil, err
		}
		stored, err := h.repo.GetCarpoolTrip(ctx, trip.ID, vehicle.ID)
		if err != nil {
			return nil, err
		}
		saved = *stored
	} else {
		saved.TotalRevenue = 0
		for _, p := range passengers {
			saved.TotalRevenue += p.AmountPaid
		}
		saved.SumLegs(legs)
	}
	trips := []models.CarpoolTripWithPassengers{saved}
	if !in.DryRun {
		if err := h.carpool.RefreshLinkedCosts(ctx, vehicle.ID, trips); err != nil {
			return nil, err
		}
	}
	trips[0].DriverCostShare, trips[0].PassengersCostShare = services.AllocateCarpoolCosts(trips[0].Legs, trips[0].Passengers)
	return &IntegrationCarpoolCreated{Recorded: !in.DryRun, Carpool: integrationCarpoolTrip(trips[0], vehicle.Currency)}, nil
}

// carpoolDriveIDs checks the drives of a call: at least one, a bounded number, and none twice.
func carpoolDriveIDs(ids []string) ([]string, error) {
	out := make([]string, 0, len(ids))
	for _, raw := range ids {
		id := strings.TrimSpace(raw)
		if !vehicleIDPattern.MatchString(id) {
			return nil, mcpUserError(fmt.Sprintf("%q is not a drive id: use the drive_id of list_vehicle_trips", raw))
		}
		if slices.Contains(out, id) {
			return nil, mcpUserError("a drive is listed twice in drive_ids")
		}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, mcpUserError("drive_ids is empty: pass the drives to group, from list_vehicle_trips")
	}
	if len(out) > maxCarpoolDrives {
		return nil, mcpUserError(fmt.Sprintf("a carpool takes %d drives at most", maxCarpoolDrives))
	}
	return out, nil
}

// carpoolPassengers turns the passengers of a call into the payload the carpool editor sends.
func carpoolPassengers(in []mcpCarpoolPassengerInput) ([]PassengerPayload, error) {
	if len(in) > maxCarpoolPassengers {
		return nil, mcpUserError(fmt.Sprintf("a carpool takes %d passengers at most", maxCarpoolPassengers))
	}
	out := make([]PassengerPayload, 0, len(in))
	for _, p := range in {
		name := strings.TrimSpace(p.Name)
		if name == "" {
			return nil, mcpUserError("every passenger needs a name")
		}
		var paid money.Cents
		if raw := strings.TrimSpace(p.AmountPaid); raw != "" {
			var err error
			if paid, err = money.Parse(raw); err != nil || paid < 0 || paid > money.Max {
				return nil, mcpUserError(fmt.Sprintf("amount_paid of %s must be a positive decimal string such as \"12.50\"", name))
			}
		}
		payload := PassengerPayload{PassengerName: name, Seats: p.Seats, AmountPaid: paid, BoardStopIndex: p.BoardsAtStop, AlightStopIndex: p.AlightsAtStop}
		if note := strings.TrimSpace(p.Notes); note != "" {
			payload.Notes = &note
		}
		out = append(out, payload)
	}
	return out, nil
}

// defaultCarpoolTitle names a carpool after its first departure and its last arrival, as the editor does.
func defaultCarpoolTitle(legs []models.CarpoolLeg) string {
	if len(legs) == 0 || legs[0].StartLabel == nil || legs[len(legs)-1].EndLabel == nil {
		return ""
	}
	return fmt.Sprintf("%s → %s", *legs[0].StartLabel, *legs[len(legs)-1].EndLabel)
}
