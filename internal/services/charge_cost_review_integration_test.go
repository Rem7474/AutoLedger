package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// TeslaMate reports a cost it does not know as 0, like a free charge: only the owner can tell them apart.
func TestIntegrationChargeCostReview(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "cost-review@example.com")
	other := mustVehicle(t, repo, "cost-review-other@example.com")

	day := time.Now().AddDate(0, 0, -3)
	zero, paid := money.Cents(0), money.Cents(500)
	sync := func(id int, hoursAgo int, kwh float64, cost *money.Cents) *models.ChargeLog {
		c := &models.ChargeLog{VehicleID: v.ID, TeslaMateChargeID: &id, Date: day.Add(-time.Duration(hoursAgo) * time.Hour), KwhAdded: kwh, Cost: cost, Currency: "EUR"}
		if _, err := repo.UpsertTeslaMateCharge(ctx, c); err != nil {
			t.Fatal(err)
		}
		return c
	}
	sync(1, 1, 10, &zero) // unknown to TeslaMate, reported as 0
	sync(2, 2, 20, nil)   // no cost at all
	sync(3, 3, 30, &paid) // priced
	manual := &models.ChargeLog{VehicleID: v.ID, Date: day.Add(-4 * time.Hour), KwhAdded: 5, Cost: &zero, Currency: "EUR"}
	if err := repo.CreateManualCharge(ctx, manual); err != nil { // a 0 typed by hand is deliberate
		t.Fatal(err)
	}

	toReview := func() map[int]bool {
		list, _, err := repo.ListCharges(ctx, v.ID, true, 50, 0)
		if err != nil {
			t.Fatal(err)
		}
		got := map[int]bool{}
		for _, c := range list {
			if !c.CostToReview {
				t.Errorf("a charge listed as to review carries cost_to_review=false: %+v", c)
			}
			if c.TeslaMateChargeID != nil {
				got[*c.TeslaMateChargeID] = true
			}
		}
		if n, err := repo.CountChargesWithoutCost(ctx, v.ID); err != nil || n != len(list) {
			t.Fatalf("the count must follow the filter: %d vs %d (%v)", n, len(list), err)
		}
		return got
	}
	if got := toReview(); len(got) != 2 || !got[1] || !got[2] {
		t.Fatalf("the 0 and the missing cost of synchronised charges are to review, nothing else: %v", got)
	}

	tco := NewTCOService(db.Pool, "UTC")
	sum, err := tco.ComputeVehicleTCO(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Completeness.ChargesWithoutCost != 2 || sum.Completeness.KwhWithoutCost != 30 {
		t.Fatalf("gaps: %+v", sum.Completeness)
	}

	list, _, err := repo.ListCharges(ctx, v.ID, false, 50, 0)
	if err != nil {
		t.Fatal(err)
	}
	var zeroID string
	for _, c := range list {
		if c.TeslaMateChargeID != nil && *c.TeslaMateChargeID == 1 {
			zeroID = c.ID
		}
	}

	if err := repo.SetChargeCostReviewed(ctx, zeroID, other.ID, true); !errors.Is(err, database.ErrNotFound) {
		t.Fatalf("another vehicle's charge cannot be reviewed: %v", err)
	}
	if err := repo.SetChargeCostReviewed(ctx, zeroID, v.ID, true); err != nil {
		t.Fatal(err)
	}
	if got := toReview(); len(got) != 1 || !got[2] {
		t.Fatalf("a confirmed 0 leaves the queue: %v", got)
	}
	sync(1, 1, 10, &zero) // the next synchronisation keeps the confirmation
	if got := toReview(); len(got) != 1 {
		t.Fatalf("a synchronisation does not reopen a confirmed 0: %v", got)
	}
	if sum, err = tco.ComputeVehicleTCO(ctx, v.ID); err != nil || sum.Completeness.ChargesWithoutCost != 1 {
		t.Fatalf("gaps after the confirmation: %+v %v", sum.Completeness, err)
	}

	if err := repo.SetChargeCostReviewed(ctx, zeroID, v.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := toReview(); len(got) != 2 {
		t.Fatalf("reopening puts the charge back: %v", got)
	}
}

// The cost of a synchronised charge is set in TeslaMate: saving the notes of the charge does not freeze it against the
// synchronisation, only changing the cost here does.
func TestIntegrationSavingNotesKeepsTheSynchronisedCost(t *testing.T) {
	_, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "cost-notes@example.com")
	zero, paid := money.Cents(0), money.Cents(700)
	id := 11
	c := &models.ChargeLog{VehicleID: v.ID, TeslaMateChargeID: &id, Date: time.Now().AddDate(0, 0, -1), KwhAdded: 10, Cost: &zero, Currency: "EUR"}
	if _, err := repo.UpsertTeslaMateCharge(ctx, c); err != nil {
		t.Fatal(err)
	}
	list, _, err := repo.ListCharges(ctx, v.ID, false, 10, 0)
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %d", err, len(list))
	}
	edit := list[0]
	note := "plugged at the office"
	edit.Notes = &note
	if err := repo.UpdateCharge(ctx, &edit); err != nil {
		t.Fatal(err)
	}
	// TeslaMate now reports the cost typed there
	if _, err := repo.UpsertTeslaMateCharge(ctx, &models.ChargeLog{VehicleID: v.ID, TeslaMateChargeID: &id, Date: c.Date, KwhAdded: 10, Cost: &paid, Currency: "EUR"}); err != nil {
		t.Fatal(err)
	}
	list, _, _ = repo.ListCharges(ctx, v.ID, false, 10, 0)
	if list[0].CostSource != "TESLAMATE" || list[0].Cost == nil || *list[0].Cost != paid {
		t.Fatalf("the cost typed in TeslaMate is read again after the notes were saved: %+v", list[0])
	}
	if list[0].Notes == nil || *list[0].Notes != note {
		t.Fatalf("the notes are kept: %+v", list[0].Notes)
	}

	// Changing the cost here is still a correction the synchronisation respects
	fix := list[0]
	own := money.Cents(900)
	fix.Cost = &own
	if err := repo.UpdateCharge(ctx, &fix); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpsertTeslaMateCharge(ctx, &models.ChargeLog{VehicleID: v.ID, TeslaMateChargeID: &id, Date: c.Date, KwhAdded: 10, Cost: &zero, Currency: "EUR"}); err != nil {
		t.Fatal(err)
	}
	list, _, _ = repo.ListCharges(ctx, v.ID, false, 10, 0)
	if list[0].CostSource != "MANUAL" || list[0].Cost == nil || *list[0].Cost != own {
		t.Fatalf("a cost changed here survives the synchronisation: %+v", list[0])
	}
}

