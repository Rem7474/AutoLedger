package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/teslacost/teslacost/internal/apierror"
	"github.com/teslacost/teslacost/internal/crypto"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/teslamate"
	"github.com/teslacost/teslacost/internal/tolldata"
)

// ErrNoGPSTrace is returned when a drive has no TeslaMate GPS trace to detect tolls from
// (e.g. a manually-entered drive).
var ErrNoGPSTrace = apierror.New("toll.no_gps", "This drive has no TeslaMate GPS trace available")

// TollDetectionService matches a drive's GPS trace against the vendored OpenTollData
// toll station reference to detect which toll gates it crossed.
type TollDetectionService struct {
	repo      *database.Repository
	encryptor *crypto.Encryptor
	dataset   *tolldata.Dataset
}

// NewTollDetectionService loads the embedded toll dataset and returns a ready-to-use service.
func NewTollDetectionService(repo *database.Repository, encryptor *crypto.Encryptor) (*TollDetectionService, error) {
	dataset, err := tolldata.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load toll dataset: %w", err)
	}
	return &TollDetectionService{repo: repo, encryptor: encryptor, dataset: dataset}, nil
}

// DetectTolls fetches the drive's GPS trace from TeslaMateAPI, matches it against the toll
// station reference, and persists the result. vehicle must belong to the caller (checked by
// the handler before calling this).
func (s *TollDetectionService) DetectTolls(ctx context.Context, vehicle *models.Vehicle, driveID string) (*models.TollDetection, error) {
	chain, err := s.detectChain(ctx, vehicle, driveID)
	if err != nil {
		return nil, err
	}
	return chain.persisted, nil
}

// tollChain is the detection of a drive read together with the drives of the same journey around it.
type tollChain struct {
	drives     []*models.Drive
	result     tolldata.ChainResult
	target     int
	persisted  *models.TollDetection
	targetSegs []tolldata.DriveSegment
}

func (s *TollDetectionService) detectChain(ctx context.Context, vehicle *models.Vehicle, driveID string) (*tollChain, error) {
	drive, err := s.repo.GetDriveByID(ctx, driveID, vehicle.ID)
	if err != nil {
		return nil, err
	}
	if drive.TeslaMateDriveID == nil {
		return nil, ErrNoGPSTrace
	}
	if vehicle.TeslaMateCarID == nil {
		return nil, ErrNoGPSTrace
	}

	client, err := buildTeslaMateClient(vehicle, s.encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to build TeslaMate client: %w", err)
	}

	chain, err := s.readChain(ctx, &teslaMateChainSource{repo: s.repo, client: client, vehicle: vehicle}, drive)
	if err != nil {
		return nil, err
	}

	td := &models.TollDetection{
		DriveID:   drive.ID,
		VehicleID: vehicle.ID,
		Segments:  toModelSegments(chain.targetSegs),
	}
	if err := s.repo.UpsertTollDetection(ctx, td); err != nil {
		return nil, fmt.Errorf("failed to save toll detection: %w", err)
	}
	chain.persisted = td
	return chain, nil
}

const (
	// MaxStopGap is the longest stop after which the next drive is still read as the continuation of a journey: a
	// meal and a charge at a service area, not a night.
	MaxStopGap = 3 * time.Hour
	// MaxStopJumpMeters is the largest distance between where a drive ended and where the next one begins for the car to
	// be the same stop: it did not move while parked, the GPS only drifts.
	MaxStopJumpMeters = 300.0
	// MaxChainDrives bounds the drives read together, each one costing a TeslaMateAPI call.
	MaxChainDrives = 5
)

// chainSource gives the neighbours of a drive and their GPS traces.
type chainSource interface {
	// neighbour returns the drive that follows (later) or precedes from, database.ErrNotFound when there is none.
	neighbour(ctx context.Context, from *models.Drive, later bool) (*models.Drive, error)
	trace(ctx context.Context, d *models.Drive) (tolldata.ChainDrive, error)
}

// readChain reads the drive's trace alone, then adds the neighbouring drives for as long as a ticket stays open on
// the edge of the chain: an entry with no exit at its end, or a gate with no entry before it at its start, means the
// journey was cut by a stop (the car left a service area, three drives for a stop and a move to the charger).
// A neighbour only joins the chain when the car did not move while parked. Neighbours that cannot be read leave the
// drive's own result.
func (s *TollDetectionService) readChain(ctx context.Context, src chainSource, drive *models.Drive) (*tollChain, error) {
	own, err := src.trace(ctx, drive)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch drive GPS trace: %w", err)
	}
	drives := []*models.Drive{drive}
	traces := []tolldata.ChainDrive{own}
	target := 0

	canExtend := map[bool]bool{true: true, false: true} // later -> still worth looking
	var res tolldata.ChainResult
	for {
		res = s.dataset.DetectChain(traces)
		if len(drives) >= MaxChainDrives {
			break
		}
		later := res.PendingTail && canExtend[true]
		earlier := !later && res.PendingHead && canExtend[false]
		if !later && !earlier {
			break
		}
		edge := drives[0]
		if later {
			edge = drives[len(drives)-1]
		}
		next, nextTrace, ok := s.stopNeighbour(ctx, src, edge, traces, later)
		if !ok {
			canExtend[later] = false
			continue
		}
		if later {
			drives, traces = append(drives, next), append(traces, nextTrace)
		} else {
			drives, traces = append([]*models.Drive{next}, drives...), append([]tolldata.ChainDrive{nextTrace}, traces...)
			target++
		}
	}
	return &tollChain{drives: drives, result: res, target: target, targetSegs: res.PerDrive[target]}, nil
}

