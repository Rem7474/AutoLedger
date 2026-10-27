package tolldata

import (
	"testing"

	"github.com/teslacost/teslacost/internal/money"
)

// chainFixture is a motorway along the equator: ENTRY at lon 0, a service area at lon 0.030 where the car stops, EXIT at
// lon 0.060. The first drive goes from ENTRY to the service area, the last from the service area to EXIT.
func chainFixture() (*Dataset, []Station) {
	stations, networkByStation := closedStations("net", map[string]float64{"ENTRY": 0.000, "EXIT": 0.060})
	ds := &Dataset{
		NetworkByStation: networkByStation,
		ClosedPrice:      map[string]map[string]money.Cents{"ENTRY": {"EXIT": 1000}},
		Stations:         stations,
	}
	return ds, stations
}

func slice(trace []LatLon, from, to int) ChainDrive {
	speeds := slowNear(trace, 20, 115, 0.000, 0.060)
	return ChainDrive{Trace: trace[from:to], SpeedsKmh: speeds[from:to]}
}

func TestDetectChain_AStopSplitsOneTicketAcrossTwoDrives(t *testing.T) {
	ds, _ := chainFixture()
	trace := buildStraightTrace(61, 0.001)
	first, last := slice(trace, 0, 31), slice(trace, 30, 61)

	alone := ds.DetectChain([]ChainDrive{first})
	if !alone.PendingTail || alone.PerDrive[0][0].Share != nil {
		t.Fatalf("an entry with no exit is pending and unpriced: %+v", alone)
	}
	orphan := ds.DetectChain([]ChainDrive{last})
	if !orphan.PendingHead || orphan.PerDrive[0][0].Share != nil {
		t.Fatalf("a lone gate reads as an entry with no exit: %+v", orphan)
	}

	res := ds.DetectChain([]ChainDrive{first, last})
	if res.PendingHead || res.PendingTail {
		t.Fatalf("the ticket is complete once both drives are read together: %+v", res)
	}
	var total money.Cents
	for i, segs := range res.PerDrive {
		if len(segs) != 1 || segs[0].Entry != "ENTRY" || segs[0].Exit == nil || *segs[0].Exit != "EXIT" || segs[0].Spans != 2 {
			t.Fatalf("drive %d should carry the whole ticket: %+v", i, segs)
		}
		if *segs[0].EstimatedPrice != 1000 {
			t.Fatalf("drive %d: the ticket costs the full price, got %d", i, *segs[0].EstimatedPrice)
		}
		total += *segs[0].Share
	}
	if total != 1000 {
		t.Fatalf("the shares add up to the price, got %d", total)
	}
	if a, b := *res.PerDrive[0][0].Share, *res.PerDrive[1][0].Share; a < 480 || a > 520 || b < 480 || b > 520 {
		t.Fatalf("drives of about the same length bear about half each, got %d and %d", a, b)
	}
}

func TestDetectChain_ShortMoveInsideTheServiceAreaBearsAlmostNothing(t *testing.T) {
	ds, _ := chainFixture()
	trace := buildStraightTrace(61, 0.001)
	first, last := slice(trace, 0, 31), slice(trace, 30, 61)
	// Three drives: to the service area, a few metres to the charger, then back on the motorway.
	move := ChainDrive{Trace: []LatLon{trace[30], {Lat: 0.0001, Lon: trace[30].Lon}}, SpeedsKmh: []float64{5, 5}}

	res := ds.DetectChain([]ChainDrive{first, move, last})
	var total money.Cents
	for i, segs := range res.PerDrive {
		if len(segs) != 1 || segs[0].Spans != 3 {
			t.Fatalf("drive %d should carry the ticket: %+v", i, segs)
		}
		total += *segs[0].Share
	}
	if total != 1000 {
		t.Fatalf("the shares add up to the price, got %d", total)
	}
	if share := *res.PerDrive[1][0].Share; share > 5 {
		t.Fatalf("a move of a few metres bears next to nothing, got %d", share)
	}
}

func TestDetectChain_TicketOnOneDriveIsNotShared(t *testing.T) {
	ds, _ := chainFixture()
	trace := buildStraightTrace(61, 0.001)
	whole := slice(trace, 0, 61)
	elsewhere := ChainDrive{Trace: []LatLon{{Lat: 10, Lon: 10}, {Lat: 10.01, Lon: 10}}, SpeedsKmh: []float64{50, 50}}

	res := ds.DetectChain([]ChainDrive{whole, elsewhere})
	if len(res.PerDrive[0]) != 1 || res.PerDrive[0][0].Spans != 1 || *res.PerDrive[0][0].Share != 1000 {
		t.Fatalf("a ticket entered and left on one drive costs its full price there: %+v", res.PerDrive[0])
	}
	if len(res.PerDrive[1]) != 0 {
		t.Fatalf("the other drive crossed nothing: %+v", res.PerDrive[1])
	}
}

func TestDetectChain_UnpricedTicketHasNoShare(t *testing.T) {
	ds, _ := chainFixture()
	ds.ClosedPrice = nil
	trace := buildStraightTrace(61, 0.001)

	res := ds.DetectChain([]ChainDrive{slice(trace, 0, 31), slice(trace, 30, 61)})
	for i, segs := range res.PerDrive {
		if len(segs) != 1 || segs[0].Share != nil {
			t.Fatalf("drive %d: a ticket the price table does not know has no share: %+v", i, segs)
		}
	}
}

func TestDetectChain_DriveWithoutSpeedsReadsTheWholeChainWithoutSpeeds(t *testing.T) {
	ds, _ := chainFixture()
	trace := buildStraightTrace(61, 0.001)
	first, last := slice(trace, 0, 31), slice(trace, 30, 61)
	last.SpeedsKmh = nil

	res := ds.DetectChain([]ChainDrive{first, last})
	if len(res.PerDrive[0]) != 1 || res.PerDrive[0][0].Share == nil {
		t.Fatalf("the ticket is still paired: %+v", res.PerDrive)
	}
}

func TestDetectChain_NoCrossing(t *testing.T) {
	ds, _ := chainFixture()
	res := ds.DetectChain([]ChainDrive{{Trace: []LatLon{{Lat: 10, Lon: 10}, {Lat: 10.01, Lon: 10}}}})
	if res.PendingHead || res.PendingTail || len(res.PerDrive) != 1 || len(res.PerDrive[0]) != 0 {
		t.Fatalf("nothing crossed: %+v", res)
	}
	if res := ds.DetectChain(nil); len(res.PerDrive) != 0 {
		t.Fatalf("no drive, no result: %+v", res)
	}
}
