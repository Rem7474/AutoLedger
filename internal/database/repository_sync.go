package database

import (
	"context"
	"time"

	"github.com/teslacost/teslacost/internal/models"
)

// Per-resource sync-state bookkeeping (has a full import completed, last success).
// IsFullImportCompleted reports whether the complete TeslaMate history of a resource was imported once.
func (r *Repository) IsFullImportCompleted(ctx context.Context, vehicleID, resource string) (bool, error) {
	var completed bool
	err := r.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM sync_state
			WHERE vehicle_id = $1 AND resource = $2 AND full_import_completed_at IS NOT NULL
		);
	`, vehicleID, resource).Scan(&completed)
	return completed, err
}

// MarkSyncSuccess records a successful sync pass; fullImport marks the complete history as imported.
func (r *Repository) MarkSyncSuccess(ctx context.Context, vehicleID, resource string, fullImport bool) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO sync_state (vehicle_id, resource, full_import_completed_at, last_success_at)
		VALUES ($1, $2, CASE WHEN $3 THEN NOW() END, NOW())
		ON CONFLICT (vehicle_id, resource) DO UPDATE
		SET last_success_at = NOW(),
		    full_import_completed_at = COALESCE(sync_state.full_import_completed_at, EXCLUDED.full_import_completed_at);
	`, vehicleID, resource, fullImport)
	return err
}

// SyncState is when a resource of a vehicle was last synchronized and when its complete history was imported.
type SyncState struct {
	Resource              string
	FullImportCompletedAt *time.Time
	LastSuccessAt         *time.Time
}

// ListSyncStates lists the synchronization state of each resource of a vehicle.
func (r *Repository) ListSyncStates(ctx context.Context, vehicleID string) ([]SyncState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT resource, full_import_completed_at, last_success_at
		FROM sync_state
		WHERE vehicle_id = $1
		ORDER BY resource;
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SyncState{}
	for rows.Next() {
		var s SyncState
		if err := rows.Scan(&s.Resource, &s.FullImportCompletedAt, &s.LastSuccessAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// ============================================================================
// Carpooling Module
// ============================================================================

// UpsertBatterySnapshot keeps the battery health of a day: a later reading the same day replaces the earlier one.
func (r *Repository) UpsertBatterySnapshot(ctx context.Context, vehicleID string, day time.Time, snap models.BatterySnapshot) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO battery_health_snapshots (vehicle_id, captured_on, max_capacity_kwh, current_capacity_kwh, health_percent)
		VALUES ($1, $2::date, $3, $4, $5)
		ON CONFLICT (vehicle_id, captured_on) DO UPDATE
		SET max_capacity_kwh = EXCLUDED.max_capacity_kwh,
		    current_capacity_kwh = EXCLUDED.current_capacity_kwh,
		    health_percent = EXCLUDED.health_percent;
	`, vehicleID, day.Format("2006-01-02"), snap.MaxCapacityKwh, snap.CurrentCapacityKwh, snap.HealthPercent)
	return err
}
