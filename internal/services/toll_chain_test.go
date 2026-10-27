package services

import (
	"context"
	"errors"
	"math"
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
	"github.com/teslacost/teslacost/internal/tolldata"
)

// fakeChainSource serves drives in chronological order, each with its trace.
type fakeChainSource struct {
	drives  []*models.Drive
	traces  map[string]tolldata.ChainDrive
	lookups int
	failOn  string
}

func (f *fakeChainSource) neighbour(_ context.Context, from *models.Drive, later bool) (*models.Drive, error) {
	f.lookups++
	for i, d := range f.drives {
		if d.ID != from.ID {
			continue
		}
		if later && i+1 < len(f.drives) {
			return f.drives[i+1], nil
		}
		if !later && i > 0 {
			return f.drives[i-1], nil
		}
	}
	return nil, database.ErrNotFound
}

func (f *fakeChainSource) trace(_ context.Context, d *models.Drive) (tolldata.ChainDrive, error) {
	if d.ID == f.failOn {
		return tolldata.ChainDrive{}, errors.New("teslamate unreachable")
	}
	return f.traces[d.ID], nil
}

// motorway is a straight road along the equator with an entry at lon 0 and an exit at lon 0.060, a service area in
// between, split in two drives that end and begin at the service area.
func motorway(t *testing.T, gap time.Duration) (*TollDetectionService, *fakeChainSource) {
	t.Helper()
	stations := []tolldata.Station{
		{Name: "ENTRY", Type: "close", Operator: "OP", Lat: 0, Lon: 0},
		{Name: "EXIT", Type: "close", Operator: "OP", Lat: 0, Lon: 0.060},
	}
	ds := &tolldata.Dataset{
		Stations:         stations,
		NetworkByStation: map[string]string{"ENTRY": "net", "EXIT": "net"},
		ClosedPrice:      map[string]map[string]money.Cents{"ENTRY": {"EXIT": 1000}},
	}
	trace := make([]tolldata.LatLon, 61)
	speeds := make([]float64, 61)
	for i := range trace {
		trace[i] = tolldata.LatLon{Lon: 0.001 * float64(i)}
		speeds[i] = 115
		if lon := trace[i].Lon; math.Abs(lon) <= 0.002 || math.Abs(lon-0.060) <= 0.002 {
			speeds[i] = 20
		}
	}
	t0 := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	first := &models.Drive{ID: "first", StartTime: t0, EndTime: t0.Add(30 * time.Minute)}
	last := &models.Drive{ID: "last", StartTime: first.EndTime.Add(gap), EndTime: first.EndTime.Add(gap + 30*time.Minute)}
	return &TollDetectionService{dataset: ds}, &fakeChainSource{
		drives: []*models.Drive{first, last},
		traces: map[string]tolldata.ChainDrive{
			"first": {Trace: trace[:31], SpeedsKmh: speeds[:31]},
			"last":  {Trace: trace[30:], SpeedsKmh: speeds[30:]},
		},
	}
}

func TestReadChain_AnEntryWithoutExitIsPairedWithTheNextDrive(t *testing.T) {
	s, src := motorway(t, 40*time.Minute)

	chain, err := s.readChain(context.Background(), src, src.drives[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.drives) != 2 || chain.target != 0 || len(chain.targetSegs) != 1 {
		t.Fatalf("the exit is on the next drive: %+v", chain)
	}
	seg := toModelSegments(chain.targetSegs)[0]
	if seg.Exit == nil || *seg.Exit != "EXIT" || seg.Spans != 2 || *seg.FullPrice != 1000 || seg.EstimatedPrice == nil || *seg.EstimatedPrice <= 0 || *seg.EstimatedPrice >= 1000 {
		t.Fatalf("the drive bears a share of the full ticket: %+v", seg)
	}
}

func TestReadChain_AGateWithoutEntryIsPairedWithThePreviousDrive(t *testing.T) {
	s, src := motorway(t, 40*time.Minute)

	chain, err := s.readChain(context.Background(), src, src.drives[1])
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.drives) != 2 || chain.target != 1 || chain.drives[0].ID != "first" {
		t.Fatalf("the entry is on the previous drive: %+v", chain)
	}
	seg := toModelSegments(chain.targetSegs)[0]
	if seg.Entry != "ENTRY" || seg.Exit == nil || seg.Spans != 2 {
		t.Fatalf("the drive carries the whole ticket's entry and exit: %+v", seg)
	}

	first, _ := s.readChain(context.Background(), src, src.drives[0])
	a, b := *toModelSegments(first.targetSegs)[0].EstimatedPrice, *seg.EstimatedPrice
	if a+b != 1000 {
		t.Fatalf("both drives together bear the full price, got %d + %d", a, b)
	}
}

