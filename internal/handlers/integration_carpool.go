package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// Caps of the carpool view: a response never grows with the history.
const (
	integrationCarpoolDefault = 10
	integrationCarpoolMax     = 30
)

// WithCarpool gives the carpool view the service that brings the legs made from drives up to date.
func (h *IntegrationReadHandler) WithCarpool(s *services.CarpoolService) *IntegrationReadHandler {
	h.carpool = s
	return h
}

// IntegrationCarpoolSummary is what carpooling brought in over every trip of a vehicle.
type IntegrationCarpoolSummary struct {
	Trips           int                 `json:"trips" jsonschema:"number of carpool trips"`
	Passengers      int                 `json:"passengers" jsonschema:"number of passenger bookings over those trips"`
	Distance        IntegrationDistance `json:"distance" jsonschema:"distance of the carpool trips"`
	RealCost        IntegrationAmount   `json:"real_cost" jsonschema:"what those trips cost to run (energy, tolls, tires, maintenance, insurance and others)"`
	Revenue         IntegrationAmount   `json:"revenue" jsonschema:"what the passengers paid"`
	NetCost         IntegrationAmount   `json:"net_cost" jsonschema:"real cost minus revenue: what the trips cost the owner after being repaid"`
	CoverageRatePct *float64            `json:"coverage_rate_pct" jsonschema:"share of the real cost the passengers paid back, in percent; null while there is no cost"`
	NetCostPerKm    *IntegrationAmount  `json:"net_cost_per_km" jsonschema:"net cost per km; null while there is no distance"`
	PassengersShare IntegrationAmount   `json:"passengers_fair_share" jsonschema:"what the passengers should cover given the legs they rode"`
	DriverShare     IntegrationAmount   `json:"driver_fair_share" jsonschema:"the part of the costs the driver keeps, including the empty seats"`
}

// IntegrationCarpoolPassenger is one person who rode, named as the owner recorded them.
type IntegrationCarpoolPassenger struct {
	Name      string            `json:"name" jsonschema:"the passenger's name as the owner typed it, user text"`
	Seats     int               `json:"seats" jsonschema:"seats the passenger took"`
	BoardsAt  *string           `json:"boards_at" jsonschema:"the stop where the passenger got on; null when that stop has no label"`
	AlightsAt *string           `json:"alights_at" jsonschema:"the stop where the passenger got off; null when that stop has no label"`
	Paid      IntegrationAmount `json:"paid" jsonschema:"what the passenger paid"`
	CostShare IntegrationAmount `json:"cost_share" jsonschema:"fair share of the costs of the legs the passenger rode"`
	Balance   IntegrationAmount `json:"balance" jsonschema:"paid minus cost share: positive when the passenger paid more than their share, negative when they still owe"`
	Notes     *string           `json:"notes" jsonschema:"free note on the booking, user text; null when none"`
}

// IntegrationCarpoolTrip is one shared trip with its costs, stops and passengers.
type IntegrationCarpoolTrip struct {
	ID         string                        `json:"id" jsonschema:"the id of the carpool trip; empty for a carpool that was only previewed"`
	Title      string                        `json:"title" jsonschema:"the trip's title, user text"`
	Date       string                        `json:"date" jsonschema:"RFC 3339 instant of the trip"`
	Distance   IntegrationDistance           `json:"distance" jsonschema:"distance of the trip"`
	Stops      []*string                     `json:"stops" jsonschema:"the stops of the trip in order, from departure to arrival; a stop without a label is null"`
	TotalCost  IntegrationAmount             `json:"total_cost" jsonschema:"what the trip cost to run"`
	Energy     IntegrationAmount             `json:"energy_cost" jsonschema:"energy part of the cost"`
	Tolls      IntegrationAmount             `json:"tolls_cost" jsonschema:"tolls part of the cost"`
	Revenue    IntegrationAmount             `json:"revenue" jsonschema:"what the passengers paid"`
	NetCost    IntegrationAmount             `json:"net_cost" jsonschema:"total cost minus revenue"`
	DriverCost IntegrationAmount             `json:"driver_cost_share" jsonschema:"the part of the cost the driver keeps"`
	Passengers []IntegrationCarpoolPassenger `json:"passengers" jsonschema:"who rode, named"`
	Notes      *string                       `json:"notes" jsonschema:"free note on the trip, user text; null when none"`
}

// IntegrationCarpool lists the carpool trips of a vehicle, the most recent first.
type IntegrationCarpool struct {
	VehicleID string                    `json:"vehicle_id" jsonschema:"the vehicle these trips belong to"`
	Currency  string                    `json:"currency" jsonschema:"currency of every amount"`
	Summary   IntegrationCarpoolSummary `json:"summary" jsonschema:"over every carpool trip of the vehicle, listed or not"`
	Trips     []IntegrationCarpoolTrip  `json:"trips" jsonschema:"the most recent carpool trips"`
	Truncated bool                      `json:"truncated" jsonschema:"true when more trips exist than the ones listed"`
}

