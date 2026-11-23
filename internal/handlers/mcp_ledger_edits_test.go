package handlers

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

func mustFail(t *testing.T, what string, err error, contains string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), contains) {
		t.Fatalf("%s: want an error containing %q, got %v", what, contains, err)
	}
}

func TestUpdateFuelFillKeepsWhatIsNotNamedAndWorksTheThirdFigureOut(t *testing.T) {
	a := newCarpoolAPI(t, "edit-fuel")
	ctx := context.Background()
	ice := &models.Vehicle{UserID: a.userID, Name: "Diesel", Currency: "EUR", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone}
	if err := a.repo.CreateVehicle(ctx, ice); err != nil {
		t.Fatal(err)
	}
	liters, price, odo := 40.0, 1.5, 50000.0
	fill := &models.FuelLog{VehicleID: ice.ID, Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Amount: 6000, Liters: &liters, PricePerLiter: &price, Odometer: &odo, IsFullTank: true}
	if err := a.repo.CreateFuelLog(ctx, fill); err != nil {
		t.Fatal(err)
	}
	h := writeHandler(a)
	amount := "66.00"
	in := mcpUpdateFuelFillInput{VehicleID: ice.ID, FillID: fill.ID, Amount: &amount, DryRun: true}

	preview, err := h.UpdateFuelFill(ctx, a.userID, ice.ID, in)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Updated || preview.After.Amount.Amount != "66.00" || *preview.After.Liters != 40 || *preview.After.PricePerLiter != 1.65 || preview.After.Odometer.Value != 50000 {
		t.Fatalf("a new amount keeps the litres and works the price out: %+v", preview)
	}
	if got, _ := a.repo.GetFuelLog(ctx, ice.ID, fill.ID); got.Amount != 6000 {
		t.Fatalf("a dry run changed the fill-up: %+v", got)
	}
	in.DryRun = false
	_, err = h.UpdateFuelFill(ctx, a.userID, ice.ID, in)
	mustFail(t, "no version", err, "expected_version")
	stale := preview.Version + 1
	in.ExpectedVersion = &stale
	_, err = h.UpdateFuelFill(ctx, a.userID, ice.ID, in)
	mustFail(t, "stale version", err, "changed since")
	in.ExpectedVersion = &preview.Version
	saved, err := h.UpdateFuelFill(ctx, a.userID, ice.ID, in)
	if err != nil || !saved.Updated || saved.Version != preview.Version+1 {
		t.Fatalf("saved: %+v %v", saved, err)
	}
	if got, _ := a.repo.GetFuelLog(ctx, ice.ID, fill.ID); got.Amount != 6600 || *got.PricePerLiter != 1.65 || got.Date.Format("2006-01-02") != "2026-09-01" {
		t.Fatalf("stored: %+v", got)
	}
	_, err = h.UpdateFuelFill(ctx, a.userID, ice.ID, mcpUpdateFuelFillInput{VehicleID: ice.ID, FillID: fill.ID, DryRun: true})
	mustFail(t, "empty change", err, "nothing to change")
	_, err = h.UpdateFuelFill(ctx, a.viewerID, ice.ID, mcpUpdateFuelFillInput{VehicleID: ice.ID, FillID: fill.ID, Amount: &amount, DryRun: true})
	if err == nil {
		t.Fatal("a viewer must not change a fill-up")
	}
}

