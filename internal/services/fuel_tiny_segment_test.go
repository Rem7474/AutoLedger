package services

import (
	"encoding/json"
	"testing"

	"github.com/teslacost/teslacost/internal/models"
)

func TestFuelStatsTinySegmentRemainsSerializable(t *testing.T) {
	stats := ComputeFuelStats([]models.FuelLog{fill(10000, 60, lit(30), true), fill(10000.01, 60, lit(30), true), fill(10500, 60, lit(30), true)}, nil)
	if stats.UnmeasurableCnt != 1 || stats.MeasurableCount != 1 || stats.Logs[1].ConsumptionL100 != nil || stats.Logs[1].CostPerKm != nil {
		t.Fatalf("tiny segment must be unmeasurable without breaking the next one: %+v", stats)
	}
	if _, err := json.Marshal(stats); err != nil {
		t.Fatalf("fuel statistics must remain valid JSON: %v", err)
	}
}
