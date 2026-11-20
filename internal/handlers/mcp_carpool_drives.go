package handlers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

// carpoolDriveChange is what update_carpool does to the legs of a carpool: the drives taken out, then the drives
// linked.
type carpoolDriveChange struct {
	add    []string
	remove []string
}

func (c carpoolDriveChange) any() bool { return len(c.add)+len(c.remove) > 0 }

// carpoolDriveChanges checks the drive ids of a call: well formed, each listed once, and none both added and removed.
func carpoolDriveChanges(add, remove []string) (carpoolDriveChange, error) {
	var out carpoolDriveChange
	var err error
	if len(add) > 0 {
		if out.add, err = carpoolDriveIDs(add); err != nil {
			return out, err
		}
	}
	if len(remove) > 0 {
		if out.remove, err = carpoolDriveIDs(remove); err != nil {
			return out, err
		}
	}
	for _, id := range out.add {
		if slices.Contains(out.remove, id) {
			return out, mcpUserError(fmt.Sprintf("drive %s is both in add_drives and remove_drives", id))
		}
	}
	return out, nil
}

// changeCarpoolDrives applies the drive changes to the carpool as it is now. It returns the payloads of the legs
// after the change, and the passengers with their stops moved along: a passenger who rode to the end of the trip
// rides to the end of the longer one, the stops after a removed leg shift down, and a passenger left with no leg
// to ride on is refused. The trip group of the carpool is left as it is, as the carpool editor leaves it. Drives taken out are the legs made from a drive; the legs the owner typed have none.
func (h *IntegrationReadHandler) changeCarpoolDrives(ctx context.Context, vehicle *models.Vehicle, existing *models.CarpoolTripWithPassengers, change carpoolDriveChange) ([]LegPayload, []models.CarpoolPassenger, error) {
	legsOf := func(src []models.CarpoolLeg) []LegPayload {
		out := make([]LegPayload, 0, len(src))
		for _, l := range src {
			out = append(out, LegPayload{
				DriveID: l.DriveID, StartLabel: l.StartLabel, EndLabel: l.EndLabel, DistanceKm: l.DistanceKm,
				EnergyCost: &l.EnergyCost, FuelCost: &l.FuelCost, TollsCost: l.TollsCost, TiresCost: l.TiresCost,
				MaintenanceCost: l.MaintenanceCost, InsuranceCost: l.InsuranceCost, OtherCost: l.OtherCost, Overrides: l.Overrides,
			})
		}
		return out
	}
	oldLen := len(existing.Legs)
	kept := make([]models.CarpoolLeg, 0, oldLen)
	removedBefore := make([]int, oldLen+1) // legs removed before each stop index
	removed := 0
	for i, l := range existing.Legs {
		removedBefore[i] = removed
		if l.DriveID != nil && slices.Contains(change.remove, *l.DriveID) {
			removed++
			continue
		}
		kept = append(kept, l)
	}
	removedBefore[oldLen] = removed
	if removed != len(change.remove) {
		var have []string
		for _, l := range existing.Legs {
			if l.DriveID != nil {
				have = append(have, *l.DriveID)
			}
		}
		return nil, nil, mcpUserError(fmt.Sprintf("remove_drives names a drive that is not a leg of this carpool: its drives are %s", strings.Join(have, ", ")))
	}
	for _, l := range existing.Legs {
		if l.DriveID != nil && slices.Contains(change.add, *l.DriveID) {
			return nil, nil, mcpUserError(fmt.Sprintf("drive %s is already a leg of this carpool", *l.DriveID))
		}
	}
	payloads := legsOf(kept)

	if len(change.add) > 0 {
		shared, err := h.repo.GetCarpoolsForDrives(ctx, vehicle.ID, change.add)
		if err != nil {
			return nil, nil, err
		}
		for _, id := range change.add {
			if refs := shared[id]; len(refs) > 0 {
				return nil, nil, mcpUserError(fmt.Sprintf("drive %s is already in the carpool %q (carpool id %s): a drive is shared once, take it out of that carpool first", id, refs[0].Title, refs[0].ID))
			}
		}
		estimate, err := h.carpool.EstimateCosts(ctx, vehicle.ID, nil, nil, change.add, 0)
		if errors.Is(err, database.ErrNotFound) {
			return nil, nil, mcpUserError("a drive in add_drives is not a drive of this vehicle: take them from list_vehicle_trips for the same vehicle")
		}
		if err != nil {
			return nil, nil, err
		}
		if len(estimate.Legs) != len(change.add) {
			return nil, nil, errors.New("the drives could not be priced")
		}
		payloads = append(payloads, legsOf(estimate.Legs)...)
	}
	if len(payloads) == 0 {
		return nil, nil, mcpUserError("a carpool keeps at least one drive: take its passengers out instead of all its drives")
	}
	if len(payloads) > maxCarpoolDrives {
		return nil, nil, mcpUserError(fmt.Sprintf("a carpool takes %d drives at most", maxCarpoolDrives))
	}

	passengers := slices.Clone(existing.Passengers)
	for i := range passengers {
		p := &passengers[i]
		board, alight := p.BoardStopIndex-removedBefore[p.BoardStopIndex], p.AlightStopIndex-removedBefore[p.AlightStopIndex]
		if p.AlightStopIndex == oldLen {
			alight = len(payloads) // rode to the end of the trip: rides to the end of the new one
		}
		if alight <= board {
			return nil, nil, mcpUserError(fmt.Sprintf("%s rode only on the drive taken out: take the passenger out of the carpool first (remove_passengers)", passengerLabel(p.PassengerName, i)))
		}
		if board != p.BoardStopIndex || alight != p.AlightStopIndex {
			p.Origin, p.Destination = nil, nil // the labels follow the stops again
		}
		p.BoardStopIndex, p.AlightStopIndex = board, alight
	}
	return payloads, passengers, nil
}
