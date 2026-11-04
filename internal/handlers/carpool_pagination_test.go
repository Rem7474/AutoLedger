package handlers

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func carpoolTitles(list map[string]any) []string {
	var titles []string
	for _, tr := range list["trips"].([]any) {
		titles = append(titles, tr.(map[string]any)["title"].(string))
	}
	return titles
}

func TestCarpoolListIsPaged(t *testing.T) {
	a := newCarpoolAPI(t, "paging")
	// Five trips on five days: the latest comes first
	for i := 1; i <= 5; i++ {
		a.expect(http.StatusCreated, "POST", "/carpools", fmt.Sprintf(`{"title":"T%d","date":"2026-03-0%d","distance_km":10,"electricity_cost":1}`, i, i))
	}

	first := a.expect(http.StatusOK, "GET", "/carpools?limit=2", "")
	if got := carpoolTitles(first); len(got) != 2 || got[0] != "T5" || got[1] != "T4" || first["total"].(float64) != 5 || first["page"].(float64) != 1 || first["limit"].(float64) != 2 {
		t.Fatalf("first page: %v", first)
	}
	last := a.expect(http.StatusOK, "GET", "/carpools?limit=2&page=3", "")
	if got := carpoolTitles(last); len(got) != 1 || got[0] != "T1" || last["total"].(float64) != 5 {
		t.Fatalf("last page: %v", last)
	}
	past := a.expect(http.StatusOK, "GET", "/carpools?limit=2&page=9", "")
	if len(carpoolTitles(past)) != 0 || past["total"].(float64) != 5 {
		t.Fatalf("a page past the end is empty: %v", past)
	}

	// No limit, a limit of zero or a negative page fall back to a page of 30; a huge limit is capped
	for query, wantLimit := range map[string]float64{"": 30, "?limit=0&page=-4": 30, "?limit=100000": 100} {
		got := a.expect(http.StatusOK, "GET", "/carpools"+query, "")
		if got["limit"].(float64) != wantLimit || got["page"].(float64) != 1 || len(carpoolTitles(got)) != 5 {
			t.Fatalf("%q: %v", query, got)
		}
	}
}

func TestCarpoolListFiltersByTripGroup(t *testing.T) {
	a := newCarpoolAPI(t, "group-filter")
	ctx := context.Background()
	group := &models.TripGroup{VehicleID: a.vid, Name: "Holidays"}
	if err := a.repo.CreateTripGroup(ctx, group, nil); err != nil {
		t.Fatal(err)
	}
	a.expect(http.StatusCreated, "POST", "/carpools", fmt.Sprintf(`{"title":"In the journey","date":"2026-03-02","distance_km":10,"trip_group_id":%q}`, group.ID))
	a.expect(http.StatusCreated, "POST", "/carpools", `{"title":"Elsewhere","date":"2026-03-03","distance_km":10}`)

	got := a.expect(http.StatusOK, "GET", "/carpools?trip_group_id="+group.ID, "")
	if titles := carpoolTitles(got); len(titles) != 1 || titles[0] != "In the journey" || got["total"].(float64) != 1 {
		t.Fatalf("the carpools of the journey: %v", got)
	}
	none := a.expect(http.StatusOK, "GET", "/carpools?trip_group_id=00000000-0000-0000-0000-000000000000", "")
	if len(carpoolTitles(none)) != 0 || none["total"].(float64) != 0 {
		t.Fatalf("an unknown journey has none: %v", none)
	}
}

// The summary covers every carpool whatever the page, and its shares are those of the trips
func TestCarpoolSummaryCoversAllTrips(t *testing.T) {
	a := newCarpoolAPI(t, "summary")
	empty := a.expect(http.StatusOK, "GET", "/carpools/summary", "")
	if empty["total_trips"].(float64) != 0 || empty["total_driver_share"].(float64) != 0 {
		t.Fatalf("nothing to total: %v", empty)
	}

	for i := 0; i < 3; i++ {
		a.expect(http.StatusCreated, "POST", "/carpools", twoLegCarpool)
	}
	all := a.expect(http.StatusOK, "GET", "/carpools", "")
	var driver, passengers float64
	for _, tr := range all["trips"].([]any) {
		trip := tr.(map[string]any)
		driver += trip["driver_cost_share"].(float64)
		passengers += trip["passengers_cost_share"].(float64)
	}
	sum := a.expect(http.StatusOK, "GET", "/carpools/summary", "")
	if sum["total_trips"].(float64) != 3 || sum["total_passengers"].(float64) != 6 {
		t.Fatalf("totals: %v", sum)
	}
	if !closeTo(sum["total_driver_share"].(float64), driver) || !closeTo(sum["total_passengers_share"].(float64), passengers) || passengers == 0 {
		t.Fatalf("the shares are the sum of the trips' (%v / %v): %v", driver, passengers, sum)
	}
	// A page of one does not change the summary
	a.expect(http.StatusOK, "GET", "/carpools?limit=1", "")
	again := a.expect(http.StatusOK, "GET", "/carpools/summary", "")
	if again["total_trips"] != sum["total_trips"] || again["total_driver_share"] != sum["total_driver_share"] {
		t.Fatalf("summary changed: %v vs %v", again, sum)
	}
}

func closeTo(a, b float64) bool {
	d := a - b
	return d < 0.005 && d > -0.005
}
