package models

import (
	"time"

	"github.com/teslacost/teslacost/internal/money"
)

// TollInvoice is the toll total a motorway operator billed a vehicle for a period (a month today). For that period it
// replaces every individual toll of the vehicle in the monthly and global totals; trips, carpools and trip groups keep
// the tolls entered on them.
type TollInvoice struct {
	ID          string      `json:"id"`
	VehicleID   string      `json:"vehicle_id"`
	Month       string      `json:"month"` // YYYY-MM, in the reporting timezone
	PeriodStart time.Time   `json:"period_start"`
	PeriodEnd   time.Time   `json:"period_end"` // exclusive
	Total       money.Cents `json:"total"`      // in the vehicle's currency
	Note        *string     `json:"note"`
	Version     int64       `json:"version"`
	UpdatedAt   time.Time   `json:"updated_at"`
	// TripTolls is what the individual tolls of the period add up to, and Gap what the invoice says beyond them
	// (Total - TripTolls: positive when the invoice is higher, negative when the trips over-count).
	TripTolls money.Cents `json:"trip_tolls"`
	Gap       money.Cents `json:"gap"`
}