func TestUpdateChargeChangesAManualOneAndOnlyTheCostOfASynchronisedOne(t *testing.T) {
	a := newCarpoolAPI(t, "edit-charge")
	ctx := context.Background()
	h := writeHandler(a)
	veh, err := a.repo.GetVehicleByID(ctx, a.vid, a.userID)
	if err != nil {
		t.Fatal(err)
	}
	cost := money.Cents(780)
	manual := &models.ChargeLog{VehicleID: a.vid, Date: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), KwhAdded: 30, Cost: &cost, Currency: "EUR"}
	if err := a.repo.CreateManualCharge(ctx, manual); err != nil {
		t.Fatal(err)
	}
	newCost, kwh := "9.00", 36.0
	preview, err := h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Cost: &newCost, KwhAdded: &kwh, DryRun: true})
	if err != nil || preview.Updated || preview.After.Cost.Amount != "9.00" || preview.After.Energy != 36 || preview.Before.Cost.Amount != "7.80" {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Cost: &newCost})
	mustFail(t, "no version", err, "expected_version")
	saved, err := h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Cost: &newCost, KwhAdded: &kwh, ExpectedVersion: &preview.Version})
	if err != nil || !saved.Updated || saved.After.Cost.Amount != "9.00" || saved.After.Energy != 36 {
		t.Fatalf("saved: %+v %v", saved, err)
	}
	if got, _ := a.repo.GetCharge(ctx, a.vid, manual.ID); got.Date.Format(time.RFC3339) != "2026-09-01T12:00:00Z" {
		t.Fatalf("the day the call did not name is kept as it was: %v", got.Date)
	}
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Cost: &newCost, ExpectedVersion: &preview.Version})
	mustFail(t, "replay", err, "changed since")

	// A synchronised charge with a 0 cost: only the cost and the review change
	synced := &models.ChargeLog{VehicleID: a.vid, Date: time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC), KwhAdded: 20, Cost: &cost, Currency: "EUR"}
	if err := a.repo.CreateManualCharge(ctx, synced); err != nil {
		t.Fatal(err)
	}
	if _, err := a.repo.Pool().Exec(ctx, `UPDATE charge_logs SET is_manual = FALSE, origin = 'TESLAMATE', cost_source = 'TESLAMATE', cost = 0 WHERE id::text = $1`, synced.ID); err != nil {
		t.Fatal(err)
	}
	current, _ := a.repo.GetCharge(ctx, a.vid, synced.ID)
	if !current.CostToReview {
		t.Fatalf("a synchronised 0 cost is to review: %+v", current)
	}
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: synced.ID, KwhAdded: &kwh, DryRun: true})
	mustFail(t, "figures of a synchronised charge", err, "synchronised")
	yes := true
	done, err := h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: synced.ID, CostReviewed: &yes, ExpectedVersion: &current.Version})
	if err != nil || !done.Updated || done.After.CostToReview || !done.After.Synchronised {
		t.Fatalf("confirming the 0 cost: %+v %v", done, err)
	}
	_ = veh
}

func TestOdometerReadingsCanBeChangedAndDeletedAtTheVersionRead(t *testing.T) {
	a := newCarpoolAPI(t, "edit-odo")
	ctx := context.Background()
	h := writeHandler(a)
	var ids []string
	for i, km := range []float64{1000, 2000, 3000} {
		c := &models.OdometerCheckpoint{VehicleID: a.vid, Date: time.Date(2026, 8, 1+i, 0, 0, 0, 0, time.UTC), Odometer: km}
		if err := a.repo.CreateOdometerCheckpoint(ctx, c); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, c.ID)
	}
	km := 2500.0
	preview, err := h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: ids[1], OdometerKm: &km, DryRun: true})
	if err != nil || preview.Updated || preview.After.Odometer.Value != 2500 || preview.Before.Odometer.Value != 2000 {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	tooHigh := 3500.0
	_, err = h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: ids[1], OdometerKm: &tooHigh, DryRun: true})
	if err == nil {
		t.Fatal("a mileage above the next reading is refused")
	}
	saved, err := h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: ids[1], OdometerKm: &km, ExpectedVersion: &preview.Version})
	if err != nil || !saved.Updated || saved.Version != preview.Version+1 {
		t.Fatalf("saved: %+v %v", saved, err)
	}

	del := mcpDeleteRecordInput{Kind: "odometer_reading", ID: ids[0], DryRun: true}
	shown, err := h.DeleteRecord(ctx, a.userID, a.vid, del)
	if err != nil || shown.Deleted || shown.Record.ID != ids[0] {
		t.Fatalf("delete preview: %+v %v", shown, err)
	}
	del.DryRun = false
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, del)
	mustFail(t, "no version", err, "expected_version")
	del.ExpectedVersion = &shown.Record.Version
	if out, err := h.DeleteRecord(ctx, a.userID, a.vid, del); err != nil || !out.Deleted {
		t.Fatalf("deleted: %+v %v", out, err)
	}
	if list, _ := a.repo.ListOdometerCheckpoints(ctx, a.vid); len(list) != 2 {
		t.Fatalf("one reading is gone: %d", len(list))
	}
	// A reading an integration reports is not the owner's to delete
	if err := a.repo.RecordIntegrationOdometer(ctx, a.vid, time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), 4000); err != nil {
		t.Fatal(err)
	}
	list, _ := a.repo.ListOdometerCheckpoints(ctx, a.vid)
	for _, c := range list {
		if c.Source != "MANUAL" {
			one := int64(1)
			_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "odometer_reading", ID: c.ID, ExpectedVersion: &one})
			mustFail(t, "integration reading", err, "no such entry")
		}
	}
}

