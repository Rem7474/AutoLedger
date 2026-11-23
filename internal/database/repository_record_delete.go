package database

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/teslacost/teslacost/internal/money"
)

// The kinds of record DeleteRecordAtVersion handles.
const (
	RecordExpense  = "expense"      // maintenance_expenses: a service, insurance, tax, accessory...
	RecordFuelFill = "fuel_fill_up" // fuel_logs
	RecordCharge   = "charge"       // charge_logs, the manual ones only: a synchronised charge would come back on the next sync
)

// RecordSummary describes a record a caller may delete, with the version a delete has to quote.
type RecordSummary struct {
	Kind    string
	ID      string
	Version int64
	Date    time.Time
	Amount  *money.Cents // nil for a charge whose cost is unknown
	Note    *string
}

// recordQueries holds, per kind, the select of a RecordSummary and the table to delete from. The identifiers are
// constants of this file: nothing from a caller is put in the SQL text.
var recordQueries = map[string]struct{ selectSQL, table string }{
	RecordExpense:  {`SELECT version, date, amount, description FROM maintenance_expenses WHERE id::text = $1 AND vehicle_id = $2`, "maintenance_expenses"},
	RecordFuelFill: {`SELECT version, date, amount, notes FROM fuel_logs WHERE id::text = $1 AND vehicle_id = $2`, "fuel_logs"},
	RecordCharge:   {`SELECT version, date, cost, notes FROM charge_logs WHERE id::text = $1 AND vehicle_id = $2 AND is_manual = TRUE`, "charge_logs"},
}

// FindRecord reads the summary of a record of the vehicle; ErrNotFound when there is none (or, for a charge, when it is
// a synchronised one).
func (r *Repository) FindRecord(ctx context.Context, kind, vehicleID, id string) (*RecordSummary, error) {
	q, ok := recordQueries[kind]
	if !ok {
		return nil, ErrNotFound
	}
	s := RecordSummary{Kind: kind, ID: id}
	err := r.pool.QueryRow(ctx, q.selectSQL, id, vehicleID).Scan(&s.Version, &s.Date, &s.Amount, &s.Note)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteRecordAtVersion deletes a record of the vehicle if it still is at the version the caller read: ErrStaleWrite
// when it was saved since, ErrNotFound when it does not exist.
func (r *Repository) DeleteRecordAtVersion(ctx context.Context, kind, vehicleID, id string, expected int64) error {
	q, ok := recordQueries[kind]
	if !ok {
		return ErrNotFound
	}
	extra := ""
	if kind == RecordCharge {
		extra = " AND is_manual = TRUE"
	}
	tag, err := r.pool.Exec(ctx, `DELETE FROM `+q.table+` WHERE id::text = $1 AND vehicle_id = $2 AND version = $3`+extra, id, vehicleID, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	if _, err := r.FindRecord(ctx, kind, vehicleID, id); err != nil {
		return err
	}
	return ErrStaleWrite
}
