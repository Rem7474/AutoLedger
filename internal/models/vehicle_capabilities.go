package models

import "strings"

// VehicleCapabilities says what a vehicle records and offers, so a client reads a capability instead of comparing the
// powertrain or testing for a connection. The server computes it on every read (ComputeCapabilities); it is never stored
// and never sent by a client.
type VehicleCapabilities struct {
	// Charges: the vehicle has a battery to plug in (charging sessions, tariffs, home charging).
	Charges bool `json:"charges"`
	// Refuels: the vehicle has a tank (fill-ups).
	Refuels bool `json:"refuels"`
	// Drives: trips apply, because the vehicle can record them by hand or a source wrote some.
	Drives bool `json:"drives"`
	// Sync: a connected source (TeslaMate) synchronizes the vehicle's data.
	Sync bool `json:"sync"`
	// OdometerAuto: a source provides the odometer, so a manual reading is not asked for.
	OdometerAuto bool `json:"odometer_auto"`
	// BatteryHealth: the battery state of health can be read or recorded.
	BatteryHealth bool `json:"battery_health"`
	// TollsAuto: trips are checked for tolls automatically.
	TollsAuto bool `json:"tolls_auto"`
	// Write: the member may add and change records (owner or editor).
	Write bool `json:"write"`
}

// CapabilityFacts are the stored facts a capability depends on, beyond the vehicle row.
type CapabilityFacts struct {
	// HasDrives: at least one drive is stored, whatever its origin.
	HasDrives bool
	// HasSourceOdometer: at least one odometer reading was reported by a source (Home Assistant).
	HasSourceOdometer bool
}

// ComputeCapabilities derives the capabilities of a vehicle from its powertrain, its connection, the facts about its
// data and the role of the member reading it. It reads Role, Powertrain, TeslaMateAPIURL and TollDetectionActive, so it
// runs before the connection is hidden from a member who is not the owner.
func ComputeCapabilities(v *Vehicle, f CapabilityFacts) VehicleCapabilities {
	charges := v.CanCharge()
	sync := v.CanLinkTeslaMate() && v.TeslaMateAPIURL != nil && strings.TrimSpace(*v.TeslaMateAPIURL) != ""
	drives := charges || f.HasDrives
	return VehicleCapabilities{
		Charges:       charges,
		Refuels:       v.CanRefuel(),
		Drives:        drives,
		Sync:          sync,
		OdometerAuto:  sync || f.HasSourceOdometer,
		BatteryHealth: charges,
		TollsAuto:     drives && v.TollDetectionActive,
		Write:         v.Role.CanEdit(),
	}
}
