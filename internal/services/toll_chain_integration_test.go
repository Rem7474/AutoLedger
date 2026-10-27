package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/tolldata"
)

// stopFixture serves, through a fake TeslaMateAPI, a motorway journey cut in two drives by a stop in a service area:
// ENTRY at lon 0 and EXIT at lon 0.060 of one closed network, the first drive ending and the second beginning at lon
// 0.030. The fare ENTRY -> EXIT is 10.00.
func stopFixture(t *testing.T) (*TollDetectionService, *models.Vehicle, *models.Drive, *models.Drive) {
	t.Helper()
	_, repo := setupIntegrationDB(t, false)
	v := mustVehicle(t, repo, fmt.Sprintf("stop-%d@example.com", time.Now().UnixNano()))

	position := func(i int) map[string]any {
		lon := 0.001 * float64(i)
		speed := 115.0
		if math.Abs(lon) <= 0.002 || math.Abs(lon-0.060) <= 0.002 {
			speed = 20
		}
		return map[string]any{"latitude": 0, "longitude": lon, "date": "2026-09-20T09:00:00Z", "speed": speed}
	}
	route := regexp.MustCompile(`/api/v1/cars/1/drives/(\d+)$`)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m := route.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		from, to := 0, 31 // the drive that enters the motorway and stops
		if m[1] == "202" {
			from, to = 30, 61
		}
		var details []map[string]any
		for i := from; i < to; i++ {
			details = append(details, position(i))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
			"drive": map[string]any{"drive_details": details},
			"units": map[string]any{"unit_of_length": "km"},
		}})
	}))
	t.Cleanup(ts.Close)
	v.TeslaMateAPIURL = &ts.URL

	start := time.Date(2026, 9, 20, 9, 0, 0, 0, time.UTC)
	first := mustDrive(t, repo, v.ID, 201, start, 20000, 60)
	last := mustDrive(t, repo, v.ID, 202, start.Add(100*time.Minute), 20060, 60)

	dataset := &tolldata.Dataset{
		Stations: []tolldata.Station{
			{Name: "ENTRY", Type: "close", Operator: "OP", Lat: 0, Lon: 0},
			{Name: "EXIT", Type: "close", Operator: "OP", Lat: 0, Lon: 0.060},
		},
		NetworkByStation: map[string]string{"ENTRY": "net", "EXIT": "net"},
		ClosedPrice:      map[string]map[string]money.Cents{"ENTRY": {"EXIT": 1000}},
	}
	return &TollDetectionService{repo: repo, dataset: dataset}, v, first, last
}

func autoTollOf(t *testing.T, s *TollDetectionService, v *models.Vehicle, d *models.Drive) money.Cents {
	t.Helper()
	expenses, err := s.repo.GetDriveExpensesByDriveID(context.Background(), v.ID, d.ID, "en")
	if err != nil {
		t.Fatal(err)
	}
	var total money.Cents
	for _, e := range expenses {
		if e.Type == "TOLL" && e.Source == models.ExpenseSourceAutoToll {
			total += e.Amount
		}
	}
	return total
}

func TestIntegrationApplyTollEstimateSharesATicketBetweenTheDrivesOfAStop(t *testing.T) {
	s, v, first, last := stopFixture(t)
	ctx := context.Background()

	res, err := s.ApplyTollEstimate(ctx, v, first.ID)
	if err != nil || res.Status != ApplyCreated || res.Amount == nil {
		t.Fatalf("the drive that entered the motorway is priced with its share: %+v, %v", res, err)
	}
	detection, err := s.repo.GetTollDetectionByDrive(ctx, v.ID, first.ID)
	if err != nil || len(detection.Segments) != 1 || detection.Segments[0].Spans != 2 ||
		detection.Segments[0].Exit == nil || *detection.Segments[0].Exit != "EXIT" || *detection.Segments[0].FullPrice != 1000 {
		t.Fatalf("the detection names the whole ticket and its price: %+v, %v", detection, err)
	}

	res, err = s.ApplyTollEstimate(ctx, v, last.ID)
	if err != nil || res.Status != ApplyCreated {
		t.Fatalf("the drive that left it is priced too: %+v, %v", res, err)
	}
	a, b := autoTollOf(t, s, v, first), autoTollOf(t, s, v, last)
	if a <= 0 || b <= 0 || a+b != 1000 {
		t.Fatalf("the two shares add up to the 10.00 fare, got %d + %d", a, b)
	}

	if res, err = s.ApplyTollEstimate(ctx, v, first.ID); err != nil || res.Status != ApplyUpdated {
		t.Fatalf("applying again updates the share in place: %+v, %v", res, err)
	}
	if again := autoTollOf(t, s, v, first); again != a {
		t.Fatalf("the share does not change from one run to the next: %d then %d", a, again)
	}
}

func TestIntegrationApplyTollEstimateLeavesAJourneyEnteredByHand(t *testing.T) {
	s, v, first, last := stopFixture(t)
	ctx := context.Background()

	byHand := &models.DriveExpense{VehicleID: v.ID, DriveID: &last.ID, Type: "TOLL", Amount: 1100, Currency: "EUR", Date: last.StartTime}
	if err := s.repo.SaveDriveExpense(ctx, byHand, nil, ""); err != nil {
		t.Fatal(err)
	}

	res, err := s.ApplyTollEstimate(ctx, v, first.ID)
	if err != nil || res.Status != ApplySkippedManual {
		t.Fatalf("the toll typed on the other drive covers the journey: %+v, %v", res, err)
	}
	if got := autoTollOf(t, s, v, first); got != 0 {
		t.Fatalf("no automatic toll is recorded on top of it, got %d", got)
	}
}
