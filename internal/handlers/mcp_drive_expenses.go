package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/money"
)

// maxDriveExpenseItems bounds a batch, so that one call cannot rewrite a whole history.
const maxDriveExpenseItems = 50

type mcpDriveExpenseItem struct {
	DriveID         string  `json:"drive_id" jsonschema:"the trip to change, by the drive_id list_vehicle_trips gives"`
	Tolls           *string `json:"tolls,omitempty" jsonschema:"what the tolls of this trip cost in all, a decimal string in the vehicle's currency such as \"14.60\": it replaces the amount recorded, it is not added to it; \"0\" removes the tolls of the trip"`
	Parking         *string `json:"parking,omitempty" jsonschema:"what parking cost on this trip in all, a decimal string; it replaces the amount recorded; \"0\" removes it"`
	ExpectedTolls   *string `json:"expected_tolls,omitempty" jsonschema:"the tolls of the trip as you read them (\"0.00\" when none): the whole call is refused when the trip holds another amount now"`
	ExpectedParking *string `json:"expected_parking,omitempty" jsonschema:"the parking of the trip as you read it (\"0.00\" when none): the whole call is refused when the trip holds another amount now"`
}

type mcpUpdateDriveExpensesInput struct {
	VehicleID string                `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles; its name is accepted when no other vehicle has the same"`
	Items     []mcpDriveExpenseItem `json:"items" jsonschema:"the trips to change and what their tolls and parking cost, 50 at most; one item for one trip"`
	DryRun    bool                  `json:"dry_run,omitempty" jsonschema:"true to see what each trip would hold, without changing anything"`
}

// IntegrationDriveExpenseChange is what one kind of expense of one trip was, and is now.
type IntegrationDriveExpenseChange struct {
	Kind   string            `json:"kind" jsonschema:"tolls or parking"`
	Action string            `json:"action" jsonschema:"created, updated, deleted or unchanged"`
	Before IntegrationAmount `json:"before" jsonschema:"the amount the trip held before; 0 when it held none"`
	After  IntegrationAmount `json:"after" jsonschema:"the amount the trip holds after the change; 0 when removed"`
}

// IntegrationDriveExpenses lists what a trip was changed to.
type IntegrationDriveExpenses struct {
	DriveID string                          `json:"drive_id" jsonschema:"the trip"`
	Changes []IntegrationDriveExpenseChange `json:"changes" jsonschema:"what changed on this trip, one entry per kind given"`
}

// IntegrationDriveExpensesUpdated is the answer of update_drive_expenses.
type IntegrationDriveExpensesUpdated struct {
	Saved  bool                       `json:"saved" jsonschema:"true when the changes were saved; false for a dry run, which changes nothing"`
	Drives []IntegrationDriveExpenses `json:"drives" jsonschema:"the trips, in the order given"`
	// CarpoolIDs are the carpools these trips are legs of: one that follows its drives takes the new cost when it is read; a frozen one keeps its amounts.
	CarpoolIDs []string `json:"carpool_ids" jsonschema:"the carpools that have one of these trips as a leg, as get_carpool lists them: read them again to see their shares; a frozen carpool keeps its amounts"`
}

func (h *MCPHandler) addDriveExpenseTools() {
	mcp.AddTool(h.server, &mcp.Tool{
		Name:  "update_drive_expenses",
		Title: "Change the tolls and parking of trips",
		Description: "Sets what the tolls and the parking of one or several trips cost: pass the drive_id of each (from list_vehicle_trips) and the amount. The amount replaces what the trip holds, an estimate included, and is then recorded as entered by the owner; \"0\" removes it. " +
			"The call applies all its items or none. Pass expected_tolls / expected_parking, the amounts you read, to be refused when a trip was changed meanwhile. " +
			"A trip whose cost is shared inside a trip group, one with several expenses of the same kind, and one with an expense in another currency than the vehicle's are refused, with the reason. " +
			"This overwrites the owner's records: call it with dry_run true first, show the owner each trip with its old and new amount, and save only once they agree. " +
			"The carpools that have these trips as legs are listed in the answer: those that follow their drives take the new cost when read. Needs a connection the owner allowed to write.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true), IdempotentHint: true, OpenWorldHint: ptr(false)},
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpUpdateDriveExpensesInput) (*mcp.CallToolResult, IntegrationDriveExpensesUpdated, error) {
		if !mcpCanWrite(req) {
			return nil, IntegrationDriveExpensesUpdated{}, errors.New("this connection can only read: the owner has to allow writing when connecting the assistant to AutoLedger")
		}
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationDriveExpensesUpdated{}, mcpFailure(ctx, "update_drive_expenses", err, "Could not change the expenses")
		}
		out, err := h.read.UpdateDriveExpenses(ctx, mcpUserID(req), vehicleID, in)
		if err != nil {
			return nil, IntegrationDriveExpensesUpdated{}, mcpFailure(ctx, "update_drive_expenses", err, "Could not change the expenses")
		}
		return nil, *out, nil
	})
}

// driveExpenseKinds pairs the kinds a call names with the expense types they are stored as.
var driveExpenseKinds = []struct{ kind, expenseType string }{{"tolls", "TOLL"}, {"parking", "PARKING"}}

// expenseAmountOf reads a decimal string given for one kind.
func expenseAmountOf(raw, label string) (money.Cents, error) {
	amount, err := money.Parse(strings.TrimSpace(raw))
	if err != nil || amount < 0 || amount > money.Max {
		return 0, mcpUserError(fmt.Sprintf("%s must be a positive decimal string such as \"14.60\", \"0\" to remove it", label))
	}
	return amount, nil
}