func TestDeleteRecordRemovesATollInvoiceAndACarpool(t *testing.T) {
	a := newCarpoolAPI(t, "edit-delete")
	ctx := context.Background()
	h := writeHandler(a)
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if _, err := a.repo.SaveTollInvoice(ctx, a.vid, "2026-08", start, start.AddDate(0, 1, 0), 14830, nil, nil); err != nil {
		t.Fatal(err)
	}
	shown, err := h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "toll_invoice", ID: "2026-08", DryRun: true})
	if err != nil || shown.Deleted || shown.Record.Amount.Amount != "148.30" || shown.Record.Date != "2026-08-01" {
		t.Fatalf("invoice preview: %+v %v", shown, err)
	}
	wrong := shown.Record.Version + 1
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "toll_invoice", ID: "2026-08", ExpectedVersion: &wrong})
	mustFail(t, "stale invoice", err, "changed since")
	if out, err := h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "toll_invoice", ID: "2026-08", ExpectedVersion: &shown.Record.Version}); err != nil || !out.Deleted {
		t.Fatalf("invoice deleted: %+v %v", out, err)
	}
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "toll_invoice", ID: "2026-08", DryRun: true})
	mustFail(t, "no invoice", err, "no toll invoice")

	drive := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon, France", "Valence, France", 100)
	made, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, mcpCreateCarpoolInput{DriveIDs: []string{drive.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: "Alice", AmountPaid: "5"}}})
	if err != nil {
		t.Fatal(err)
	}
	id := made.Carpool.ID
	shown, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "carpool", ID: id, DryRun: true})
	if err != nil || shown.Deleted || shown.Record.ID != id {
		t.Fatalf("carpool preview: %+v %v", shown, err)
	}
	stale := shown.Record.Version + 5
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "carpool", ID: id, ExpectedVersion: &stale})
	mustFail(t, "stale carpool", err, "changed since")
	if out, err := h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "carpool", ID: id, ExpectedVersion: &shown.Record.Version}); err != nil || !out.Deleted {
		t.Fatalf("carpool deleted: %+v %v", out, err)
	}
	if _, err := a.repo.GetDriveByID(ctx, drive.ID, a.vid); err != nil {
		t.Fatalf("the drive stays: %v", err)
	}
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "carpool", ID: id, DryRun: true})
	mustFail(t, "gone", err, "no such carpool")
}

