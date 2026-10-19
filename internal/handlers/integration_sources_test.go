package handlers

import (
	"testing"
	"time"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

func TestIntegrationSourcesSayWhereTheDataComesFromAndHowFreshItIs(t *testing.T) {
	url := "http://teslamate.local"
	last := time.Date(2026, 10, 18, 7, 30, 0, 0, time.UTC)
	owner := &models.Vehicle{ID: "v1", Role: models.RoleOwner, TeslaMateAPIURL: &url}
	activity := []database.SourceActivity{{Origin: "TESLAMATE", Drives: 120, Charges: 40, LastAt: &last}, {Origin: "MANUAL", OdometerReadings: 3}}
	states := []database.SyncState{{Resource: "charges", LastSuccessAt: &last}, {Resource: "drives"}}

	out := buildIntegrationSources(owner, activity, states, 2)

	if !out.TeslaMate || out.PendingCharges != 2 || len(out.Sources) != 2 || out.Sources[0].Origin != "teslamate" || out.Sources[0].LastRecord == nil || *out.Sources[0].LastRecord != "2026-10-18T07:30:00Z" || out.Sources[1].LastRecord != nil {
		t.Fatalf("sources: %+v", out)
	}
	if len(out.Sync) != 2 || out.Sync[0].LastSuccess == nil || out.Sync[1].LastSuccess != nil || out.Sync[1].HistoryImported != nil {
		t.Errorf("sync: %+v", out.Sync)
	}
	// The address of the TeslaMate instance is never returned, and a member does not see the connection
	member := &models.Vehicle{ID: "v1", Role: models.RoleViewer, TeslaMateAPIURL: &url}
	if buildIntegrationSources(member, nil, nil, 0).TeslaMate {
		t.Error("only the owner sees the TeslaMate connection")
	}
	if none := buildIntegrationSources(&models.Vehicle{ID: "v1", Role: models.RoleOwner}, nil, nil, 0); none.TeslaMate || none.Sources == nil || none.Sync == nil {
		t.Errorf("a vehicle without source: %+v", none)
	}
}
