package handlers

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
)

// IntegrationSource is what one origin wrote for a vehicle.
type IntegrationSource struct {
	Origin           string  `json:"origin" jsonschema:"teslamate (the synchronization), webhook (Home Assistant or a script), csv (an import) or manual (typed in the app)"`
	Drives           int     `json:"drives" jsonschema:"drives it wrote"`
	Charges          int     `json:"charges" jsonschema:"charging sessions it wrote"`
	OdometerReadings int     `json:"odometer_readings" jsonschema:"odometer readings it wrote"`
	LastRecord       *string `json:"last_record" jsonschema:"RFC 3339 instant of its most recent record; null when it wrote none"`
}

// IntegrationSyncResource is the synchronization of one kind of record from TeslaMate.
type IntegrationSyncResource struct {
	Resource        string  `json:"resource" jsonschema:"drives or charges"`
	HistoryImported *string `json:"history_imported_at" jsonschema:"RFC 3339 instant the complete history was imported; null while it has not been"`
	LastSuccess     *string `json:"last_success_at" jsonschema:"RFC 3339 instant of the last successful synchronization; null when none has succeeded"`
}

// IntegrationDataSources says where a vehicle's data comes from and how fresh it is.
type IntegrationDataSources struct {
	VehicleID      string                    `json:"vehicle_id" jsonschema:"the vehicle these figures belong to"`
	TeslaMate      bool                      `json:"teslamate_connected" jsonschema:"true when the vehicle is connected to a TeslaMate instance; only the owner of the vehicle sees it as true"`
	Sources        []IntegrationSource       `json:"sources" jsonschema:"what each origin wrote; an origin that wrote nothing is absent"`
	Sync           []IntegrationSyncResource `json:"sync" jsonschema:"the synchronization state of each resource read from TeslaMate; empty when no synchronization ran"`
	PendingCharges int                       `json:"pending_charges" jsonschema:"charging sessions pushed to the account that no vehicle claimed yet; they are in no vehicle's figures until the owner assigns them (all vehicles of the account together)"`
}

func timeText(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}

func buildIntegrationSources(vehicle *models.Vehicle, activity []database.SourceActivity, states []database.SyncState, pending int) IntegrationDataSources {
	out := IntegrationDataSources{
		VehicleID: vehicle.ID, PendingCharges: pending, Sources: []IntegrationSource{}, Sync: []IntegrationSyncResource{},
		TeslaMate: vehicle.Role == models.RoleOwner && vehicle.TeslaMateAPIURL != nil && strings.TrimSpace(*vehicle.TeslaMateAPIURL) != "",
	}
	for _, a := range activity {
		out.Sources = append(out.Sources, IntegrationSource{
			Origin: strings.ToLower(a.Origin), Drives: a.Drives, Charges: a.Charges, OdometerReadings: a.OdometerReadings, LastRecord: timeText(a.LastAt),
		})
	}
	for _, s := range states {
		out.Sync = append(out.Sync, IntegrationSyncResource{Resource: s.Resource, HistoryImported: timeText(s.FullImportCompletedAt), LastSuccess: timeText(s.LastSuccessAt)})
	}
	return out
}

// DataSources says where a vehicle's records come from, when they were last synchronized and how many pushed
// charges wait for a vehicle.
func (h *IntegrationReadHandler) DataSources(ctx context.Context, userID, vehicleID string) (*IntegrationDataSources, error) {
	vehicle, err := h.vehicleOf(ctx, userID, vehicleID)
	if err != nil {
		return nil, err
	}
	activity, err := h.repo.ListSourceActivity(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	states, err := h.repo.ListSyncStates(ctx, vehicle.ID)
	if err != nil {
		return nil, err
	}
	pending, err := h.repo.ListPendingCharges(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := buildIntegrationSources(vehicle, activity, states, len(pending))
	return &out, nil
}

// GetDataSources serves DataSources.
func (h *IntegrationReadHandler) GetDataSources(w http.ResponseWriter, r *http.Request) {
	out, err := h.DataSources(r.Context(), middleware.GetUserID(r.Context()), chi.URLParam(r, "vehicleId"))
	if err != nil {
		writeIntegrationError(w, r, err, "Failed to read the data sources")
		return
	}
	writeJSON(w, http.StatusOK, out)
}
