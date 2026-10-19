package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

// Caps of the trip groups view: a response never grows with the history.
const (
	integrationTripGroupsDefault = 20
	integrationTripGroupsMax     = 50
)

// IntegrationTripGroup is a journey the owner grouped several drives into.
type IntegrationTripGroup struct {
	Name           string              `json:"name" jsonschema:"the group's name, user text"`
	Start          *string             `json:"start_time" jsonschema:"RFC 3339 instant the first drive began; null when the group has no drive"`
	End            *string             `json:"end_time" jsonschema:"RFC 3339 instant the last drive ended; null when the group has no drive"`
	Drives         int                 `json:"drives" jsonschema:"number of drives in the group"`
	Distance       IntegrationDistance `json:"distance" jsonschema:"distance of its drives"`
	TravelExpenses IntegrationAmount   `json:"travel_expenses" jsonschema:"tolls, parking and ferries borne by the group: those of its drives and those attached to the group itself"`
	GroupExpenses  int                 `json:"group_expenses" jsonschema:"how many of those expenses are attached to the group rather than to one drive"`
	CarpoolTrips   int                 `json:"carpool_trips" jsonschema:"carpool trips recorded for this journey"`
	Notes          *string             `json:"notes" jsonschema:"free note on the group, user text; null when none"`
}

// IntegrationTripGroups lists the journeys of a vehicle, the most recent first.
type IntegrationTripGroups struct {
	VehicleID string                 `json:"vehicle_id" jsonschema:"the vehicle these groups belong to"`
	Groups    []IntegrationTripGroup `json:"groups" jsonschema:"the most recent trip groups"`
	Matching  int                    `json:"matching" jsonschema:"how many trip groups the vehicle has, listed or not"`
	Truncated bool                   `json:"truncated" jsonschema:"true when more groups exist than the ones listed"`
}

func buildIntegrationTripGroups(vehicleID, currency string, groups []models.TripGroup, limit int) IntegrationTripGroups {
	out := IntegrationTripGroups{VehicleID: vehicleID, Groups: []IntegrationTripGroup{}, Matching: len(groups), Truncated: len(groups) > limit}
	for i, g := range groups {
		if i >= limit {
			break
		}
		item := IntegrationTripGroup{
			Name: g.Name, Drives: len(g.DriveIDs), Distance: kilometres(g.DistanceKm), TravelExpenses: amountOf(g.TollsTotal, currency),
			GroupExpenses: g.ExpenseCount, CarpoolTrips: g.CarpoolCount, Notes: noteOf(g.Notes),
		}
		if g.StartTime != nil {
			s := g.StartTime.UTC().Format(time.RFC3339)
			item.Start = &s
		}
		if g.EndTime != nil {
			e := g.EndTime.UTC().Format(time.RFC3339)
			item.End = &e
		}
		out.Groups = append(out.Groups, item)
	}
	return out
}

// TripGroups lists the most recent journeys the owner grouped drives into.
func (h *IntegrationReadHandler) TripGroups(ctx context.Context, userID, vehicleID string, limit int) (*IntegrationTripGroups, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = integrationTripGroupsDefault
	}
	limit = min(limit, integrationTripGroupsMax)
	groups, err := h.repo.ListTripGroups(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationTripGroups(vehicle.ID, vehicle.Currency, groups, limit)
	return &out, nil
}

// GetTripGroups serves TripGroups.
func (h *IntegrationReadHandler) GetTripGroups(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			writeAPIError(w, http.StatusBadRequest, apierror.New("request.invalid_value", "Invalid value"))
			return
		}
		limit = n
	}
	out, err := h.TripGroups(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"), limit)
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to list the trip groups")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
