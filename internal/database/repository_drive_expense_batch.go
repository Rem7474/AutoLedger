package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// Actions of a DriveExpenseResult.
const (
	DriveExpenseCreated   = "created"
	DriveExpenseUpdated   = "updated"
	DriveExpenseDeleted   = "deleted"
	DriveExpenseUnchanged = "unchanged"
)

// DriveExpenseChange sets what one drive costs in one type of travel expense (TOLL, PARKING). A zero Amount removes the
// expense of the drive. ExpectedAmount, when set, is the amount the caller read: the change is refused if the drive
// holds another amount now.
type DriveExpenseChange struct {
	DriveID        string
	Type           string
	Amount         money.Cents
	ExpectedAmount *money.Cents
}

// DriveExpenseResult is what one change did, or would do.
type DriveExpenseResult struct {
	DriveID string
	Type    string
	Action  string
	Before  money.Cents
	After   money.Cents
}

// DriveExpenseConflict says why a change cannot be applied to a drive. The batch it belongs to is not applied.
type DriveExpenseConflict struct {
	DriveID string
	Type    string
	// Reason is DriveExpenseShared, DriveExpenseSeveral, DriveExpenseForeign or DriveExpenseChanged.
	Reason string
	// Current is the amount the drive holds, for DriveExpenseChanged.
	Current money.Cents
}

// Reasons of a DriveExpenseConflict.
const (
	DriveExpenseShared  = "shared"  // the expense belongs to a trip group, split between its drives
	DriveExpenseSeveral = "several" // the drive holds more than one expense of this type
	DriveExpenseForeign = "foreign" // the expense is in another currency than the vehicle's
	DriveExpenseChanged = "changed" // the drive holds another amount than the one the caller read
)

func (e *DriveExpenseConflict) Error() string {
	return fmt.Sprintf("drive %s, %s: %s", e.DriveID, e.Type, e.Reason)
}

// ApplyDriveExpenseChanges sets the direct travel expenses of drives in one transaction: either every change is applied
// or none is. A drive without an expense of the type gets one (dated the day of the drive, in the vehicle's currency,
// entered by hand); one that has it is overwritten, whatever its source. With dryRun the changes are checked and
// computed exactly as for a save, then rolled back.
func (r *Repository) ApplyDriveExpenseChanges(ctx context.Context, vehicleID, currency string, changes []DriveExpenseChange, dryRun bool) ([]DriveExpenseResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	results := make([]DriveExpenseResult, 0, len(changes))
	for _, c := range changes {
		res, err := applyDriveExpenseChange(ctx, tx, vehicleID, currency, c)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	if dryRun {
		return results, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return results, nil
}

func applyDriveExpenseChange(ctx context.Context, tx pgx.Tx, vehicleID, currency string, c DriveExpenseChange) (DriveExpenseResult, error) {
	res := DriveExpenseResult{DriveID: c.DriveID, Type: c.Type, After: c.Amount}

	var start time.Time
	err := tx.QueryRow(ctx, `
		SELECT start_time FROM drives WHERE id::text = $1 AND vehicle_id = $2 AND deleted_upstream_at IS NULL FOR UPDATE;
	`, c.DriveID, vehicleID).Scan(&start)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, ErrForeignReference
	}
	if err != nil {
		return res, err
	}

	var shared bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM drive_expenses e
			JOIN trip_group_drives tgd ON tgd.trip_group_id = e.trip_group_id
			WHERE e.vehicle_id = $1 AND e.type = $3 AND tgd.drive_id::text = $2
		);
	`, vehicleID, c.DriveID, c.Type).Scan(&shared); err != nil {
		return res, err
	}
	if shared {
		return res, &DriveExpenseConflict{DriveID: c.DriveID, Type: c.Type, Reason: DriveExpenseShared}
	}

	rows, err := tx.Query(ctx, `
		SELECT id::text, amount, currency FROM drive_expenses
		WHERE vehicle_id = $1 AND drive_id::text = $2 AND type = $3 ORDER BY created_at FOR UPDATE;
	`, vehicleID, c.DriveID, c.Type)
	if err != nil {
		return res, err
	}
	type held struct {
		id       string
		amount   money.Cents
		currency string
	}
	var existing []held
	for rows.Next() {
		var h held
		if err := rows.Scan(&h.id, &h.amount, &h.currency); err != nil {
			rows.Close()
			return res, err
		}
		existing = append(existing, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	if len(existing) > 1 {
		return res, &DriveExpenseConflict{DriveID: c.DriveID, Type: c.Type, Reason: DriveExpenseSeveral}
	}
	if len(existing) == 1 {
		if existing[0].currency != currency {
			return res, &DriveExpenseConflict{DriveID: c.DriveID, Type: c.Type, Reason: DriveExpenseForeign}
		}
		res.Before = existing[0].amount
	}
	if c.ExpectedAmount != nil && *c.ExpectedAmount != res.Before {
		return res, &DriveExpenseConflict{DriveID: c.DriveID, Type: c.Type, Reason: DriveExpenseChanged, Current: res.Before}
	}

	switch {
	case c.Amount == res.Before:
		res.Action = DriveExpenseUnchanged
	case c.Amount == 0:
		res.Action = DriveExpenseDeleted
		_, err = tx.Exec(ctx, `DELETE FROM drive_expenses WHERE id::text = $1 AND vehicle_id = $2;`, existing[0].id, vehicleID)
	case len(existing) == 0:
		res.Action = DriveExpenseCreated
		_, err = tx.Exec(ctx, `
			INSERT INTO drive_expenses (vehicle_id, drive_id, type, amount, currency, date, source)
			VALUES ($1, $2, $3, $4, $5, $6, $7);
		`, vehicleID, c.DriveID, c.Type, c.Amount, currency, time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC), models.ExpenseSourceManual)
	default:
		res.Action = DriveExpenseUpdated
		_, err = tx.Exec(ctx, `
			UPDATE drive_expenses SET amount = $1, source = $2, version = version + 1
			WHERE id::text = $3 AND vehicle_id = $4;
		`, c.Amount, models.ExpenseSourceManual, existing[0].id, vehicleID)
	}
	return res, err
}