// driveExpenseChanges turns the items of a call into the changes to apply, in the order given.
func driveExpenseChanges(items []mcpDriveExpenseItem) ([]database.DriveExpenseChange, error) {
	if len(items) == 0 {
		return nil, mcpUserError("items is empty: give the trips to change, with their tolls or parking")
	}
	if len(items) > maxDriveExpenseItems {
		return nil, mcpUserError(fmt.Sprintf("%d trips at most in one call: split the list", maxDriveExpenseItems))
	}
	seen := make(map[string]bool, len(items))
	var changes []database.DriveExpenseChange
	for _, item := range items {
		id := strings.TrimSpace(item.DriveID)
		if !vehicleIDPattern.MatchString(id) {
			return nil, mcpUserError(fmt.Sprintf("%q is not a drive id: use the drive_id of list_vehicle_trips", item.DriveID))
		}
		if seen[id] {
			return nil, mcpUserError(fmt.Sprintf("the trip %s is given twice: give it once", id))
		}
		seen[id] = true
		given := map[string]*string{"tolls": item.Tolls, "parking": item.Parking}
		expected := map[string]*string{"tolls": item.ExpectedTolls, "parking": item.ExpectedParking}
		count := 0
		for _, k := range driveExpenseKinds {
			if given[k.kind] == nil {
				if expected[k.kind] != nil {
					return nil, mcpUserError(fmt.Sprintf("expected_%s of %s is given without %s: give both or neither", k.kind, id, k.kind))
				}
				continue
			}
			amount, err := expenseAmountOf(*given[k.kind], k.kind+" of "+id)
			if err != nil {
				return nil, err
			}
			change := database.DriveExpenseChange{DriveID: id, Type: k.expenseType, Amount: amount}
			if expected[k.kind] != nil {
				was, err := expenseAmountOf(*expected[k.kind], "expected_"+k.kind+" of "+id)
				if err != nil {
					return nil, err
				}
				change.ExpectedAmount = &was
			}
			changes = append(changes, change)
			count++
		}
		if count == 0 {
			return nil, mcpUserError(fmt.Sprintf("nothing to change on %s: give tolls or parking", id))
		}
	}
	return changes, nil
}

// UpdateDriveExpenses sets the tolls and the parking of trips, all or none. With in.DryRun nothing is stored.
func (h *IntegrationReadHandler) UpdateDriveExpenses(ctx context.Context, userID, vehicleID string, in mcpUpdateDriveExpensesInput) (*IntegrationDriveExpensesUpdated, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if !vehicle.Role.CanEdit() {
		return nil, mcpUserError("the owner shares this vehicle read-only: its expenses cannot be changed")
	}
	changes, err := driveExpenseChanges(in.Items)
	if err != nil {
		return nil, err
	}
	results, err := h.repo.ApplyDriveExpenseChanges(ctx, vehicle.ID, vehicle.Currency, changes, in.DryRun)
	if err != nil {
		return nil, driveExpenseRefusal(err, vehicle.Currency)
	}

	byDrive := make(map[string]*IntegrationDriveExpenses, len(in.Items))
	out := &IntegrationDriveExpensesUpdated{Saved: !in.DryRun, Drives: make([]IntegrationDriveExpenses, 0, len(in.Items)), CarpoolIDs: []string{}}
	ids := make([]string, 0, len(in.Items))
	for _, item := range in.Items {
		id := strings.TrimSpace(item.DriveID)
		ids = append(ids, id)
		out.Drives = append(out.Drives, IntegrationDriveExpenses{DriveID: id, Changes: []IntegrationDriveExpenseChange{}})
		byDrive[id] = &out.Drives[len(out.Drives)-1]
	}
	for _, r := range results {
		kind := "tolls"
		if r.Type == "PARKING" {
			kind = "parking"
		}
		d := byDrive[r.DriveID]
		d.Changes = append(d.Changes, IntegrationDriveExpenseChange{
			Kind: kind, Action: r.Action, Before: amountOf(r.Before, vehicle.Currency), After: amountOf(r.After, vehicle.Currency),
		})
	}

	carpools, err := h.repo.GetCarpoolsForDrives(ctx, vehicle.ID, ids)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, id := range ids {
		for _, c := range carpools[id] {
			if !seen[c.ID] {
				seen[c.ID] = true
				out.CarpoolIDs = append(out.CarpoolIDs, c.ID)
			}
		}
	}
	return out, nil
}

// driveExpenseRefusal words what the repository refused, with what to do next.
func driveExpenseRefusal(err error, currency string) error {
	var conflict *database.DriveExpenseConflict
	switch {
	case errors.Is(err, database.ErrForeignReference):
		return mcpUserError("a drive_id is not a drive of this vehicle: take them from list_vehicle_trips for the same vehicle")
	case errors.As(err, &conflict):
		kind := "tolls"
		if conflict.Type == "PARKING" {
			kind = "parking"
		}
		switch conflict.Reason {
		case database.DriveExpenseShared:
			return mcpUserError(fmt.Sprintf("the %s of the trip %s are one cost shared by a trip group: change them in the application, nothing was changed", kind, conflict.DriveID))
		case database.DriveExpenseSeveral:
			return mcpUserError(fmt.Sprintf("the trip %s holds several %s expenses: they cannot be set as one amount here, change them in the application, nothing was changed", conflict.DriveID, kind))
		case database.DriveExpenseForeign:
			return mcpUserError(fmt.Sprintf("the %s of the trip %s were recorded in another currency than %s: change them in the application, nothing was changed", kind, conflict.DriveID, currency))
		case database.DriveExpenseChanged:
			return mcpUserError(fmt.Sprintf("the %s of the trip %s are %s %s now, not what you read: read the trips again with list_vehicle_trips, show the owner, and redo the change; nothing was changed", kind, conflict.DriveID, conflict.Current.String(), currency))
		}
	}
	return err
}