func TestCarpoolCostsFreezesUnfreezesAndRecalculates(t *testing.T) {
	a := newCarpoolAPI(t, "edit-costs")
	ctx := context.Background()
	h := writeHandler(a)
	drive := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon, France", "Valence, France", 100)
	made, err := h.CreateCarpoolFromDrives(ctx, a.userID, a.vid, mcpCreateCarpoolInput{DriveIDs: []string{drive.ID}, Passengers: []mcpCarpoolPassengerInput{{Name: "Alice", AmountPaid: "5"}}})
	if err != nil {
		t.Fatal(err)
	}
	id := made.Carpool.ID
	frozen := func() bool {
		trip, err := a.repo.GetCarpoolTrip(ctx, id, a.vid)
		if err != nil {
			t.Fatal(err)
		}
		return trip.CostsFrozenAt != nil
	}
	for _, step := range []struct {
		action string
		frozen bool
	}{{"freeze", true}, {"recalculate", true}, {"unfreeze", false}} {
		out, err := h.CarpoolCosts(ctx, a.userID, a.vid, mcpCarpoolCostsInput{CarpoolID: id, Action: step.action})
		if err != nil || !out.Saved || out.Carpool.ID != id || frozen() != step.frozen {
			t.Fatalf("%s: %+v %v frozen=%v", step.action, out, err, frozen())
		}
	}
	_, err = h.CarpoolCosts(ctx, a.userID, a.vid, mcpCarpoolCostsInput{CarpoolID: id, Action: "melt"})
	mustFail(t, "unknown action", err, "action must be")
	_, err = h.CarpoolCosts(ctx, a.viewerID, a.vid, mcpCarpoolCostsInput{CarpoolID: id, Action: "freeze"})
	if err == nil {
		t.Fatal("a viewer must not freeze a carpool")
	}
}

func TestUpdateTripsTagsDriverAndTollReviewAreAppliedToAllOrNone(t *testing.T) {
	a := newCarpoolAPI(t, "edit-trips")
	ctx := context.Background()
	h := writeHandler(a)
	first := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon, France", "Valence, France", 100)
	second := manualDrive(t, a, time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC), "Valence, France", "Lyon, France", 100)
	if err := a.repo.UpdateDriveTags(ctx, second.ID, a.vid, []string{"Pro"}); err != nil {
		t.Fatal(err)
	}
	people, err := a.repo.ListVehiclePeople(ctx, a.vid)
	if err != nil || len(people) == 0 {
		t.Fatalf("a vehicle has its owner as a person: %v %v", people, err)
	}
	in := mcpUpdateTripsInput{DriveIDs: []string{first.ID, second.ID}, AddTags: []string{" pro ", "holiday"}, RemoveTags: []string{"missing"}, DryRun: true}
	preview, err := h.UpdateTrips(ctx, a.userID, a.vid, in)
	if err != nil || preview.Saved || len(preview.Trips) != 2 || !preview.Trips[0].Changed || len(preview.Trips[0].Tags) != 2 || len(preview.Trips[1].Tags) != 2 {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	if d, _ := a.repo.GetDriveByID(ctx, first.ID, a.vid); len(d.Tags) != 0 {
		t.Fatalf("a dry run tagged the trip: %v", d.Tags)
	}
	// All or none: one unknown trip stops everything
	bad := in
	bad.DryRun, bad.DriveIDs = false, []string{first.ID, "00000000-0000-4000-8000-000000000009"}
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, bad)
	mustFail(t, "unknown trip", err, "nothing was changed")
	if d, _ := a.repo.GetDriveByID(ctx, first.ID, a.vid); len(d.Tags) != 0 {
		t.Fatalf("a refused call tagged a trip: %v", d.Tags)
	}
	in.DryRun = false
	driver, reviewed := people[0].Name, true
	in.Driver, in.TollReviewed = &driver, &reviewed
	saved, err := h.UpdateTrips(ctx, a.userID, a.vid, in)
	if err != nil || !saved.Saved {
		t.Fatalf("saved: %+v %v", saved, err)
	}
	d, _ := a.repo.GetDriveByID(ctx, first.ID, a.vid)
	name, _ := a.repo.GetDriveDriverName(ctx, first.ID, a.vid)
	if len(d.Tags) != 2 || d.TollReviewedAt == nil || name == nil || *name != driver {
		t.Fatalf("first trip: %+v driver=%v", d, name)
	}
	d2, _ := a.repo.GetDriveByID(ctx, second.ID, a.vid)
	if len(d2.Tags) != 2 {
		t.Fatalf("a tag the trip already has (any case) is kept once: %v", d2.Tags)
	}
	cleared := ""
	if _, err := h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{first.ID}, Driver: &cleared, RemoveTags: []string{"HOLIDAY"}}); err != nil {
		t.Fatal(err)
	}
	if name, _ := a.repo.GetDriveDriverName(ctx, first.ID, a.vid); name != nil {
		t.Fatalf("an empty driver clears it: %v", *name)
	}
	if d, _ := a.repo.GetDriveByID(ctx, first.ID, a.vid); len(d.Tags) != 1 {
		t.Fatalf("a tag is removed whatever its case: %v", d.Tags)
	}
	unknown := "Nobody Here"
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{first.ID}, Driver: &unknown})
	mustFail(t, "unknown driver", err, "no person named")
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{first.ID}})
	mustFail(t, "empty", err, "nothing to change")
	_, err = h.UpdateTrips(ctx, a.viewerID, a.vid, in)
	if err == nil {
		t.Fatal("a viewer must not change trips")
	}
}

