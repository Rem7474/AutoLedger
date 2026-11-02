package database

import (
	"context"
	"fmt"
)

// versionedTables are the tables whose rows carry a version column that the writes made by a user advance. A save that
// names the version it read is refused when the row has moved on (see ErrStaleWrite). Jobs (sync, geocoding, cost
// refresh) do not advance it: they must not make an open editor fail.
var versionedTables = map[string]bool{
	"carpool_trips":        true,
	"fuel_logs":            true,
	"maintenance_expenses": true,
	"odometer_checkpoints": true,
	"trip_groups":          true,
	"drive_expenses":       true,
}

// staleOrNotFound tells why an UPDATE guarded by a version touched no row: the row is gone (ErrNotFound) or it was saved
// since the caller read it (ErrStaleWrite). Without an expected version the row can only be missing.
func staleOrNotFound(ctx context.Context, q rowQuerier, table, id, vehicleID string, expected *int64) error {
	if expected == nil {
		return ErrNotFound
	}
	if !versionedTables[table] {
		return fmt.Errorf("table %s has no version", table)
	}
	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS (SELECT 1 FROM %s WHERE id::text = $1 AND vehicle_id = $2);`, table)
	if err := q.QueryRow(ctx, query, id, vehicleID).Scan(&exists); err != nil {
		return fmt.Errorf("failed to check %s: %w", table, err)
	}
	if exists {
		return ErrStaleWrite
	}
	return ErrNotFound
}
