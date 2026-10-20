package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/services"
)

type CarpoolHandler struct {
	repo           *database.Repository
	carpoolService *services.CarpoolService
}

func NewCarpoolHandler(repo *database.Repository, carpoolService *services.CarpoolService) *CarpoolHandler {
	return &CarpoolHandler{
		repo:           repo,
		carpoolService: carpoolService,
	}
}

type PassengerPayload struct {
	PassengerName   string      `json:"passenger_name"`
	Origin          *string     `json:"origin"`
	Destination     *string     `json:"destination"`
	Seats           int         `json:"seats"`
	AmountPaid      money.Cents `json:"amount_paid"`
	Notes           *string     `json:"notes"`
	BoardStopIndex  *int        `json:"board_stop_index"`  // Default: first stop
	AlightStopIndex *int        `json:"alight_stop_index"` // Default: last stop
}

type LegPayload struct {
	DriveID         *string     `json:"drive_id"`
	StartLabel      *string     `json:"start_label"`
	EndLabel        *string     `json:"end_label"`
	DistanceKm      float64     `json:"distance_km"`
	ElectricityCost money.Cents `json:"electricity_cost"`
	TollsCost       money.Cents `json:"tolls_cost"`
	TiresCost       money.Cents `json:"tires_cost"`
	MaintenanceCost money.Cents `json:"maintenance_cost"`
	InsuranceCost   money.Cents `json:"insurance_cost"`
	OtherCost       money.Cents `json:"other_cost"`
}

// UpsertCarpoolRequest describes a carpool trip. Legs are the ordered stages of the trip; when omitted,
// the trip-level distance and costs form a single leg.
type UpsertCarpoolRequest struct {
	Title           string             `json:"title"`
	Date            string             `json:"date"`
	DistanceKm      float64            `json:"distance_km"`
	DriveID         *string            `json:"drive_id"`
	TripGroupID     *string            `json:"trip_group_id"`
	ElectricityCost money.Cents        `json:"electricity_cost"`
	TollsCost       money.Cents        `json:"tolls_cost"`
	TiresCost       money.Cents        `json:"tires_cost"`
	MaintenanceCost money.Cents        `json:"maintenance_cost"`
	InsuranceCost   money.Cents        `json:"insurance_cost"`
	OtherCost       money.Cents        `json:"other_cost"`
	Notes           *string            `json:"notes"`
	Legs            []LegPayload       `json:"legs"`
	Passengers      []PassengerPayload `json:"passengers"`
}

// Maximum seats occupied by passengers on a single leg (7-seat vehicles).
const maxCarpoolSeatsPerLeg = 7

func nonEmptyLabel(s *string) *string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	v := strings.TrimSpace(*s)
	if r := []rune(v); len(r) > 150 {
		v = string(r[:150])
	}
	return &v
}

