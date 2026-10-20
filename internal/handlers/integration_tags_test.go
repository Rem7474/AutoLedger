package handlers

import (
	"testing"

	"github.com/teslacost/teslacost/internal/services"
)

func TestIntegrationTagsSplitTheDrivesAndLeaveTheUntaggedOnesWithoutAName(t *testing.T) {
	out := buildIntegrationTags("v1", "CHF", []services.TagCostBreakdown{
		{Tag: "pro", DistanceKm: 1200, EnergyKwh: 210, TollsAmount: 3500, Percentage: 60, Cost: 42000, CostPerKm: 0.35, SharedDrives: 2},
		{Tag: "", DistanceKm: 0, Cost: 0},
	})

	if len(out.Tags) != 2 || out.Tags[0].Tag == nil || *out.Tags[0].Tag != "pro" || out.Tags[0].Cost.Amount != "420.00" ||
		out.Tags[0].CostPerKm == nil || out.Tags[0].CostPerKm.Amount != "0.3500" || out.Tags[0].Tolls.Amount != "35.00" || out.Tags[0].SharedDrives != 2 {
		t.Fatalf("a tag row: %+v", out.Tags)
	}
	if out.Tags[1].Tag != nil || out.Tags[1].CostPerKm != nil {
		t.Errorf("untagged drives have no name and no rate without distance: %+v", out.Tags[1])
	}
	if empty := buildIntegrationTags("v1", "CHF", nil); empty.Tags == nil || len(empty.Tags) != 0 {
		t.Errorf("no tag is an empty list: %+v", empty.Tags)
	}
}