func TestManageTripGroupCreatesUpdatesAndDeletesAtTheVersionRead(t *testing.T) {
	a := newCarpoolAPI(t, "edit-groups")
	ctx := context.Background()
	h := writeHandler(a)
	first := manualDrive(t, a, time.Date(2026, 9, 5, 8, 0, 0, 0, time.UTC), "Lyon, France", "Valence, France", 100)
	second := manualDrive(t, a, time.Date(2026, 9, 6, 8, 0, 0, 0, time.UTC), "Valence, France", "Lyon, France", 100)
	name, notes := " Holiday ", "  south  "
	drives := []string{first.ID}
	create := mcpManageTripGroupInput{Action: "create", Name: &name, Notes: &notes, DriveIDs: &drives, DryRun: true}
	preview, err := h.ManageTripGroup(ctx, a.userID, a.vid, create)
	if err != nil || preview.Done || preview.Group.Name != "Holiday" || preview.Group.ID != "" {
		t.Fatalf("preview: %+v %v", preview, err)
	}
	if groups, _ := a.repo.ListTripGroups(ctx, a.vid); len(groups) != 0 {
		t.Fatalf("a dry run created a group: %v", groups)
	}
	create.DryRun = false
	made, err := h.ManageTripGroup(ctx, a.userID, a.vid, create)
	if err != nil || !made.Done || made.Group.ID == "" || made.Group.Drives != 1 || *made.Group.Notes != "south" {
		t.Fatalf("created: %+v %v", made, err)
	}
	listed, _ := h.TripGroups(ctx, a.userID, a.vid, 0)
	if listed.Groups[0].ID != made.Group.ID || len(listed.Groups[0].DriveIDs) != 1 || listed.Groups[0].Version != made.Group.Version {
		t.Fatalf("list_trip_groups carries the id, the version and the drives: %+v", listed.Groups)
	}
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "create"})
	mustFail(t, "create without drives", err, "needs a name")

	both := []string{first.ID, second.ID}
	rename := "Holiday 2026"
	upd := mcpManageTripGroupInput{Action: "update", GroupID: made.Group.ID, Name: &rename, DriveIDs: &both}
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, upd)
	mustFail(t, "no version", err, "expected_version")
	stale := made.Group.Version + 3
	upd.ExpectedVersion = &stale
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, upd)
	mustFail(t, "stale", err, "changed since")
	upd.ExpectedVersion = &made.Group.Version
	updated, err := h.ManageTripGroup(ctx, a.userID, a.vid, upd)
	if err != nil || !updated.Done || updated.Group.Name != "Holiday 2026" || updated.Group.Drives != 2 || updated.Group.Version != made.Group.Version+1 {
		t.Fatalf("updated: %+v %v", updated, err)
	}
	none := []string{}
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "update", GroupID: made.Group.ID, DriveIDs: &none, DryRun: true})
	mustFail(t, "empty group", err, "at least one drive")

	del := mcpManageTripGroupInput{Action: "delete", GroupID: made.Group.ID, ExpectedVersion: &updated.Group.Version}
	if _, err := h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "delete", GroupID: made.Group.ID, ExpectedVersion: &made.Group.Version}); err == nil {
		t.Fatal("the old version must not delete")
	}
	if out, err := h.ManageTripGroup(ctx, a.userID, a.vid, del); err != nil || !out.Done {
		t.Fatalf("deleted: %+v %v", out, err)
	}
	if _, err := a.repo.GetDriveByID(ctx, first.ID, a.vid); err != nil {
		t.Fatalf("the drives stay: %v", err)
	}
	_, err = h.ManageTripGroup(ctx, a.viewerID, a.vid, create)
	if err == nil {
		t.Fatal("a viewer must not change groups")
	}
}

