package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

func TestHomeAssistantChargingSessions(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	newUser := func(email string) *models.User {
		u, err := repo.CreateUser(ctx, email, "hash")
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	newVehicle := func(u *models.User, name, powertrain string) *models.Vehicle {
		v := &models.Vehicle{Currency: "EUR", UserID: u.ID, Name: name, Powertrain: powertrain, TeslaMateAuthType: models.AuthModeNone}
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	h := NewHomeAssistantHandler(repo, services.NewTariffService())
	type response struct {
		Status    string `json:"status"`
		Code      string `json:"code"`
		VehicleID string `json:"vehicle_id"`
		ChargeID  string `json:"charge_id"`
		PendingID string `json:"pending_id"`
	}
	post := func(u *models.User, body string) (int, response) {
		req := httptest.NewRequest(http.MethodPost, "/api/integrations/homeassistant/event", bytes.NewBufferString(body))
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, u.ID))
		rec := httptest.NewRecorder()
		h.HandleEvent(rec, req)
		var resp response
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return rec.Code, resp
	}
	charges := func(v *models.Vehicle) []models.ChargeLog {
		list, _, err := repo.ListCharges(ctx, v.ID, false, 50, 0)
		if err != nil {
			t.Fatal(err)
		}
		return list
	}

	// One electric vehicle of its own, a combustion one, and an electric vehicle shared with it read-only
	owner := newUser("ha-charging@example.com")
	ev := newVehicle(owner, "EV", models.PowertrainEV)
	ice := newVehicle(owner, "ICE", models.PowertrainICE)
	friend := newUser("ha-friend@example.com")
	shared := newVehicle(friend, "Friend's EV", models.PowertrainEV)
	if _, err := repo.AddVehicleMember(ctx, shared.ID, owner.Email, models.RoleViewer, nil); err != nil {
		t.Fatal(err)
	}

	session := `{"event_id":"wallbox:2026-10-01T01:00:00Z","event_type":"charging_session_end",
		"data":{"start_time":"2026-10-01T01:00:00Z","end_time":"2026-10-01T04:00:00Z","energy_added_kwh":21.5,"soc_start":35,"soc_end":80}}`
	status, resp := post(owner, session)
	if status != http.StatusCreated || resp.VehicleID != ev.ID {
		t.Fatalf("session without vehicle: got %d %+v, want 201 on the only electric vehicle it can edit", status, resp)
	}
	got := charges(ev)
	if len(got) != 1 {
		t.Fatalf("charges recorded: %d, want 1", len(got))
	}
	c := got[0]
	if c.CostSource != "HOMEASSISTANT" || c.Notes != nil || c.Address != nil {
		t.Errorf("stored charge: source %q, notes %v, address %v", c.CostSource, c.Notes, c.Address)
	}
	// The battery levels are read by the energy statistics, not returned with the charge
	var socStart, socEnd *int
	if err := repo.Pool().QueryRow(ctx, `SELECT start_battery_level, end_battery_level FROM charge_logs WHERE id = $1`, c.ID).Scan(&socStart, &socEnd); err != nil {
		t.Fatal(err)
	}
	if socStart == nil || *socStart != 35 || socEnd == nil || *socEnd != 80 {
		t.Errorf("stored battery levels: %v -> %v, want 35 -> 80", socStart, socEnd)
	}

	if status, resp := post(owner, session); status != http.StatusOK || resp.Status != "duplicate" || resp.ChargeID != c.ID {
		t.Errorf("same event resent: got %d %+v, want 200 duplicate of %s", status, resp, c.ID)
	}
	resentWithoutID := `{"data":{"start_time":"2026-10-01T01:10:00Z","end_time":"2026-10-01T04:00:00Z","energy_kwh":21.3}}`
	if status, resp := post(owner, resentWithoutID); status != http.StatusOK || resp.Status != "duplicate" {
		t.Errorf("same session without event_id: got %d %+v, want 200 duplicate", status, resp)
	}
	if n := len(charges(ev)); n != 1 {
		t.Errorf("charges after the resends: %d, want 1", n)
	}

	for _, c := range []struct{ name, body, code string }{
		{"no energy", `{"data":{"start_time":"2026-10-02T01:00:00Z","end_time":"2026-10-02T02:00:00Z","energy_kwh":0}}`, "charge.invalid_energy"},
		{"cumulative meter value", `{"data":{"start_time":"2026-10-02T01:00:00Z","end_time":"2026-10-02T02:00:00Z","energy_kwh":5123}}`, "charge.invalid_energy"},
		{"start after end", `{"data":{"start_time":"2026-10-02T03:00:00Z","end_time":"2026-10-02T02:00:00Z","energy_kwh":10}}`, "charge.invalid_period"},
		{"combustion vehicle", `{"vehicle_id":"` + ice.ID + `","data":{"start_time":"2026-10-02T01:00:00Z","end_time":"2026-10-02T02:00:00Z","energy_kwh":10}}`, "charge.electric_only"},
	} {
		if status, resp := post(owner, c.body); status != http.StatusBadRequest || resp.Code != c.code {
			t.Errorf("%s: got %d %q, want 400 %s", c.name, status, resp.Code, c.code)
		}
	}

	// Two electric vehicles and no home default: the session waits for its vehicle, once
	household := newUser("ha-household@example.com")
	newVehicle(household, "EV 1", models.PowertrainEV)
	newVehicle(household, "EV 2", models.PowertrainEV)
	shared2 := `{"event_id":"wallbox:2026-10-03T22:00:00Z","data":{"start_time":"2026-10-03T22:00:00Z","end_time":"2026-10-04T02:00:00Z","energy_kwh":30}}`
	status, resp = post(household, shared2)
	if status != http.StatusCreated || resp.Status != "pending_qualification" {
		t.Fatalf("shared charger: got %d %+v, want 201 pending_qualification", status, resp)
	}
	if status, again := post(household, shared2); status != http.StatusOK || again.Status != "duplicate" || again.PendingID != resp.PendingID {
		t.Errorf("pending session resent: got %d %+v, want 200 duplicate of %s", status, again, resp.PendingID)
	}
	pending, err := repo.ListPendingCharges(ctx, household.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Location != nil {
		t.Errorf("pending charges: %+v, want one without location", pending)
	}
}

// A grant limited to some vehicles must not route a session without a vehicle to one outside it, nor answer that a
// session is a duplicate of a charge recorded on one outside it.
func TestHomeAssistantChargingSessionStaysWithinTheVehiclesOfALimitedGrant(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "ha-limited@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	newEV := func(name string) *models.Vehicle {
		v := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: name, Powertrain: models.PowertrainEV, TeslaMateAuthType: models.AuthModeNone}
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	granted, other := newEV("Granted"), newEV("Not granted")
	h := NewHomeAssistantHandler(repo, services.NewTariffService())
	post := func(vehicles []string, body string) (int, map[string]any) {
		req := httptest.NewRequest(http.MethodPost, "/api/integrations/homeassistant/event", bytes.NewBufferString(body))
		reqCtx := context.WithValue(req.Context(), middleware.UserIDKey, owner.ID)
		reqCtx = middleware.WithVehicleScope(reqCtx, vehicles)
		rec := httptest.NewRecorder()
		h.HandleEvent(rec, req.WithContext(reqCtx))
		var out map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
		return rec.Code, out
	}
	session := `{"event_id":"wallbox:limited","event_type":"charging_session_end","data":{"start_time":"2026-10-01T01:00:00Z","end_time":"2026-10-01T04:00:00Z","energy_added_kwh":21.5}}`

	// Two electric vehicles and no vehicle named: the account alone cannot tell, so the session waits; a grant limited
	// to one of them can, and records it there
	if status, resp := post(nil, session); status != http.StatusCreated || resp["pending_id"] == nil {
		t.Fatalf("unlimited, two electric vehicles: %d %v, want the session pending", status, resp)
	}
	pending, err := repo.ListPendingCharges(ctx, owner.ID)
	if err != nil || len(pending) != 1 {
		t.Fatalf("pending charges: %v %v", pending, err)
	}
	if err := repo.DeletePendingCharge(ctx, pending[0].ID, owner.ID); err != nil {
		t.Fatal(err)
	}
	status, resp := post([]string{granted.ID}, session)
	if status != http.StatusCreated || resp["vehicle_id"] != granted.ID {
		t.Fatalf("limited to one vehicle: %d %v, want it recorded on %s", status, resp, granted.ID)
	}

	// The same session sent by a grant that only sees the other vehicle is not reported as a duplicate of a charge it
	// cannot see
	if status, resp := post([]string{other.ID}, session); status != http.StatusCreated || resp["vehicle_id"] != other.ID {
		t.Fatalf("limited to the other vehicle: %d %v, want a new charge on %s", status, resp, other.ID)
	}
	if status, _ := post([]string{other.ID}, `{"vehicle_id":"`+granted.ID+`","event_id":"x","event_type":"charging_session_end","data":{"start_time":"2026-10-02T01:00:00Z","end_time":"2026-10-02T04:00:00Z","energy_added_kwh":10}}`); status != http.StatusNotFound {
		t.Errorf("naming a vehicle outside the grant: %d, want 404", status)
	}
}
