package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// Caps of the trips view: a response never grows with the history.
const (
	integrationTripsDefault = 20
	integrationTripsMax     = 50
)

// IntegrationTripsQuery narrows the trips of a vehicle. Search matches the departure or arrival address; From and To
// are days (YYYY-MM-DD, UTC) or RFC 3339 instants, both included.
type IntegrationTripsQuery struct {
	Search string
	From   string
	To     string
	Limit  int
}

// IntegrationTrip is one drive. It carries the addresses the account recorded, so an assistant can talk about "the
// trip to Lyon", but no position and no identifier: an address stored as coordinates is reported as unknown.
type IntegrationTrip struct {
	DriveID     string               `json:"drive_id" jsonschema:"the id of the trip, to name it in create_carpool_from_drives"`
	CarpoolIDs  []string             `json:"carpool_ids" jsonschema:"the ids of the carpool trips this trip is a leg of, as get_carpool lists them; empty when it is in none"`
	StartTime   string               `json:"start_time" jsonschema:"RFC 3339 instant the trip began"`
	EndTime     string               `json:"end_time" jsonschema:"RFC 3339 instant the trip ended"`
	From        *string              `json:"from" jsonschema:"departure address as the owner wrote or looked it up, user text; null when unknown"`
	To          *string              `json:"to" jsonschema:"arrival address as the owner wrote or looked it up, user text; null when unknown"`
	Distance    *IntegrationDistance `json:"distance" jsonschema:"distance of the trip; null when it was not recorded"`
	DurationMin int                  `json:"duration_min" jsonschema:"duration in minutes"`
	EnergyKwh   *float64             `json:"energy_kwh" jsonschema:"energy consumed in kWh; null when unknown"`
	// TravelExpenses is what tolls, parking and ferries cost for this drive, in the vehicle's currency; null when none
	// is recorded. An expense shared by a trip group is split between its drives by distance.
	TravelExpenses *IntegrationAmount `json:"travel_expenses" jsonschema:"cost of tolls, parking and ferries for this trip, null when none is recorded; a cost shared by a trip group is split between its drives by distance"`
	Tags           []string           `json:"tags" jsonschema:"tags the owner put on the trip, user text"`
	Driver         *string            `json:"driver" jsonschema:"name of the person who drove, as the owner recorded them, user text; null when no driver is assigned"`
}

// IntegrationTrips lists the trips of a vehicle, the most recent first.
type IntegrationTrips struct {
	VehicleID string            `json:"vehicle_id" jsonschema:"the vehicle these trips belong to"`
	Trips     []IntegrationTrip `json:"trips" jsonschema:"the trips that fit the query, most recent first"`
	// Matching counts the trips that fit the query; Truncated says more than the ones listed exist.
	Matching  int  `json:"matching" jsonschema:"how many trips fit the query, listed or not"`
	Truncated bool `json:"truncated" jsonschema:"true when more trips fit than the ones listed"`
}

// tripBound reads a From or To bound. A day as upper bound runs to the end of that day.
func tripBound(raw string, upper bool) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	t, err := parseDate(raw, time.UTC)
	if err != nil {
		return nil, err
	}
	if upper && len(raw) == len("2006-01-02") {
		t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
	}
	return &t, nil
}

// Trips lists the drives of a vehicle that fit the query.
func (h *IntegrationReadHandler) Trips(ctx context.Context, userID, vehicleID string, q IntegrationTripsQuery) (*IntegrationTrips, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	filter := database.DriveFilter{Query: q.Search}
	if filter.From, err = tripBound(q.From, false); err != nil {
		return nil, err
	}
	if filter.To, err = tripBound(q.To, true); err != nil {
		return nil, err
	}
	limit := q.Limit
	if limit <= 0 {
		limit = integrationTripsDefault
	}
	limit = min(limit, integrationTripsMax)

	drives, total, err := h.repo.ListDrives(ctx, vehicle.ID, filter, limit, 0)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(drives))
	for i, d := range drives {
		ids[i] = d.ID
	}
	expenses, err := h.repo.GetTollExpensesForDrives(ctx, vehicle.ID, ids)
	if err != nil {
		return nil, err
	}

	carpools, err := h.repo.GetCarpoolsForDrives(ctx, vehicle.ID, ids)
	if err != nil {
		return nil, err
	}
	trips := make([]IntegrationTrip, 0, len(drives))
	for _, d := range drives {
		trip := integrationTrip(d, vehicle.Currency, expenses)
		for _, c := range carpools[d.ID] {
			trip.CarpoolIDs = append(trip.CarpoolIDs, c.ID)
		}
		trips = append(trips, trip)
	}
	return &IntegrationTrips{VehicleID: vehicle.ID, Trips: trips, Matching: total, Truncated: total > len(trips)}, nil
}

func integrationTrip(d models.Drive, currency string, expenses map[string]money.Cents) IntegrationTrip {
	trip := IntegrationTrip{
		DriveID: d.ID, CarpoolIDs: []string{}, StartTime: d.StartTime.UTC().Format(time.RFC3339), EndTime: d.EndTime.UTC().Format(time.RFC3339),
		From: publicAddress(d.StartAddress), To: publicAddress(d.EndAddress),
		DurationMin: d.DurationMin, EnergyKwh: d.EnergyConsumedKwh, Tags: d.Tags, Driver: noteOf(d.DriverName),
	}
	if trip.Tags == nil {
		trip.Tags = []string{}
	}
	if d.DistanceKm != nil {
		km := kilometres(*d.DistanceKm)
		trip.Distance = &km
	}
	if amount, ok := expenses[d.ID]; ok && amount != 0 {
		a := amountOf(amount, currency)
		trip.TravelExpenses = &a
	}
	return trip
}

// publicAddress is an address as the account wrote or looked it up, or nil when it is empty or only a position.
func publicAddress(stored *string) *string {
	if stored == nil {
		return nil
	}
	address := strings.TrimSpace(*stored)
	if address == "" || coordinateAddress.MatchString(address) {
		return nil
	}
	return &address
}

// GetTrips serves Trips.
func (h *IntegrationReadHandler) GetTrips(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	q := IntegrationTripsQuery{Search: query.Get("search"), From: query.Get("from"), To: query.Get("to")}
	if raw := query.Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil || limit < 1 {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_value", "Invalid value"))
			return
		}
		q.Limit = limit
	}
	out, err := h.Trips(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), q)
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to list the trips")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
