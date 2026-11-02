package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func TestSetChargeCostReviewAccess(t *testing.T) {
	a := newVehicleAPI(t, "ccr")
	r := chi.NewRouter()
	r.Patch("/{vehicleId}/charges/{chargeId}/cost-review", NewExpenseHandler(a.repo, nil).SetChargeCostReview)
	a.router = r

	id, zero := 7, money.Cents(0)
	if _, err := a.repo.UpsertTeslaMateCharge(context.Background(), &models.ChargeLog{
		VehicleID: a.vid, TeslaMateChargeID: &id, Date: time.Now().Add(-time.Hour), KwhAdded: 10, Cost: &zero, Currency: "EUR",
	}); err != nil {
		t.Fatal(err)
	}
	list, _, err := a.repo.ListCharges(context.Background(), a.vid, true, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("one charge to review: %v %v", list, err)
	}
	path := "/" + a.vid + "/charges/" + list[0].ID + "/cost-review"

	a.do(a.viewer, http.StatusForbidden, "PATCH", path, `{"reviewed":true}`)
	a.do(a.other, http.StatusNotFound, "PATCH", path, `{"reviewed":true}`)
	a.do(a.editor, http.StatusBadRequest, "PATCH", path, `{`)
	a.do(a.editor, http.StatusNotFound, "PATCH", "/"+a.vid+"/charges/00000000-0000-0000-0000-000000000000/cost-review", `{"reviewed":true}`)
	if n, _ := a.repo.CountChargesWithoutCost(context.Background(), a.vid); n != 1 {
		t.Fatalf("the refused calls changed nothing: %d", n)
	}

	a.do(a.editor, http.StatusOK, "PATCH", path, `{"reviewed":true}`)
	if n, _ := a.repo.CountChargesWithoutCost(context.Background(), a.vid); n != 0 {
		t.Fatalf("a confirmed 0 is no longer missing a cost: %d", n)
	}
	a.do(a.ownerID, http.StatusOK, "PATCH", path, `{"reviewed":false}`)
	if n, _ := a.repo.CountChargesWithoutCost(context.Background(), a.vid); n != 1 {
		t.Fatalf("reopening puts it back: %d", n)
	}
}