func TestEstimateTripTollsNeedsDetectionAndAWritingRole(t *testing.T) {
	a := newCarpoolAPI(t, "edit-tolls")
	ctx := context.Background()
	h := writeHandler(a)
	_, err := h.EstimateTripTolls(ctx, a.userID, a.vid, mcpEstimateTollsInput{DriveIDs: []string{"x"}})
	mustFail(t, "no service", err, "unavailable")
	_, err = h.EstimateTripTolls(ctx, a.viewerID, a.vid, mcpEstimateTollsInput{DriveIDs: []string{"x"}})
	mustFail(t, "viewer", err, "read-only")
}

func TestLedgerEditToolsRefuseAReaderAndWhatIsNotAskedProperly(t *testing.T) {
	a := newCarpoolAPI(t, "edit-refusals")
	ctx := context.Background()
	h := writeHandler(a)
	missing := "00000000-0000-4000-8000-000000000009"
	one := int64(1)

	// A member who may only read changes nothing, whichever tool is called.
	_, err := h.UpdateFuelFill(ctx, a.viewerID, a.vid, mcpUpdateFuelFillInput{FillID: missing, ExpectedVersion: &one})
	mustFail(t, "viewer fuel fill", err, "read-only")
	_, err = h.UpdateCharge(ctx, a.viewerID, a.vid, mcpUpdateChargeInput{ChargeID: missing, ExpectedVersion: &one})
	mustFail(t, "viewer charge", err, "read-only")
	_, err = h.UpdateOdometerReading(ctx, a.viewerID, a.vid, mcpUpdateOdometerInput{ReadingID: missing, ExpectedVersion: &one})
	mustFail(t, "viewer reading", err, "read-only")
	_, err = h.CarpoolCosts(ctx, a.viewerID, a.vid, mcpCarpoolCostsInput{CarpoolID: missing, Action: "freeze"})
	mustFail(t, "viewer carpool", err, "read-only")
	_, err = h.UpdateTrips(ctx, a.viewerID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{missing}, AddTags: []string{"x"}})
	mustFail(t, "viewer trips", err, "read-only")
	_, err = h.ManageTripGroup(ctx, a.viewerID, a.vid, mcpManageTripGroupInput{Action: "delete", GroupID: missing})
	mustFail(t, "viewer group", err, "read-only")
	_, err = h.DeleteRecord(ctx, a.viewerID, a.vid, mcpDeleteRecordInput{Kind: "carpool", ID: missing, DryRun: true})
	mustFail(t, "viewer delete", err, "read-only")

	// The owner is refused what cannot be found or understood, with a message to act on.
	_, err = h.UpdateFuelFill(ctx, a.userID, a.vid, mcpUpdateFuelFillInput{FillID: missing, DryRun: true})
	mustFail(t, "unknown fill", err, "")
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: missing, DryRun: true})
	mustFail(t, "unknown charge", err, "")
	_, err = h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: missing, DryRun: true})
	mustFail(t, "unknown reading", err, "")
	_, err = h.CarpoolCosts(ctx, a.userID, a.vid, mcpCarpoolCostsInput{CarpoolID: missing, Action: "freeze"})
	mustFail(t, "unknown carpool", err, "")
	_, err = h.CarpoolCosts(ctx, a.userID, a.vid, mcpCarpoolCostsInput{CarpoolID: missing, Action: "melt"})
	mustFail(t, "bad action", err, "freeze, unfreeze or recalculate")
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{missing}})
	mustFail(t, "nothing to change", err, "nothing to change")
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{missing}, AddTags: []string{strings.Repeat("x", 51)}})
	mustFail(t, "tag too long", err, "")
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{missing}, AddTags: []string{"x"}})
	mustFail(t, "unknown trip", err, "not a trip of this vehicle")
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{AddTags: []string{"x"}})
	mustFail(t, "no trip", err, "drive_ids")
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "rename"})
	mustFail(t, "bad group action", err, "")
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "delete", GroupID: missing, DryRun: true})
	mustFail(t, "unknown group", err, "")
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "odometer_reading", ID: missing, DryRun: true})
	mustFail(t, "unknown reading to delete", err, "")
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "toll_invoice", ID: "not-a-month", DryRun: true})
	mustFail(t, "bad month", err, "")
	_, err = h.DeleteRecord(ctx, a.userID, a.vid, mcpDeleteRecordInput{Kind: "carpool", ID: missing, DryRun: true})
	mustFail(t, "unknown carpool to delete", err, "")
}

