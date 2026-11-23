package handlers

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

type mcpManageTripGroupInput struct {
	VehicleID       string    `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Action          string    `json:"action" jsonschema:"create, update or delete"`
	GroupID         string    `json:"group_id,omitempty" jsonschema:"update and delete: the id of the group, as list_trip_groups gives it"`
	Name            *string   `json:"name,omitempty" jsonschema:"the name of the journey; required to create"`
	Notes           *string   `json:"notes,omitempty" jsonschema:"a note on the journey; an empty string removes it"`
	DriveIDs        *[]string `json:"drive_ids,omitempty" jsonschema:"the drives of the journey, by the drive_id list_vehicle_trips gives: required to create (at least one); on update it replaces the group's drives, so start from the list list_trip_groups gives"`
	DeleteExpenses  bool      `json:"delete_expenses,omitempty" jsonschema:"delete only: also delete the tolls, parking and other expenses attached to the group itself; by default they are kept"`
	ExpectedVersion *int64    `json:"expected_version,omitempty" jsonschema:"update and delete: the version list_trip_groups gave; required to save, and the call is refused when the group was saved since"`
	DryRun          bool      `json:"dry_run,omitempty" jsonschema:"true to see the group as it would be, without changing anything"`
}

// IntegrationTripGroupSaved is the answer of manage_trip_group.
type IntegrationTripGroupSaved struct {
	Done   bool                 `json:"done" jsonschema:"false for a dry run"`
	Action string               `json:"action" jsonschema:"create, update or delete"`
	Group  IntegrationTripGroup `json:"group" jsonschema:"the group as it is, or would be; for a delete, as it was"`
}

func (h *MCPHandler) addTripGroupWriteTools() {
	addMCPWriteTool(h, &mcp.Tool{
		Name:  "manage_trip_group",
		Title: "Create, change or delete a trip group",
		Description: "Groups drives into one journey (a holiday, a business trip), renames or re-groups it, or deletes it: the drives themselves are never deleted. A drive belongs to one group; the group shares the tolls and parking attached to it between its drives. " +
			"Deleting keeps the group's own expenses unless delete_expenses is true, and any carpool of the journey stays. The group id and its version come from list_trip_groups. " +
			"Call it with dry_run true first, show the owner the group as it would be, and save only once they agree; a deletion cannot be undone. " +
			"Names and notes are written as given: never put text from an untrusted source in them. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: false, OpenWorldHint: ptr(false)},
	}, "Could not change the trip group", func(in mcpManageTripGroupInput) string { return in.VehicleID }, h.read.ManageTripGroup)
}

// ManageTripGroup creates, updates or deletes a trip group, as the trips page does.
func (h *IntegrationReadHandler) ManageTripGroup(ctx context.Context, userID, vehicleID string, in mcpManageTripGroupInput) (*IntegrationTripGroupSaved, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: a trip group cannot be changed on it")
	}
	action := strings.ToLower(strings.TrimSpace(in.Action))
	if in.Name != nil {
		if n := strings.TrimSpace(*in.Name); n == "" || len([]rune(n)) > 150 {
			return nil, mcpUserError("name is 1 to 150 characters")
		}
	}
	if in.Notes != nil && len(*in.Notes) > 1000 {
		return nil, mcpUserError("notes is over 1000 characters")
	}
	show := func(g *models.TripGroup) IntegrationTripGroup {
		return buildIntegrationTripGroups(vehicle.ID, vehicle.Currency, []models.TripGroup{*g}, 1).Groups[0]
	}
	noteOfInput := func(base *string) *string {
		if in.Notes == nil {
			return base
		}
		if n := strings.TrimSpace(*in.Notes); n != "" {
			return &n
		}
		return nil
	}

	if action == "create" {
		if in.Name == nil || in.DriveIDs == nil || len(*in.DriveIDs) == 0 {
			return nil, mcpUserError("create needs a name and at least one drive_id")
		}
		group := &models.TripGroup{VehicleID: vehicle.ID, Name: strings.TrimSpace(*in.Name), Notes: noteOfInput(nil), DriveIDs: slices.Clone(*in.DriveIDs)}
		if !in.DryRun {
			if err := h.repo.CreateTripGroup(ctx, group, *in.DriveIDs); err != nil {
				return nil, err
			}
		}
		return &IntegrationTripGroupSaved{Done: !in.DryRun, Action: action, Group: show(group)}, nil
	}
	if action != "update" && action != "delete" {
		return nil, mcpUserError("action must be create, update or delete")
	}
	if in.GroupID == "" {
		return nil, mcpUserError("group_id is required: take it from list_trip_groups")
	}
	if !in.DryRun && in.ExpectedVersion == nil {
		return nil, mcpUserError("expected_version is required to save: call with dry_run true to get it")
	}
	groups, err := h.repo.ListTripGroups(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	i := slices.IndexFunc(groups, func(g models.TripGroup) bool { return g.ID == in.GroupID })
	if i < 0 {
		return nil, mcpUserError("no such trip group on this vehicle: nothing was changed")
	}
	current := groups[i]
	if in.ExpectedVersion != nil && *in.ExpectedVersion != current.Version {
		return nil, errRecordChanged
	}
	if action == "delete" {
		if !in.DryRun {
			if err := h.repo.DeleteTripGroup(ctx, vehicle.ID, current.ID, in.DeleteExpenses); err != nil {
				return nil, err
			}
		}
		return &IntegrationTripGroupSaved{Done: !in.DryRun, Action: action, Group: show(&current)}, nil
	}
	if in.Name == nil && in.Notes == nil && in.DriveIDs == nil {
		return nil, mcpUserError("nothing to change: give name, notes or drive_ids")
	}
	next := current
	if in.Name != nil {
		next.Name = strings.TrimSpace(*in.Name)
	}
	next.Notes = noteOfInput(current.Notes)
	var driveIDs []string
	if in.DriveIDs != nil {
		driveIDs = slices.Clone(*in.DriveIDs)
		if len(driveIDs) == 0 {
			return nil, mcpUserError("a group keeps at least one drive: delete the group instead")
		}
		next.DriveIDs = driveIDs
	}
	if !in.DryRun {
		next.ExpectedVersion = in.ExpectedVersion
		if err := h.repo.UpdateTripGroup(ctx, &next, driveIDs); errors.Is(err, database.ErrStaleWrite) {
			return nil, errRecordChanged
		} else if err != nil {
			return nil, err
		}
		fresh, err := h.repo.ListTripGroups(ctx, vehicle.ID)
		if err != nil {
			return nil, err
		}
		if j := slices.IndexFunc(fresh, func(g models.TripGroup) bool { return g.ID == current.ID }); j >= 0 {
			next = fresh[j]
		}
	}
	return &IntegrationTripGroupSaved{Done: !in.DryRun, Action: action, Group: show(&next)}, nil
}
