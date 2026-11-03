package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
)

// QuickRotateTires applies a predefined rotation or a seasonal pack swap.
func (r *Repository) QuickRotateTires(ctx context.Context, vehicleID string, mode string, odometer float64, swapWithPackTireIDs []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := lockVehicleTires(ctx, tx, vehicleID)
	if err != nil {
		return err
	}

	mountedMap := make(map[models.TirePosition]*models.Tire)
	for _, t := range current {
		if isMountedPosition(t.CurrentPosition) {
			mountedMap[t.CurrentPosition] = t
		}
	}

	newPositions := make(map[string]models.TirePosition)
	swap := func(a, b models.TirePosition) {
		if t, ok := mountedMap[a]; ok {
			newPositions[t.ID] = b
		}
		if t, ok := mountedMap[b]; ok {
			newPositions[t.ID] = a
		}
	}

	switch mode {
	case "FRONT_BACK":
		swap(models.TirePosFL, models.TirePosRL)
		swap(models.TirePosFR, models.TirePosRR)
	case "CROSS":
		swap(models.TirePosFL, models.TirePosRR)
		swap(models.TirePosFR, models.TirePosRL)
	case "SWAP_PACK":
		ids := uniqueStrings(swapWithPackTireIDs)
		if len(ids) == 0 || len(ids) > 4 {
			return apierror.New("tire.swap_count", "An axle swap needs between 1 and 4 tires")
		}
		for _, t := range mountedMap {
			newPositions[t.ID] = models.TirePosStorage
		}
		positions := []models.TirePosition{models.TirePosFL, models.TirePosFR, models.TirePosRL, models.TirePosRR}
		for i, tireID := range ids {
			t, ok := current[tireID]
			if !ok {
				return ErrForeignReference
			}
			if isMountedPosition(t.CurrentPosition) || t.CurrentPosition == models.TirePosDisposed {
				return apierror.Newf("tire.not_in_storage", "The tire %s is not in storage", tireID)
			}
			newPositions[tireID] = positions[i]
		}
	default:
		return apierror.Newf("tire.rotation_mode", "Unknown rotation mode: %s", mode)
	}

	notes := fmt.Sprintf("Permutation rapide: %s", mode)
	return r.applyTirePositions(ctx, tx, vehicleID, current, newPositions, time.Now().UTC(), odometer, &notes)
}

// AddTireRotation applies an explicit position mapping ({"FL": "<tire id>", ...}).
func (r *Repository) AddTireRotation(ctx context.Context, rot *models.TireRotation) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := lockVehicleTires(ctx, tx, rot.VehicleID)
	if err != nil {
		return err
	}

	newPositions := make(map[string]models.TirePosition)
	for posStr, tireIDVal := range rot.MappingJSON {
		pos := models.TirePosition(posStr)
		tireID, ok := tireIDVal.(string)
		if !ok || tireID == "" {
			continue
		}
		if !isValidTirePosition(pos) {
			return apierror.Newf("tire.position_invalid", "Invalid tire position: %s", posStr)
		}
		if _, owned := current[tireID]; !owned {
			return ErrForeignReference
		}
		newPositions[tireID] = pos
	}

	return r.applyTirePositions(ctx, tx, rot.VehicleID, current, newPositions, rot.Date, rot.Odometer, rot.Notes)
}

func lockVehicleTires(ctx context.Context, tx pgx.Tx, vehicleID string) (map[string]*models.Tire, error) {
	rows, err := tx.Query(ctx, `
		SELECT `+tireColumns+`
		FROM tires
		WHERE vehicle_id = $1 AND is_archived = FALSE
		FOR UPDATE;
	`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tires := make(map[string]*models.Tire)
	for rows.Next() {
		var t models.Tire
		if err := scanTire(rows, &t); err != nil {
			return nil, err
		}
		tires[t.ID] = &t
	}
	return tires, rows.Err()
}

// applyTirePositions closes/opens mount sessions for every moved tire, updates positions,
// recomputes distances and records the rotation, all inside the caller's transaction.
func (r *Repository) applyTirePositions(ctx context.Context, tx pgx.Tx, vehicleID string, current map[string]*models.Tire, newPositions map[string]models.TirePosition, at time.Time, odometer float64, notes *string) error {
	if len(newPositions) == 0 {
		return apierror.New("tire.nothing_to_move", "No tire to move")
	}
	if odometer <= 0 {
		return apierror.New("odometer.valid_required", "A valid odometer reading is required")
	}
	for tireID := range newPositions {
		if t := current[tireID]; t.MountedOdometer != nil && odometer < *t.MountedOdometer {
			return apierror.Newf("tire.odometer_below_mount_tire", "The odometer (%.0f km) is lower than the fitting odometer of the tire %s (%.0f km)", apierror.Km(odometer), tireID, apierror.Km(*t.MountedOdometer))
		}
	}

	// Two tires must never end up on the same wheel.
	finalPositions := make(map[models.TirePosition]string)
	for id, t := range current {
		pos := t.CurrentPosition
		if p, moved := newPositions[id]; moved {
			pos = p
		}
		if isMountedPosition(pos) {
			if other, taken := finalPositions[pos]; taken {
				return apierror.Newf("tire.position_double", "The position %s would be occupied by two tires (%s and %s)", pos, other, id)
			}
			finalPositions[pos] = id
		}
	}

	mapping := make(map[string]any)
	var stored []string
	for tireID, newPos := range newPositions {
		t := current[tireID]
		if newPos == t.CurrentPosition {
			continue
		}
		if isMountedPosition(newPos) {
			mapping[string(newPos)] = tireID
		} else {
			stored = append(stored, tireID)
		}

		if isMountedPosition(t.CurrentPosition) {
			if _, err := tx.Exec(ctx, `
				UPDATE tire_mount_sessions
				SET dismounted_date = $1, dismounted_odometer = $2,
				    distance_km = GREATEST($2 - mounted_odometer, 0), updated_at = NOW()
				WHERE tire_id = $3 AND dismounted_date IS NULL;
			`, at, odometer, tireID); err != nil {
				return fmt.Errorf("failed to close mount session of tire %s: %w", tireID, err)
			}
		}
		if isMountedPosition(newPos) {
			if _, err := tx.Exec(ctx, `
				INSERT INTO tire_mount_sessions (tire_id, vehicle_id, position, mounted_date, mounted_odometer)
				VALUES ($1, $2, $3, $4, $5);
			`, tireID, vehicleID, newPos, at, odometer); err != nil {
				return fmt.Errorf("failed to open mount session of tire %s: %w", tireID, err)
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE tires SET current_position = $1, version = version + 1, updated_at = NOW() WHERE id = $2;`, newPos, tireID); err != nil {
			return fmt.Errorf("failed to update position of tire %s: %w", tireID, err)
		}
		if err := recalcTireDistance(ctx, tx, tireID); err != nil {
			return err
		}
	}
	if len(stored) > 0 {
		mapping[string(models.TirePosStorage)] = stored
	}

	mapBytes, err := json.Marshal(mapping)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO tire_rotations (vehicle_id, date, odometer, mapping_json, notes)
		VALUES ($1, $2, $3, $4, $5);
	`, vehicleID, at, odometer, mapBytes, notes); err != nil {
		return fmt.Errorf("failed to record tire rotation: %w", err)
	}

	return tx.Commit(ctx)
}