// stopNeighbour returns the drive after (later) or before edge when the car stood still between the two.
func (s *TollDetectionService) stopNeighbour(ctx context.Context, src chainSource, edge *models.Drive, traces []tolldata.ChainDrive, later bool) (*models.Drive, tolldata.ChainDrive, bool) {
	none := tolldata.ChainDrive{}
	next, err := src.neighbour(ctx, edge, later)
	if err != nil {
		if !errors.Is(err, database.ErrNotFound) {
			slog.WarnContext(ctx, "toll detection: neighbouring drive unavailable", "component", "toll", "error", err)
		}
		return nil, none, false
	}
	gap := next.StartTime.Sub(edge.EndTime)
	if !later {
		gap = edge.StartTime.Sub(next.EndTime)
	}
	if gap > MaxStopGap {
		return nil, none, false
	}
	nextTrace, err := src.trace(ctx, next)
	if err != nil || len(nextTrace.Trace) < 2 {
		if err != nil {
			slog.WarnContext(ctx, "toll detection: neighbouring drive trace unavailable", "component", "toll", "error", err)
		}
		return nil, none, false
	}
	var from, to tolldata.LatLon
	if later {
		last := traces[len(traces)-1].Trace
		from, to = last[len(last)-1], nextTrace.Trace[0]
	} else {
		from, to = nextTrace.Trace[len(nextTrace.Trace)-1], traces[0].Trace[0]
	}
	if tolldata.DistanceMeters(from, to) > MaxStopJumpMeters {
		return nil, none, false
	}
	return next, nextTrace, true
}

// teslaMateChainSource reads the neighbours from the database and their traces from TeslaMateAPI.
type teslaMateChainSource struct {
	repo    *database.Repository
	client  *teslamate.Client
	vehicle *models.Vehicle
}

func (c *teslaMateChainSource) neighbour(ctx context.Context, from *models.Drive, later bool) (*models.Drive, error) {
	return c.repo.GetNeighbourTeslaMateDrive(ctx, c.vehicle.ID, from, later)
}

func (c *teslaMateChainSource) trace(ctx context.Context, d *models.Drive) (tolldata.ChainDrive, error) {
	positions, units, err := c.client.GetDriveDetails(ctx, *c.vehicle.TeslaMateCarID, *d.TeslaMateDriveID)
	if err != nil {
		return tolldata.ChainDrive{}, err
	}
	trace := make([]tolldata.LatLon, len(positions))
	for i, p := range positions {
		trace[i] = tolldata.LatLon{Lat: p.Latitude, Lon: p.Longitude}
	}
	return tolldata.ChainDrive{Trace: trace, SpeedsKmh: traceSpeedsKmh(positions, units.UnitOfLength)}, nil
}

// traceSpeedsKmh returns the trace's speeds in km/h, nil when TeslaMate sent none (it reports a
// missing speed as 0, so a trace where every speed is 0 has no speed information).
func traceSpeedsKmh(positions []teslamate.DrivePosition, unitOfLength string) []float64 {
	speeds := make([]float64, len(positions))
	known := false
	for i, p := range positions {
		speeds[i] = teslamate.ConvertDistanceToKm(p.Speed, unitOfLength)
		known = known || p.Speed > 0
	}
	if !known {
		return nil
	}
	return speeds
}

// toModelSegments converts the crossings of one drive; a ticket shared with other drives carries this drive's part as
// its price and the whole price next to it.
func toModelSegments(segments []tolldata.DriveSegment) []models.TollSegment {
	out := make([]models.TollSegment, len(segments))
	for i, s := range segments {
		out[i] = models.TollSegment{
			Network:        s.Network,
			Operator:       s.Operator,
			Type:           s.Type,
			Entry:          s.Entry,
			Exit:           s.Exit,
			EstimatedPrice: s.Share,
		}
		if s.Spans > 1 {
			out[i].FullPrice = s.EstimatedPrice
			out[i].Spans = s.Spans
		}
	}
	return out
}
