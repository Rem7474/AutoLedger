package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/middleware"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// A charge, a travel expense and a maintenance expense can each be read by its id, by anyone who can see the vehicle
// and by nobody else; an id of another vehicle, or one that is not an id, is simply not found.
func TestExpensesAreReadByIDWithinTheirVehicle(t *testing.T) {
	repo := authTestRepo(t)
	ctx := context.Background()
	mk := func(name string) *models.User {
		u, err := repo.CreateUser(ctx, name+"-get@example.com", "hash")
		if err != nil {
			t.Fatal(err)
		}
		return u
	}
	owner, viewer, stranger := mk("owner"), mk("viewer"), mk("stranger")
	vehicle := func(name string) *models.Vehicle {
		v := &models.Vehicle{Currency: "EUR", UserID: owner.ID, Name: name, Powertrain: models.PowertrainEV, TeslaMateAuthType: models.AuthModeNone}
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	car, other := vehicle("Car"), vehicle("Other")
	if _, err := repo.AddVehicleMember(ctx, car.ID, viewer.Email, models.RoleViewer, nil); err != nil {
		t.Fatal(err)
	}

	day := time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC)
	address, cost := "Home", money.Cents(980)
	charge := &models.ChargeLog{VehicleID: car.ID, Date: day, Address: &address, KwhAdded: 37.5, Cost: &cost, Currency: "EUR"}
	if err := repo.CreateManualCharge(ctx, charge); err != nil {
		t.Fatal(err)
	}
	toll := &models.DriveExpense{VehicleID: car.ID, Type: "TOLL", Amount: 1250, Currency: "EUR", Date: day}
	if err := repo.SaveDriveExpense(ctx, toll, nil, ""); err != nil {
		t.Fatal(err)
	}
	var maintenanceID string
	if err := repo.Pool().QueryRow(ctx, `INSERT INTO maintenance_expenses (vehicle_id, category, amount, date, description) VALUES ($1, 'MAINTENANCE', 80, $2, 'Brakes') RETURNING id::text`, car.ID, day).Scan(&maintenanceID); err != nil {
		t.Fatal(err)
	}

	expense := NewExpenseHandler(repo, nil)
	router := chi.NewRouter()
	router.Get("/vehicles/{vehicleId}/charges/{chargeId}", expense.GetCharge)
	router.Get("/vehicles/{vehicleId}/expenses/{expenseId}", expense.GetDriveExpense)
	router.Get("/vehicles/{vehicleId}/maintenance/{maintenanceId}", expense.GetMaintenance)
	call := func(userID, path string) (int, map[string]any) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(context.WithValue(req.Context(), middleware.UserIDKey, userID))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		var body map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		return rec.Code, body
	}

	routes := map[string]struct{ id, field, want string }{
		"charges":     {charge.ID, "address", "Home"},
		"expenses":    {toll.ID, "type", "TOLL"},
		"maintenance": {maintenanceID, "description", "Brakes"},
	}
	for kind, r := range routes {
		path := func(vehicleID, id string) string { return "/vehicles/" + vehicleID + "/" + kind + "/" + id }
		for _, user := range []*models.User{owner, viewer} {
			status, body := call(user.ID, path(car.ID, r.id))
			if status != http.StatusOK || body["id"] != r.id || body[r.field] != r.want {
				t.Errorf("%s as %s: %d %v", kind, user.Email, status, body)
			}
		}
		if status, _ := call(stranger.ID, path(car.ID, r.id)); status != http.StatusNotFound {
			t.Errorf("%s: another user's vehicle must answer 404, got %d", kind, status)
		}
		if status, _ := call(owner.ID, path(other.ID, r.id)); status != http.StatusNotFound {
			t.Errorf("%s: the id of another vehicle must answer 404, got %d", kind, status)
		}
		if status, _ := call(owner.ID, path(car.ID, "not-an-id")); status != http.StatusNotFound {
			t.Errorf("%s: an id that is not one must answer 404, got %d", kind, status)
		}
	}
}
