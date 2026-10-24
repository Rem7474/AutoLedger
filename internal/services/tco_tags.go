package services

import (
	"context"
	"sort"

	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/money"
)

// tagBreakdown splits the distance, the measured energy and the cost of the drives by tag. A drive carrying several
// tags counts in each of them (shared_drives says how many do); a drive with no tag counts under the empty tag. The
// cost of a drive is the one the drive list shows (ComputeDriveCost), so the total of a tag is the sum of the drives
// listed for it. Costs are only computed once a drive is tagged: the untagged-only split is never displayed.
func (s *TCOService) tagBreakdown(ctx context.Context, vehicleID string, totalDistance float64) ([]TagCostBreakdown, error) {
	drives, err := s.repo.ListDriveCostInputs(ctx, vehicleID)
	if err != nil {
		return nil, err
	}

	anyTagged := false
	ids := make([]string, len(drives))
	for i, d := range drives {
		ids[i] = d.ID
		if len(d.Tags) > 0 {
			anyTagged = true
		}
	}
	tolls, err := s.repo.GetTollExpensesForDrives(ctx, vehicleID, ids)
	if err != nil {
		return nil, err
	}

	var costs map[string]money.Cents
	if anyTagged {
		if costs, err = s.driveCosts(ctx, vehicleID, drives, tolls); err != nil {
			return nil, err
		}
	}

	return splitByTag(drives, costs, tolls, totalDistance), nil
}

// splitByTag groups drives by tag; costs may be nil, which leaves every cost at zero.
func splitByTag(drives []models.Drive, costs map[string]money.Cents, tolls map[string]money.Cents, totalDistance float64) []TagCostBreakdown {
	byTag := map[string]*TagCostBreakdown{}
	for _, d := range drives {
		tags := d.Tags
		if len(tags) == 0 {
			tags = []string{""}
		}
		cost := costs[d.ID]
		for _, tag := range tags {
			tb := byTag[tag]
			if tb == nil {
				tb = &TagCostBreakdown{Tag: tag}
				byTag[tag] = tb
			}
			tb.DistanceKm += d.Distance()
			if d.EnergyConsumedKwh != nil {
				tb.EnergyKwh += *d.EnergyConsumedKwh
			}
			tb.TollsAmount += tolls[d.ID]
			tb.Cost += cost
			if len(d.Tags) > 1 {
				tb.SharedDrives++
			}
		}
	}

	list := make([]TagCostBreakdown, 0, len(byTag))
	for _, tb := range byTag {
		if totalDistance > 0 {
			tb.Percentage = round1(tb.DistanceKm / totalDistance * 100)
		}
		if tb.DistanceKm > 0 && tb.Cost > 0 {
			tb.CostPerKm = round3(tb.Cost.Float() / tb.DistanceKm)
		}
		tb.DistanceKm = round1(tb.DistanceKm)
		tb.EnergyKwh = round1(tb.EnergyKwh)
		list = append(list, *tb)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].DistanceKm != list[j].DistanceKm {
			return list[i].DistanceKm > list[j].DistanceKm
		}
		return list[i].Tag < list[j].Tag
	})
	return list
}

// driveCosts is the cost of each drive as the drive list shows it: energy, wear, insurance share and tolls at the
// vehicle's unit rates. The insurance is shared over all the drives, so they must all be passed, whatever period the
// caller reports on.
func (s *TCOService) driveCosts(ctx context.Context, vehicleID string, drives []models.Drive, tolls map[string]money.Cents) (map[string]money.Cents, error) {
	rates, err := s.carpool.DriveRates(ctx, vehicleID, drives)
	if err != nil {
		return nil, err
	}
	insurance, err := s.carpool.AllocateInsuranceCosts(ctx, vehicleID, drives, rates)
	if err != nil {
		return nil, err
	}
	costs := make(map[string]money.Cents, len(drives))
	for _, d := range drives {
		costs[d.ID] = ComputeDriveCost(rates[d.ID], d, insurance[d.ID], tolls[d.ID]).TotalCost
	}
	return costs, nil
}