// carpoolStops returns the labels of the stops of a trip: the start of each leg, then the end of the last one.
func carpoolStops(legs []models.CarpoolLeg) []*string {
	stops := []*string{}
	for i, l := range legs {
		if i == 0 {
			stops = append(stops, publicAddress(l.StartLabel))
		}
		stops = append(stops, publicAddress(l.EndLabel))
	}
	return stops
}

func stopAt(stops []*string, index int) *string {
	if index < 0 || index >= len(stops) {
		return nil
	}
	return stops[index]
}

func noteOf(n *string) *string {
	if n == nil {
		return nil
	}
	text := strings.TrimSpace(*n)
	if text == "" {
		return nil
	}
	return &text
}

// integrationCarpoolTrip is the view of a trip whose shares are computed.
func integrationCarpoolTrip(t models.CarpoolTripWithPassengers, currency string) IntegrationCarpoolTrip {
	stops := carpoolStops(t.Legs)
	item := IntegrationCarpoolTrip{
		ID: t.ID, Title: t.Title, Date: t.Date.UTC().Format(time.RFC3339), Distance: kilometres(t.DistanceKm), Stops: stops,
		TotalCost: amountOf(t.TotalCost, currency), Energy: amountOf(t.ElectricityCost, currency), Tolls: amountOf(t.TollsCost, currency),
		Revenue: amountOf(t.TotalRevenue, currency), NetCost: amountOf(t.NetCost, currency), DriverCost: amountOf(t.DriverCostShare, currency),
		Passengers: []IntegrationCarpoolPassenger{}, Notes: noteOf(t.Notes),
	}
	for _, p := range t.Passengers {
		item.Passengers = append(item.Passengers, IntegrationCarpoolPassenger{
			Name: p.PassengerName, Seats: p.Seats, BoardsAt: stopAt(stops, p.BoardStopIndex), AlightsAt: stopAt(stops, p.AlightStopIndex),
			Paid: amountOf(p.AmountPaid, currency), CostShare: amountOf(p.CostShare, currency), Balance: amountOf(p.Balance, currency), Notes: noteOf(p.Notes),
		})
	}
	return item
}

func buildIntegrationCarpool(vehicleID, currency string, trips []models.CarpoolTripWithPassengers, sum models.CarpoolSummary, limit int) IntegrationCarpool {
	// The fair shares of the summary add up the ones of every trip
	for i := range trips {
		trips[i].DriverCostShare, trips[i].PassengersCostShare = services.AllocateCarpoolCosts(trips[i].Legs, trips[i].Passengers)
		sum.TotalDriverShare += trips[i].DriverCostShare
		sum.TotalPassengersShare += trips[i].PassengersCostShare
	}
	out := IntegrationCarpool{
		VehicleID: vehicleID, Currency: currency, Trips: []IntegrationCarpoolTrip{}, Truncated: len(trips) > limit,
		Summary: IntegrationCarpoolSummary{
			Trips: sum.TotalTrips, Passengers: sum.TotalPassengers, Distance: kilometres(sum.TotalDistanceKm),
			RealCost: amountOf(sum.TotalRealCost, currency), Revenue: amountOf(sum.TotalRevenue, currency), NetCost: amountOf(sum.TotalNetCost, currency),
			CoverageRatePct: sum.CoverageRatePct, PassengersShare: amountOf(sum.TotalPassengersShare, currency), DriverShare: amountOf(sum.TotalDriverShare, currency),
		},
	}
	if sum.TotalDistanceKm > 0 {
		out.Summary.NetCostPerKm = decimalAmount(&sum.NetCostPerKm, currency, 3)
	}
	for i, t := range trips {
		if i >= limit {
			break
		}
		out.Trips = append(out.Trips, integrationCarpoolTrip(t, currency))
	}
	return out
}

// Carpool lists the most recent carpool trips of a vehicle with their stops and named passengers, and the totals.
func (h *IntegrationReadHandler) Carpool(ctx context.Context, userID, vehicleID string, limit int) (*IntegrationCarpool, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = integrationCarpoolDefault
	}
	limit = min(limit, integrationCarpoolMax)

	trips, err := h.repo.ListCarpoolTrips(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	if h.carpool != nil {
		// Before the summary, which sums the stored amounts
		if err := h.carpool.RefreshLinkedCosts(ctx, vehicle.ID, trips); err != nil {
			return nil, err
		}
	}
	sum, err := h.repo.GetCarpoolSummary(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationCarpool(vehicle.ID, vehicle.Currency, trips, *sum, limit)
	return &out, nil
}

// GetCarpool serves Carpool.
func (h *IntegrationReadHandler) GetCarpool(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_value", "Invalid value"))
			return
		}
		limit = n
	}
	out, err := h.Carpool(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), limit)
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to list the carpool trips")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