func TestReadChain_TooLongAStopIsAnotherJourney(t *testing.T) {
	s, src := motorway(t, MaxStopGap+time.Minute)

	chain, err := s.readChain(context.Background(), src, src.drives[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.drives) != 1 || chain.targetSegs[0].Share != nil || chain.result.PerDrive[0][0].Exit != nil {
		t.Fatalf("a night between the drives does not join them: %+v", chain)
	}
}

func TestReadChain_ACarThatMovedWhileParkedIsNotTheSameJourney(t *testing.T) {
	s, src := motorway(t, 40*time.Minute)
	moved := src.traces["last"]
	moved.Trace = append([]tolldata.LatLon(nil), moved.Trace...)
	moved.Trace[0].Lat = 0.01 // about 1.1 km north of where the first drive ended
	src.traces["last"] = moved

	chain, err := s.readChain(context.Background(), src, src.drives[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.drives) != 1 {
		t.Fatalf("the next drive starts elsewhere: %d drives", len(chain.drives))
	}
}

func TestReadChain_NothingPendingReadsNoNeighbour(t *testing.T) {
	s, src := motorway(t, 40*time.Minute)
	whole := tolldata.ChainDrive{Trace: append(append([]tolldata.LatLon(nil), src.traces["first"].Trace...), src.traces["last"].Trace[1:]...)}
	whole.SpeedsKmh = append(append([]float64(nil), src.traces["first"].SpeedsKmh...), src.traces["last"].SpeedsKmh[1:]...)
	src.traces["first"] = whole

	chain, err := s.readChain(context.Background(), src, src.drives[0])
	if err != nil {
		t.Fatal(err)
	}
	if src.lookups != 0 || len(chain.drives) != 1 || chain.targetSegs[0].Spans != 1 || *chain.targetSegs[0].Share != 1000 {
		t.Fatalf("a complete ticket needs no neighbour (lookups %d): %+v", src.lookups, chain)
	}
}

func TestReadChain_AnUnreadableNeighbourLeavesTheDriveAlone(t *testing.T) {
	s, src := motorway(t, 40*time.Minute)
	src.failOn = "last"

	chain, err := s.readChain(context.Background(), src, src.drives[0])
	if err != nil || len(chain.drives) != 1 {
		t.Fatalf("the drive's own detection stands: %+v, %v", chain, err)
	}

	src.failOn = "first"
	if _, err := s.readChain(context.Background(), src, src.drives[0]); err == nil {
		t.Fatal("the drive's own trace must be readable")
	}
}

func TestReadChain_StopsAtTheChainLimit(t *testing.T) {
	s, src := motorway(t, 10*time.Minute)
	// Every drive after the first is a few metres of the same stop, so each one is continuous with the previous.
	stop := src.traces["first"].Trace[30]
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		src.drives = append(src.drives[:1], append([]*models.Drive{{ID: id, StartTime: src.drives[0].EndTime.Add(time.Duration(i+1) * time.Minute), EndTime: src.drives[0].EndTime.Add(time.Duration(i+1) * time.Minute)}}, src.drives[1:]...)...)
		src.traces[id] = tolldata.ChainDrive{Trace: []tolldata.LatLon{stop, stop}, SpeedsKmh: []float64{0, 0}}
	}
	// "last" starts 10 minutes after the first ends, which the stops fit in front of, so they would all be read.
	src.drives[len(src.drives)-1].StartTime = src.drives[0].EndTime.Add(20 * time.Minute)

	chain, err := s.readChain(context.Background(), src, src.drives[0])
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.drives) != MaxChainDrives {
		t.Fatalf("%d drives are read together at most and the ticket is still open, got %d", MaxChainDrives, len(chain.drives))
	}
}

func TestSharesATicket(t *testing.T) {
	a := []tolldata.DriveSegment{{Segment: tolldata.Segment{StartMeters: 0, EndMeters: 5000}, Spans: 2}}
	if !sharesATicket(a, a) {
		t.Error("the same ticket on two drives is shared")
	}
	other := []tolldata.DriveSegment{{Segment: tolldata.Segment{StartMeters: 6000, EndMeters: 9000}, Spans: 2}}
	if sharesATicket(a, other) {
		t.Error("two different tickets are not shared")
	}
	alone := []tolldata.DriveSegment{{Segment: tolldata.Segment{StartMeters: 0, EndMeters: 5000}, Spans: 1}}
	if sharesATicket(alone, alone) {
		t.Error("a ticket on one drive is not shared")
	}
}

func TestAutoTollNotesMarksAShareOfTheJourney(t *testing.T) {
	full := money.Cents(1240)
	segs := []models.TollSegment{{Type: "close", Entry: "A", Exit: strPtr("B"), Spans: 2, FullPrice: &full}}
	if got, want := autoTollNotes("en", segs), "Auto toll: A → B (share of a 12.40 € ticket over 2 drives)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := autoTollNotes("fr", segs); got != "Péage auto : A → B (part d'un ticket de 12,40 € sur 2 trajets)" {
		t.Errorf("got %q", got)
	}
	unpriced := []models.TollSegment{{Type: "close", Entry: "A", Exit: strPtr("B"), Spans: 2}}
	if got, want := autoTollNotes("en", unpriced), "Auto toll: A → B (share of the whole journey)"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
