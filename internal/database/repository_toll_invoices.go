package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// tollInvoiceSelect reads an invoice with the tolls entered individually in its period, converted to the vehicle's
// currency the way cost_ledger does.
const tollInvoiceSelect = `
	SELECT i.id::text, i.vehicle_id::text, i.month, i.period_start, i.period_end, i.total, i.note, i.version, i.updated_at,
	       COALESCE((
	           SELECT SUM(ROUND(CASE WHEN e.currency = v.currency THEN e.amount ELSE e.amount * e.fx_rate END, 2))
	           FROM drive_expenses e
	           JOIN vehicles v ON v.id = e.vehicle_id
	           WHERE e.vehicle_id = i.vehicle_id AND e.type = 'TOLL'
	             AND e.date >= i.period_start AND e.date < i.period_end
	             AND (e.currency = v.currency OR e.fx_rate IS NOT NULL)
	       ), 0)
	FROM toll_invoices i
`

func scanTollInvoice(row pgx.Row) (*models.TollInvoice, error) {
	var t models.TollInvoice
	if err := row.Scan(&t.ID, &t.VehicleID, &t.Month, &t.PeriodStart, &t.PeriodEnd, &t.Total, &t.Note, &t.Version, &t.UpdatedAt, &t.TripTolls); err != nil {
		return nil, err
	}
	t.Gap = t.Total - t.TripTolls
	return &t, nil
}

// ListTollInvoices returns the invoices of a vehicle, the latest period first.
func (r *Repository) ListTollInvoices(ctx context.Context, vehicleID string) ([]models.TollInvoice, error) {
	rows, err := r.pool.Query(ctx, tollInvoiceSelect+` WHERE i.vehicle_id = $1 ORDER BY i.period_start DESC;`, vehicleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []models.TollInvoice{}
	for rows.Next() {
		t, err := scanTollInvoice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// GetTollInvoice returns the invoice of a vehicle for a month, or ErrNotFound.
func (r *Repository) GetTollInvoice(ctx context.Context, vehicleID, month string) (*models.TollInvoice, error) {
	t, err := scanTollInvoice(r.pool.QueryRow(ctx, tollInvoiceSelect+` WHERE i.vehicle_id = $1 AND i.month = $2;`, vehicleID, month))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// SaveTollInvoice creates the invoice of a month or replaces it. A save that names the version it read is refused with
// ErrStaleWrite when the invoice has moved on, or when it was created meanwhile (expected non-nil on a month that has
// none is ErrNotFound).
func (r *Repository) SaveTollInvoice(ctx context.Context, vehicleID, month string, start, end time.Time, total money.Cents, note *string, expected *int64) (*models.TollInvoice, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var id string
	var version int64
	err = tx.QueryRow(ctx, `SELECT id::text, version FROM toll_invoices WHERE vehicle_id = $1 AND month = $2 FOR UPDATE;`, vehicleID, month).Scan(&id, &version)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		if expected != nil {
			return nil, ErrNotFound
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO toll_invoices (vehicle_id, month, period_start, period_end, total, note)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text;
		`, vehicleID, month, start, end, total, note).Scan(&id)
	case err != nil:
		return nil, err
	default:
		if expected != nil && *expected != version {
			return nil, ErrStaleWrite
		}
		_, err = tx.Exec(ctx, `
			UPDATE toll_invoices
			SET period_start = $2, period_end = $3, total = $4, note = $5, version = version + 1, updated_at = NOW()
			WHERE id = $1;
		`, id, start, end, total, note)
	}
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return r.GetTollInvoice(ctx, vehicleID, month)
}

// DeleteTollInvoice removes the invoice of a month: the individual tolls count again. ErrNotFound when there is none,
// ErrStaleWrite when expected names an older version.
func (r *Repository) DeleteTollInvoice(ctx context.Context, vehicleID, month string, expected *int64) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM toll_invoices WHERE vehicle_id = $1 AND month = $2 AND ($3::bigint IS NULL OR version = $3);
	`, vehicleID, month, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return staleOrNotFoundInvoice(ctx, r, vehicleID, month, expected)
	}
	return nil
}

func staleOrNotFoundInvoice(ctx context.Context, r *Repository, vehicleID, month string, expected *int64) error {
	if expected == nil {
		return ErrNotFound
	}
	var exists bool
	if err := r.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM toll_invoices WHERE vehicle_id = $1 AND month = $2);`, vehicleID, month).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return ErrStaleWrite
	}
	return ErrNotFound
}
