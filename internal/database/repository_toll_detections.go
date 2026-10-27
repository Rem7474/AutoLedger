package database

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
)

// UpsertTollDetection stores (or replaces) the toll detection result for a drive.
func (r *Repository) UpsertTollDetection(ctx context.Context, td *models.TollDetection) error {
	segmentsJSON, err := json.Marshal(td.Segments)
	if err != nil {
		return err
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO toll_detections (drive_id, vehicle_id, segments)
		VALUES ($1, $2, $3)
		ON CONFLICT (drive_id) DO UPDATE
		SET segments = EXCLUDED.segments, detected_at = NOW()
		RETURNING id, detected_at;
	`, td.DriveID, td.VehicleID, segmentsJSON).Scan(&td.ID, &td.DetectedAt)
}

// GetTollDetectionByDrive returns the cached toll detection result for a drive, or
// database.ErrNotFound if detection has never been run on it.
func (r *Repository) GetTollDetectionByDrive(ctx context.Context, vehicleID, driveID string) (*models.TollDetection, error) {
	var td models.TollDetection
	var segmentsJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT id, drive_id, vehicle_id, segments, detected_at
		FROM toll_detections
		WHERE vehicle_id = $1 AND drive_id = $2;
	`, vehicleID, driveID).Scan(&td.ID, &td.DriveID, &td.VehicleID, &segmentsJSON, &td.DetectedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(segmentsJSON, &td.Segments); err != nil {
		return nil, err
	}

	return &td, nil
}

// GetNeighbourTeslaMateDrive returns the TeslaMate drive of the vehicle that follows (later) or precedes the given
// drive, or ErrNotFound when there is none. A stop in a service area splits one journey into such neighbours.
func (r *Repository) GetNeighbourTeslaMateDrive(ctx context.Context, vehicleID string, from *models.Drive, later bool) (*models.Drive, error) {
	query := `
		SELECT id::text FROM drives
		WHERE vehicle_id = $1 AND deleted_upstream_at IS NULL AND teslamate_drive_id IS NOT NULL
		  AND id::text <> $3 AND end_time <= $2
		ORDER BY end_time DESC LIMIT 1;`
	at := from.StartTime
	if later {
		query = `
		SELECT id::text FROM drives
		WHERE vehicle_id = $1 AND deleted_upstream_at IS NULL AND teslamate_drive_id IS NOT NULL
		  AND id::text <> $3 AND start_time >= $2
		ORDER BY start_time ASC LIMIT 1;`
		at = from.EndTime
	}
	var id string
	if err := r.pool.QueryRow(ctx, query, vehicleID, at, from.ID).Scan(&id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return r.GetDriveByID(ctx, id, vehicleID)
}
