package database

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// TirePatch holds the fields applied to several tires at once; nil fields are left unchanged.
type TirePatch struct {
	Brand               *string
	Model               *string
	Dimension           *string
	Season              *models.TireSeason
	PurchaseDate        *time.Time
	PurchasePrice       *money.Cents // Unit price
	TotalPrice          *money.Cents // Split to the cent across the selected tires (overrides PurchasePrice)
	InitialDepthMm      *float64
	MinLegalDepthMm     *float64
	DotCode             *string
	InitialDistanceKm   *float64
	EstimatedLifespanKm *int
	// Active mount session of mounted tires
	MountedDate     *time.Time
	MountedOdometer *float64
}

// BatchUpdateTires applies a patch to several tires of a vehicle in one transaction.
func (r *Repository) BatchUpdateTires(ctx context.Context, vehicleID string, tireIDs []string, p TirePatch) error {
	ids := uniqueStrings(tireIDs)
	if len(ids) == 0 {
		return apierror.New("tire.none_selected", "No tire selected")
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := lockVehicleTires(ctx, tx, vehicleID)
	if err != nil {
		return err
	}
	var prices []money.Cents
	if p.TotalPrice != nil {
		prices = money.Split(*p.TotalPrice, len(ids))
	}

	for i, id := range ids {
		t, ok := current[id]
		if !ok {
			return ErrForeignReference
		}
		if p.Brand != nil {
			t.Brand = *p.Brand
		}
		if p.Model != nil {
			t.Model = *p.Model
		}
		if p.Dimension != nil {
			t.Dimension = *p.Dimension
		}
		if p.Season != nil {
			t.Season = *p.Season
		}
		if p.PurchaseDate != nil {
			t.PurchaseDate = *p.PurchaseDate
		}
		if prices != nil {
			t.PurchasePrice = prices[i]
		} else if p.PurchasePrice != nil {
			t.PurchasePrice = *p.PurchasePrice
		}
		if p.InitialDepthMm != nil {
			t.InitialDepthMm = *p.InitialDepthMm
		}
		if p.MinLegalDepthMm != nil {
			t.MinLegalDepthMm = *p.MinLegalDepthMm
		}
		if p.DotCode != nil {
			t.DotCode = p.DotCode
		}
		if p.InitialDistanceKm != nil {
			t.InitialDistanceKm = *p.InitialDistanceKm
		}
		if p.EstimatedLifespanKm != nil {
			t.EstimatedLifespanKm = *p.EstimatedLifespanKm
		}
		t.VehicleID = &vehicleID
		if err := updateTireTx(ctx, tx, t); err != nil {
			return err
		}

		if (p.MountedDate != nil || p.MountedOdometer != nil) && isMountedPosition(t.CurrentPosition) {
			if err := setActiveMount(ctx, tx, t, vehicleID, p.MountedDate, p.MountedOdometer); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

// setActiveMount updates the open mount session of a mounted tire, creating it when missing.
func setActiveMount(ctx context.Context, tx pgx.Tx, t *models.Tire, vehicleID string, date *time.Time, odometer *float64) error {
	var sessionID string
	var curDate time.Time
	var curOdo float64
	err := tx.QueryRow(ctx, `
		SELECT id, mounted_date, mounted_odometer FROM tire_mount_sessions
		WHERE tire_id::text = $1 AND dismounted_date IS NULL
		ORDER BY mounted_date DESC LIMIT 1;
	`, t.ID).Scan(&sessionID, &curDate, &curOdo)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if date == nil || odometer == nil {
			return apierror.Newf("tire.no_current_mount", "The tire %s %s has no current fitting: fitting date and odometer required", t.Brand, t.CurrentPosition)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO tire_mount_sessions (tire_id, vehicle_id, position, mounted_date, mounted_odometer)
			VALUES ($1, $2, $3, $4, $5);
		`, t.ID, vehicleID, t.CurrentPosition, *date, *odometer); err != nil {
			return err
		}
	case err != nil:
		return err
	default:
		if date != nil {
			curDate = *date
		}
		if odometer != nil {
			curOdo = *odometer
		}
		if _, err := tx.Exec(ctx, `
			UPDATE tire_mount_sessions SET mounted_date = $1, mounted_odometer = $2, updated_at = NOW() WHERE id = $3;
		`, curDate, curOdo, sessionID); err != nil {
			return err
		}
	}
	return recalcTireDistance(ctx, tx, t.ID)
}

// BatchDisposeTires retires multiple tires in a single transaction.
func (r *Repository) BatchDisposeTires(ctx context.Context, vehicleID string, tireIDs []string, at time.Time, odometer *float64) error {
	if len(tireIDs) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	current, err := lockVehicleTires(ctx, tx, vehicleID)
	if err != nil {
		return err
	}

	for _, tireID := range tireIDs {
		t, ok := current[tireID]
		if !ok {
			return ErrNotFound
		}
		if isMountedPosition(t.CurrentPosition) {
			if odometer == nil {
				return apierror.Newf("tire.dismount_odometer_required_for", "The removal odometer is required for the fitted tire %s", t.Brand)
			}
			if t.MountedOdometer != nil && *odometer < *t.MountedOdometer {
				return apierror.Newf("tire.odometer_below_mount_for", "The odometer (%.0f km) is lower than the fitting odometer (%.0f km) for %s", apierror.Km(*odometer), apierror.Km(*t.MountedOdometer), t.Brand)
			}
			if _, err := tx.Exec(ctx, `
				UPDATE tire_mount_sessions
				SET dismounted_date = $1, dismounted_odometer = $2,
				    distance_km = GREATEST($2 - mounted_odometer, 0), updated_at = NOW()
				WHERE tire_id = $3 AND dismounted_date IS NULL;
			`, at, *odometer, tireID); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(ctx, `UPDATE tires SET current_position = 'DISPOSED', version = version + 1, updated_at = NOW() WHERE id = $1;`, tireID); err != nil {
			return err
		}
		if err := recalcTireDistance(ctx, tx, tireID); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// CreateTiresBatch inserts tires and their initial mount sessions atomically.
// AccumulatedDistanceKm provided at creation is recorded as the tire's initial distance.
func (r *Repository) CreateTiresBatch(ctx context.Context, tires []*models.Tire) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Two tires must never end up on the same wheel: lock the vehicle's existing
	// tires so a concurrent insert can't slip a second one onto the same position.
	occupied := make(map[models.TirePosition]string)
	if len(tires) > 0 && tires[0].VehicleID != nil {
		existing, err := lockVehicleTires(ctx, tx, *tires[0].VehicleID)
		if err != nil {
			return err
		}
		for _, et := range existing {
			if isMountedPosition(et.CurrentPosition) {
				occupied[et.CurrentPosition] = et.Brand + " " + et.Model
			}
		}
	}

	for _, t := range tires {
		lifespan := t.EstimatedLifespanKm
		if lifespan <= 0 {
			lifespan = 40000
		}
		if !isValidTirePosition(t.CurrentPosition) {
			return apierror.Newf("tire.position_invalid", "Invalid tire position: %s", t.CurrentPosition)
		}
		if isMountedPosition(t.CurrentPosition) && (t.MountedOdometer == nil || t.VehicleID == nil) {
			return apierror.New("tire.mounted_needs_odometer", "A fitted tire needs the fitting odometer")
		}
		if isMountedPosition(t.CurrentPosition) {
			if other, taken := occupied[t.CurrentPosition]; taken {
				return apierror.Newf("tire.position_taken", "The position %s is already occupied by %s: scrap it or change its position before fitting a new one", t.CurrentPosition, other)
			}
			occupied[t.CurrentPosition] = t.Brand + " " + t.Model
		}
		if !isMountedPosition(t.CurrentPosition) {
			t.MountedOdometer = nil
		}
		t.InitialDistanceKm = math.Max(0, t.AccumulatedDistanceKm)

		query := `
			INSERT INTO tires (
				vehicle_id, brand, model, dimension, season,
				purchase_date, purchase_price, current_position,
				initial_depth_mm, min_legal_depth_mm, dot_code,
				mounted_odometer, initial_distance_km, accumulated_distance_km, estimated_lifespan_km
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13, $14)
			RETURNING id, created_at, updated_at, version;
		`
		if err := tx.QueryRow(ctx, query,
			t.VehicleID, t.Brand, t.Model, t.Dimension, t.Season,
			t.PurchaseDate, t.PurchasePrice, t.CurrentPosition,
			t.InitialDepthMm, t.MinLegalDepthMm, t.DotCode,
			t.MountedOdometer, t.InitialDistanceKm, lifespan,
		).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt, &t.Version); err != nil {
			return fmt.Errorf("failed to insert tire: %w", err)
		}
		t.AccumulatedDistanceKm = t.InitialDistanceKm
		t.EstimatedLifespanKm = lifespan

		if isMountedPosition(t.CurrentPosition) {
			sessionQuery := `
				INSERT INTO tire_mount_sessions (
					tire_id, vehicle_id, position, mounted_date, mounted_odometer
				) VALUES ($1, $2, $3, $4, $5);
			`
			if _, err := tx.Exec(ctx, sessionQuery, t.ID, *t.VehicleID, t.CurrentPosition, t.PurchaseDate, *t.MountedOdometer); err != nil {
				return fmt.Errorf("failed to create mount session: %w", err)
			}
		}
	}

	return tx.Commit(ctx)
}
