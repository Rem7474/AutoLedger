package models

import "testing"

func TestComputeCapabilitiesMatrix(t *testing.T) {
	url := "http://teslamate.local"
	blank := "  "
	cases := []struct {
		name  string
		v     Vehicle
		facts CapabilityFacts
		want  VehicleCapabilities
	}{
		{
			name: "electric vehicle with TeslaMate, owner",
			v:    Vehicle{Powertrain: PowertrainEV, Role: RoleOwner, TeslaMateAPIURL: &url, TollDetectionActive: true},
			want: VehicleCapabilities{Charges: true, Drives: true, Sync: true, OdometerAuto: true, BatteryHealth: true, TollsAuto: true, Write: true},
		},
		{
			name: "electric vehicle without telemetry, editor",
			v:    Vehicle{Powertrain: PowertrainEV, Role: RoleEditor},
			want: VehicleCapabilities{Charges: true, Drives: true, BatteryHealth: true, Write: true},
		},
		{
			name: "a blank connection address is no connection",
			v:    Vehicle{Powertrain: PowertrainEV, Role: RoleOwner, TeslaMateAPIURL: &blank},
			want: VehicleCapabilities{Charges: true, Drives: true, BatteryHealth: true, Write: true},
		},
		{
			name: "combustion vehicle, owner",
			v:    Vehicle{Powertrain: PowertrainICE, Role: RoleOwner, TollDetectionActive: true},
			want: VehicleCapabilities{Refuels: true, Write: true},
		},
		{
			name:  "combustion vehicle with trips pushed by a source, viewer",
			v:     Vehicle{Powertrain: PowertrainICE, Role: RoleViewer, TollDetectionActive: true},
			facts: CapabilityFacts{HasDrives: true, HasSourceOdometer: true},
			want:  VehicleCapabilities{Refuels: true, Drives: true, OdometerAuto: true, TollsAuto: true},
		},
		{
			name: "plug-in hybrid charges and refuels",
			v:    Vehicle{Powertrain: PowertrainPHEV, Role: RoleOwner},
			want: VehicleCapabilities{Charges: true, Refuels: true, Drives: true, BatteryHealth: true, Write: true},
		},
		{
			name: "range extender charges and refuels",
			v:    Vehicle{Powertrain: PowertrainREEV, Role: RoleViewer},
			want: VehicleCapabilities{Charges: true, Refuels: true, Drives: true, BatteryHealth: true},
		},
		{
			name: "a connection address on a combustion vehicle is ignored",
			v:    Vehicle{Powertrain: PowertrainICE, Role: RoleOwner, TeslaMateAPIURL: &url},
			want: VehicleCapabilities{Refuels: true, Write: true},
		},
		{
			name:  "an odometer reported by a source without a connection",
			v:     Vehicle{Powertrain: PowertrainEV, Role: RoleOwner},
			facts: CapabilityFacts{HasSourceOdometer: true},
			want:  VehicleCapabilities{Charges: true, Drives: true, OdometerAuto: true, BatteryHealth: true, Write: true},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ComputeCapabilities(&c.v, c.facts); got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}
