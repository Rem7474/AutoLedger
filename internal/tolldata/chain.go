package tolldata

import "github.com/teslacost/teslacost/internal/money"

// ChainDrive is the GPS trace of one drive of a chain: drives separated by a stop (a service area, a charger),
// where the car left the place it came to.
type ChainDrive struct {
	Trace     []LatLon
	SpeedsKmh []float64 // nil when the trace has no speeds
}

// DriveSegment is a toll crossing as seen from one drive of a chain. A closed-network ticket whose entry and exit are
// on different drives belongs to each of them, and its price is shared between them.
type DriveSegment struct {
	Segment
	// Share is the part of the ticket's price this drive bears, nil when the ticket is not priced. It is the whole
	// price when the ticket stays on one drive, and Spans is then 1.
	Share *money.Cents
	// Spans is the number of drives the ticket covers.
	Spans int
}

// ChainResult is the toll crossings of a chain of drives.
type ChainResult struct {
	// PerDrive holds, for each drive of the chain, the crossings that touch it, in chronological order.
	PerDrive [][]DriveSegment
	// PendingHead is true when the chain starts with a closed-network gate that has no entry before it: it may be
	// the exit of a ticket taken on an earlier drive.
	PendingHead bool
	// PendingTail is true when the chain ends with a closed-network entry that has no exit after it: the exit may be
	// on a later drive.
	PendingTail bool
}

// DetectChain matches the traces of consecutive drives as if they were a single trace, so that an entry on one drive
// is paired with its exit on another. The crossings of each drive come back with its part of the price, shared along
// the distance of the ticket.
func (d *Dataset) DetectChain(drives []ChainDrive) ChainResult {
	var trace []LatLon
	var speeds []float64
	hasSpeeds := true
	offsets := make([]int, len(drives))
	for i, dr := range drives {
		offsets[i] = len(trace)
		trace = append(trace, dr.Trace...)
		speeds = append(speeds, dr.SpeedsKmh...)
		hasSpeeds = hasSpeeds && len(dr.SpeedsKmh) == len(dr.Trace)
	}
	if !hasSpeeds {
		speeds = nil
	}

	res := ChainResult{PerDrive: make([][]DriveSegment, len(drives))}
	matches := DetectCrossings(trace, speeds, d.Stations, DefaultThresholdMeters)
	segments := d.BuildSegments(matches)
	if len(segments) == 0 {
		return res
	}
	res.PendingHead = segments[0].Type == "close" && segments[0].Exit == nil
	res.PendingTail = segments[len(segments)-1].Type == "close" && segments[len(segments)-1].Exit == nil

	cumulative := CumulativeMeters(trace)
	// Drive i covers the trace from its first point to the first point of the next drive, the join included.
	starts := make([]float64, len(drives)+1)
	for i := range drives {
		starts[i] = cumulative[offsets[i]]
	}
	starts[len(drives)] = cumulative[len(cumulative)-1]

	for _, seg := range segments {
		touched := touchedDrives(seg, starts)
		shares := splitPrice(seg, starts, touched)
		for k, i := range touched {
			res.PerDrive[i] = append(res.PerDrive[i], DriveSegment{Segment: seg, Share: shares[k], Spans: len(touched)})
		}
	}
	return res
}

// touchedDrives returns the drives the segment runs over, in order: the drive of its entry up to the drive of its exit.
func touchedDrives(seg Segment, starts []float64) []int {
	first, last := driveAt(seg.StartMeters, starts), driveAt(seg.EndMeters, starts)
	touched := make([]int, 0, last-first+1)
	for i := first; i <= last; i++ {
		touched = append(touched, i)
	}
	return touched
}

// driveAt returns the drive covering a distance along the chain's trace.
func driveAt(meters float64, starts []float64) int {
	drive := 0
	for i := 1; i < len(starts)-1; i++ {
		if meters >= starts[i] {
			drive = i
		}
	}
	return drive
}

// splitPrice shares the price of a segment between the drives it covers, along the distance covered on each. The
// rounding is cumulative, so the shares add up to the price. A ticket with no distance (every station at the same
// place) is borne by the drive of its exit.
func splitPrice(seg Segment, starts []float64, touched []int) []*money.Cents {
	shares := make([]*money.Cents, len(touched))
	if seg.EstimatedPrice == nil {
		return shares
	}
	price := *seg.EstimatedPrice
	if len(touched) == 1 {
		shares[0] = &price
		return shares
	}

	total := seg.EndMeters - seg.StartMeters
	var done money.Cents
	covered := 0.0
	for k, i := range touched {
		lo, hi := starts[i], starts[i+1]
		if k == 0 {
			lo = seg.StartMeters
		}
		if k == len(touched)-1 {
			hi = seg.EndMeters
		}
		covered += hi - lo
		cumulative := price
		if k < len(touched)-1 && total > 0 {
			cumulative = money.Cents(float64(price)*covered/total + 0.5)
		}
		part := cumulative - done
		done = cumulative
		shares[k] = &part
	}
	return shares
}
