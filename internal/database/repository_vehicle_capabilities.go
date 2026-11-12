package database

import (
	"context"
	"fmt"

	"github.com/teslacost/teslacost/internal/models"
)

// FillVehicleCapabilities sets Capabilities on every vehicle with one query for the whole set. It reads the role, the
// connection and the toll detection already loaded on the vehicle, so it runs after those are filled and before
// sanitizeVehicleForRole hides the connection from a member.
func (r *Repository) FillVehicleCapabilities(ctx context.Context, vehicles ...*models.Vehicle) error {
	if len(vehicles) == 0 {
		return nil
	}
	ids := make([]string, len(vehicles))
	for i, v := range vehicles {
		ids[i] = v.ID
	}
	rows, err := r.pool.Query(ctx, `
		SELECT v.id::text,
		       EXISTS (SELECT 1 FROM drives d WHERE d.vehicle_id = v.id),
		       EXISTS (SELECT 1 FROM odometer_checkpoints o WHERE o.vehicle_id = v.id AND o.source = 'HA')
		FROM vehicles v WHERE v.id::text = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("failed to read vehicle capabilities: %w", err)
	}
	defer rows.Close()
	facts := make(map[string]models.CapabilityFacts, len(vehicles))
	for rows.Next() {
		var id string
		var f models.CapabilityFacts
		if err := rows.Scan(&id, &f.HasDrives, &f.HasSourceOdometer); err != nil {
			return err
		}
		facts[id] = f
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, v := range vehicles {
		v.Capabilities = models.ComputeCapabilities(v, facts[v.ID])
	}
	return nil
}