func TestIntegrationPlaceStatsSeparateConfirmedFreeFromUnknown(t *testing.T) {
	db, repo := setupIntegrationDB(t, false)
	ctx := context.Background()
	v := mustVehicle(t, repo, "cost-review-places@example.com")
	pub, err := repo.CreateChargingPlace(ctx, v.UserID, "Public", models.PlacePublic)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SetChargingPlaceAlias(ctx, v.UserID, "Station", &pub.ID); err != nil {
		t.Fatal(err)
	}
	addr, zero := "Station", money.Cents(0)
	day := time.Date(time.Now().Year(), time.Now().Month(), 2, 8, 0, 0, 0, time.UTC)
	ids := map[int]string{}
	for i := 1; i <= 2; i++ {
		id := i
		c := &models.ChargeLog{VehicleID: v.ID, TeslaMateChargeID: &id, Date: day.Add(time.Duration(i) * time.Hour), Address: &addr, KwhAdded: 10, Cost: &zero, Currency: "EUR"}
		if _, err := repo.UpsertTeslaMateCharge(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	list, _, err := repo.ListCharges(ctx, v.ID, false, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range list {
		ids[*c.TeslaMateChargeID] = c.ID
	}
	if err := repo.SetChargeCostReviewed(ctx, ids[1], v.ID, true); err != nil {
		t.Fatal(err)
	}

	s, err := NewEnergyStatsService(db.Pool, "UTC").PlaceStats(ctx, v.ID, v.UserID, PlacePeriod{Months: 12}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if g := group(t, s, pub.ID); g.FreeSessions != 1 || g.UnknownSessions != 1 || g.UnknownKwh != 10 || g.PricedKwh != 10 {
		t.Errorf("one confirmed free session, one 0 still to confirm: %+v", g)
	}
}
