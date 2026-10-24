package services

import (
	"context"
	"time"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// DriveSummary is what the drives of a period amount to, as the drive list and the tag breakdown count them.
type DriveSummary struct {
	Drives int
	// DrivesWithoutDistance are the drives whose distance was not recorded: they count in Drives but add nothing to
	// DistanceKm, so a cost per km over DistanceKm is not the cost per km of all the drives.
	DrivesWithoutDistance int
	DistanceKm            float64
	EnergyKwh             float64
	TollsAmount           money.Cents
	Cost                  money.Cents
	CostPerKm             float64
	UntaggedDrives        int
	// ToQualify counts the motorway-type drives of the period still waiting for a toll or a "no toll" review.
	ToQualify int
	// TripGroups counts the groups with at least one drive in the period.
	TripGroups int
	ByTag      []TagCostBreakdown
}

// DriveSummaryFor sums the drives that began between from and to (both optional, both included). The cost of a drive
// is the one the drive list shows; the insurance is shared over every drive of the vehicle, not only the ones of the
// period, so a drive costs the same whatever period is asked.
func (s *TCOService) DriveSummaryFor(ctx context.Context, vehicleID string, from, to *time.Time) (*DriveSummary, error) {
	all, err := s.repo.ListDriveCostInputs(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(all))
	for i, d := range all {
		ids[i] = d.ID
	}
	tolls, err := s.repo.GetTollExpensesForDrives(ctx, vehicleID, ids)
	if err != nil {
		return nil, err
	}
	costs, err := s.driveCosts(ctx, vehicleID, all, tolls)
	if err != nil {
		return nil, err
	}

	inPeriod := map[string]bool{}
	var drives []models.Drive
	out := &DriveSummary{}
	for _, d := range all {
		if (from != nil && d.StartTime.Before(*from)) || (to != nil && d.StartTime.After(*to)) {
			continue
		}
		drives = append(drives, d)
		inPeriod[d.ID] = true
		out.Drives++
		if d.DistanceKm == nil {
			out.DrivesWithoutDistance++
		}
		if len(d.Tags) == 0 {
			out.UntaggedDrives++
		}
		out.DistanceKm += d.Distance()
		if d.EnergyConsumedKwh != nil {
			out.EnergyKwh += *d.EnergyConsumedKwh
		}
		out.TollsAmount += tolls[d.ID]
		out.Cost += costs[d.ID]
	}
	if out.DistanceKm > 0 && out.Cost > 0 {
		out.CostPerKm = round3(out.Cost.Float() / out.DistanceKm)
	}
	out.DistanceKm, out.EnergyKwh = round1(out.DistanceKm), round1(out.EnergyKwh)
	out.ByTag = splitByTag(drives, costs, tolls, out.DistanceKm)

	if out.ToQualify, err = s.repo.CountUnqualifiedDrivesBetween(ctx, vehicleID, from, to); err != nil {
		return nil, err
	}
	groups, err := s.repo.ListTripGroups(ctx, vehicleID)
	if err != nil {
		return nil, err
	}
	for _, g := range groups {
		for _, id := range g.DriveIDs {
			if inPeriod[id] {
				out.TripGroups++
				break
			}
		}
	}
	return out, nil
}