// buildCarpool validates the payload and returns the trip, its legs and its passengers.
func buildCarpool(vehicleID string, req *UpsertCarpoolRequest) (*models.CarpoolTrip, []models.CarpoolLeg, []models.CarpoolPassenger, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, nil, nil, apierror.New("carpool.title_required", "The carpool title is required")
	}
	date := time.Now().UTC()
	if req.Date != "" {
		parsed, err := parseDate(req.Date)
		if err != nil {
			return nil, nil, nil, err
		}
		date = parsed
	}

	payloads := req.Legs
	if len(payloads) == 0 {
		payloads = []LegPayload{{
			DriveID: req.DriveID, DistanceKm: req.DistanceKm,
			ElectricityCost: req.ElectricityCost, TollsCost: req.TollsCost, TiresCost: req.TiresCost,
			MaintenanceCost: req.MaintenanceCost, InsuranceCost: req.InsuranceCost, OtherCost: req.OtherCost,
		}}
	}
	if len(payloads) > 30 {
		return nil, nil, nil, apierror.New("carpool.too_many_legs", "A carpool is limited to 30 legs")
	}

	legs := make([]models.CarpoolLeg, len(payloads))
	for i, lp := range payloads {
		if err := validateQuantity(lp.DistanceKm, 5000); err != nil {
			return nil, nil, nil, apierror.Newf("carpool.leg_distance", "Invalid distance for leg %d", i+1)
		}
		for _, c := range []money.Cents{lp.ElectricityCost, lp.TollsCost, lp.TiresCost, lp.MaintenanceCost, lp.InsuranceCost, lp.OtherCost} {
			if err := validateAmount(c, true); err != nil {
				return nil, nil, nil, apierror.Newf("carpool.leg_error", "Leg %d: %w", i+1, err)
			}
		}
		driveID := lp.DriveID
		if driveID != nil && *driveID == "" {
			driveID = nil
		}
		legs[i] = models.CarpoolLeg{
			OrderIndex: i, DriveID: driveID, StartLabel: nonEmptyLabel(lp.StartLabel), EndLabel: nonEmptyLabel(lp.EndLabel),
			DistanceKm: lp.DistanceKm, ElectricityCost: lp.ElectricityCost, TollsCost: lp.TollsCost, TiresCost: lp.TiresCost,
			MaintenanceCost: lp.MaintenanceCost, InsuranceCost: lp.InsuranceCost, OtherCost: lp.OtherCost,
		}
	}
	stopLabel := func(stop int) *string {
		if stop < len(legs) {
			return legs[stop].StartLabel
		}
		return legs[len(legs)-1].EndLabel
	}

	passengers := make([]models.CarpoolPassenger, 0, len(req.Passengers))
	seatsPerLeg := make([]int, len(legs))
	for i, p := range req.Passengers {
		if err := validateAmount(p.AmountPaid, true); err != nil {
			return nil, nil, nil, err
		}
		seats := p.Seats
		if seats <= 0 {
			seats = 1
		}
		// A blank name stays blank: the label shown for it ("Passenger 2") belongs to the interface language
		name := strings.TrimSpace(p.PassengerName)
		who := name
		if who == "" {
			who = fmt.Sprintf("#%d", i+1)
		}
		board, alight := 0, len(legs)
		if p.BoardStopIndex != nil {
			board = *p.BoardStopIndex
		}
		if p.AlightStopIndex != nil {
			alight = *p.AlightStopIndex
		}
		if board < 0 || alight <= board || alight > len(legs) {
			return nil, nil, nil, apierror.Newf("carpool.alight_after_board", "%s: the drop-off stop must come after the pick-up stop", who)
		}
		for i := board; i < alight; i++ {
			seatsPerLeg[i] += seats
			if seatsPerLeg[i] > maxCarpoolSeatsPerLeg {
				return nil, nil, nil, apierror.Newf("carpool.too_many_seats", "More than %d seats taken on leg %d", maxCarpoolSeatsPerLeg, i+1)
			}
		}
		origin, destination := nonEmptyLabel(p.Origin), nonEmptyLabel(p.Destination)
		if origin == nil {
			origin = stopLabel(board)
		}
		if destination == nil {
			destination = stopLabel(alight)
		}
		passengers = append(passengers, models.CarpoolPassenger{
			PassengerName: name, Origin: origin, Destination: destination, Seats: seats,
			AmountPaid: p.AmountPaid, Notes: p.Notes, BoardStopIndex: board, AlightStopIndex: alight,
		})
	}

	trip := &models.CarpoolTrip{
		VehicleID: vehicleID, DriveID: req.DriveID, TripGroupID: req.TripGroupID,
		Title: strings.TrimSpace(req.Title), Date: date, Notes: req.Notes,
	}
	if len(legs) == 1 && legs[0].DriveID != nil {
		trip.DriveID = legs[0].DriveID
	}
	return trip, legs, passengers, nil
}

// allocate computes the fair split of the costs of a trip between its driver and passengers.
func allocate(t *models.CarpoolTripWithPassengers) {
	t.DriverCostShare, t.PassengersCostShare = services.AllocateCarpoolCosts(t.Legs, t.Passengers)
}