func TestLedgerEditToolsAcceptEveryFieldTheyAdvertise(t *testing.T) {
	a := newCarpoolAPI(t, "edit-fields")
	ctx := context.Background()
	h := writeHandler(a)

	// a fill-up: every figure at once, then the third one worked out from each pair
	ice := &models.Vehicle{UserID: a.userID, Name: "Petrol", Currency: "EUR", Powertrain: models.PowertrainICE, TeslaMateAuthType: models.AuthModeNone}
	if err := a.repo.CreateVehicle(ctx, ice); err != nil {
		t.Fatal(err)
	}
	liters, price, odo := 40.0, 1.5, 50000.0
	fill := &models.FuelLog{VehicleID: ice.ID, Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), Amount: 6000, Liters: &liters, PricePerLiter: &price, Odometer: &odo, IsFullTank: true}
	if err := a.repo.CreateFuelLog(ctx, fill); err != nil {
		t.Fatal(err)
	}
	date, fuel, notes, full, km, l, p := "2026-09-02", "DIESEL", " a note ", false, 50100.0, 30.0, 1.8
	got, err := h.UpdateFuelFill(ctx, a.userID, ice.ID, mcpUpdateFuelFillInput{FillID: fill.ID, Date: &date, FuelType: &fuel, Notes: &notes, IsFullTank: &full, OdometerKm: &km, Liters: &l, PricePerLiter: &p, DryRun: true})
	if err != nil || got.After.Amount.Amount != "54.00" || got.After.Date != date || got.After.IsFullTank {
		t.Fatalf("every field: %+v %v", got, err)
	}
	for _, in := range []mcpUpdateFuelFillInput{{Liters: &l}, {PricePerLiter: &p}} {
		in.FillID, in.DryRun = fill.ID, true
		if _, err := h.UpdateFuelFill(ctx, a.userID, ice.ID, in); err != nil {
			t.Fatalf("one figure alone: %v", err)
		}
	}
	long := strings.Repeat("n", 501)
	_, err = h.UpdateFuelFill(ctx, a.userID, ice.ID, mcpUpdateFuelFillInput{FillID: fill.ID, Notes: &long, DryRun: true})
	mustFail(t, "long note", err, "500 characters")
	_, err = h.UpdateFuelFill(ctx, a.userID, ice.ID, mcpUpdateFuelFillInput{DryRun: true})
	mustFail(t, "no id", err, "fill_id is required")

	// a manual charge: every field, with a note removed and a note added
	cost := money.Cents(780)
	noted := "old"
	manual := &models.ChargeLog{VehicleID: a.vid, Date: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC), KwhAdded: 30, Cost: &cost, Currency: "EUR", Notes: &noted}
	if err := a.repo.CreateManualCharge(ctx, manual); err != nil {
		t.Fatal(err)
	}
	c, kwh, odoKm, start, end, empty, reviewed := "12.50", 40.0, 12000.0, 20, 80, "", true
	out, err := h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Cost: &c, Date: &date, KwhAdded: &kwh, OdometerKm: &odoKm, StartBatteryLevel: &start, EndBatteryLevel: &end, Notes: &empty, CostReviewed: &reviewed, DryRun: true})
	if err != nil || out.After.Odometer == nil || out.After.Date != date {
		t.Fatalf("every field: %+v %v", out, err)
	}
	note := " kept "
	if _, err := h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Notes: &note, DryRun: true}); err != nil {
		t.Fatal(err)
	}
	zero := 0.0
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, KwhAdded: &zero, DryRun: true})
	mustFail(t, "no energy", err, "more than zero")
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{ChargeID: manual.ID, Notes: &long, DryRun: true})
	mustFail(t, "long note", err, "500 characters")
	_, err = h.UpdateCharge(ctx, a.userID, a.vid, mcpUpdateChargeInput{DryRun: true})
	mustFail(t, "no id", err, "charge_id is required")

	// an odometer reading: its day and its note
	reading := &models.OdometerCheckpoint{VehicleID: a.vid, Date: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Odometer: 1000}
	if err := a.repo.CreateOdometerCheckpoint(ctx, reading); err != nil {
		t.Fatal(err)
	}
	later := "2026-08-05"
	ro, err := h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: reading.ID, Date: &later, Notes: &note, DryRun: true})
	if err != nil || ro.After.Date != later {
		t.Fatalf("reading: %+v %v", ro, err)
	}
	_, err = h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: reading.ID, Notes: &long, DryRun: true})
	mustFail(t, "long note", err, "500 characters")
	_, err = h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{DryRun: true})
	mustFail(t, "no id", err, "reading_id is required")
	_, err = h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: reading.ID, Notes: &note})
	mustFail(t, "no version", err, "expected_version")
	stale := reading.Version + 7
	_, err = h.UpdateOdometerReading(ctx, a.userID, a.vid, mcpUpdateOdometerInput{ReadingID: reading.ID, Notes: &note, ExpectedVersion: &stale})
	mustFail(t, "stale", err, "changed since")

	// deleting: a stale version, then an id that names nothing
	for _, in := range []mcpDeleteRecordInput{
		{Kind: "odometer_reading", ID: reading.ID, ExpectedVersion: &stale},
		{Kind: "fuel_fill_up", ID: fill.ID, ExpectedVersion: &stale},
		{Kind: "banana", ID: reading.ID, DryRun: true},
		{Kind: "carpool", ID: "../x", DryRun: true},
		{Kind: "carpool", ID: reading.ID},
		{Kind: "toll_invoice", ID: "2026-01"},
		{Kind: "carpool", DryRun: true},
	} {
		if _, err := h.DeleteRecord(ctx, a.userID, a.vid, in); err == nil {
			t.Fatalf("%+v must be refused", in)
		}
	}

	// trips and groups: too many tags, nothing to change, bad names
	many := make([]string, 0, 25)
	for i := range 25 {
		many = append(many, strings.Repeat("t", 3)+string(rune('a'+i)))
	}
	d := manualDrive(t, a, time.Date(2026, 9, 3, 8, 0, 0, 0, time.UTC), "A", "B", 10).ID
	_, err = h.UpdateTrips(ctx, a.userID, a.vid, mcpUpdateTripsInput{DriveIDs: []string{d, d}, AddTags: many, DryRun: true})
	mustFail(t, "too many tags", err, "tags at most")
	empty2 := ""
	name := strings.Repeat("g", 151)
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "create", Name: &name})
	mustFail(t, "long name", err, "")
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "create", Name: &empty2})
	mustFail(t, "empty name", err, "")
	_, err = h.ManageTripGroup(ctx, a.userID, a.vid, mcpManageTripGroupInput{Action: "update"})
	mustFail(t, "no group", err, "group_id is required")
	_, err = h.CarpoolCosts(ctx, a.userID, a.vid, mcpCarpoolCostsInput{CarpoolID: "../x", Action: "freeze"})
	mustFail(t, "bad id", err, "not a carpool id")
}