func (h *CarpoolHandler) List(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer); v == nil {
		return
	}

	trips, err := h.repo.ListCarpoolTrips(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to list carpool trips")
		return
	}

	summary, err := h.repo.GetCarpoolSummary(r.Context(), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to summarize carpool trips")
		return
	}
	for i := range trips {
		allocate(&trips[i])
		summary.TotalDriverShare += trips[i].DriverCostShare
		summary.TotalPassengersShare += trips[i].PassengersCostShare
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"trips":   trips,
		"summary": summary,
	})
}

func (h *CarpoolHandler) Get(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer); v == nil {
		return
	}

	trip, err := h.repo.GetCarpoolTrip(r.Context(), chi.URLParam(r, "id"), vehicleID)
	if err != nil {
		writeRepoError(w, r, err, "Failed to load carpool trip")
		return
	}
	allocate(trip)
	writeJSON(w, http.StatusOK, trip)
}

func (h *CarpoolHandler) save(w http.ResponseWriter, r *http.Request, tripID string) {
	vehicleID := chi.URLParam(r, "vehicleId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	var req UpsertCarpoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
		return
	}
	trip, legs, passengers, err := buildCarpool(vehicleID, &req)
	if err != nil {
		writeErr(w, http.StatusBadRequest, err)
		return
	}

	status := http.StatusCreated
	if tripID == "" {
		err = h.repo.CreateCarpoolTrip(r.Context(), trip, legs, passengers)
	} else {
		trip.ID = tripID
		status = http.StatusOK
		err = h.repo.UpdateCarpoolTrip(r.Context(), trip, legs, passengers)
	}
	if err != nil {
		writeRepoError(w, r, err, "Failed to save carpool trip")
		return
	}

	result := &models.CarpoolTripWithPassengers{CarpoolTrip: *trip, Legs: legs, Passengers: passengers}
	allocate(result)
	writeJSON(w, status, result)
}

func (h *CarpoolHandler) Create(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, "")
}

func (h *CarpoolHandler) Update(w http.ResponseWriter, r *http.Request) {
	h.save(w, r, chi.URLParam(r, "id"))
}

func (h *CarpoolHandler) Delete(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")
	id := chi.URLParam(r, "id")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	if err := h.repo.DeleteCarpoolTrip(r.Context(), id, vehicleID); err != nil {
		writeRepoError(w, r, err, "Failed to delete carpool trip")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

func (h *CarpoolHandler) Estimate(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleViewer); v == nil {
		return
	}

	driveIDParam := r.URL.Query().Get("drive_id")
	var driveID *string
	if driveIDParam != "" {
		driveID = &driveIDParam
	}

	tripGroupIDParam := r.URL.Query().Get("trip_group_id")
	var tripGroupID *string
	if tripGroupIDParam != "" {
		tripGroupID = &tripGroupIDParam
	}

	var driveIDs []string
	driveIDsParam := r.URL.Query().Get("drive_ids")
	if driveIDsParam != "" {
		driveIDs = strings.Split(driveIDsParam, ",")
	}

	distanceKm, _ := strconv.ParseFloat(r.URL.Query().Get("distance_km"), 64)

	estimate, err := h.carpoolService.EstimateCosts(r.Context(), vehicleID, driveID, tripGroupID, driveIDs, distanceKm)
	if err != nil {
		writeRepoError(w, r, err, "Failed to estimate costs")
		return
	}

	writeJSON(w, http.StatusOK, estimate)
}

type RecalculateCarpoolsRequest struct {
	TripIDs []string `json:"trip_ids"`
}

func (h *CarpoolHandler) Recalculate(w http.ResponseWriter, r *http.Request) {
	vehicleID := chi.URLParam(r, "vehicleId")

	if v := requireVehicleAccess(w, r, h.repo, vehicleID, models.RoleEditor); v == nil {
		return
	}

	var req RecalculateCarpoolsRequest
	if r.Body != nil && r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_body", "Invalid request body"))
			return
		}
	}

	trips, err := h.carpoolService.RecalculateTrips(r.Context(), vehicleID, req.TripIDs)
	if err != nil {
		writeRepoError(w, r, err, "Failed to recalculate carpool trips")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"updated_count": len(trips),
		"trips":         trips,
	})
}
