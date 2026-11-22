package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
)

func TestMCPRecordOdometerChecksRecordsAndReplays(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	writeTok := f.newToken(t, f.owner.ID, auth.ScopeWrite)
	call := func(token string, args map[string]any) (*mcp.CallToolResult, IntegrationOdometerRecorded) {
		t.Helper()
		res, err := f.connect(t, token).CallTool(ctx, &mcp.CallToolParams{Name: "record_odometer", Arguments: args})
		if err != nil {
			t.Fatal(err)
		}
		var out IntegrationOdometerRecorded
		if !res.IsError {
			raw, _ := json.Marshal(res.StructuredContent)
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatal(err)
			}
		}
		return res, out
	}
	reason := func(res *mcp.CallToolResult) string { return res.Content[0].(*mcp.TextContent).Text }
	count := func() int {
		list, err := f.repo.ListOdometerCheckpoints(ctx, f.mine.ID)
		if err != nil {
			t.Fatal(err)
		}
		return len(list)
	}
	args := func(extra map[string]any) map[string]any {
		a := map[string]any{"vehicle_id": f.mine.ID, "date": "2026-09-01", "odometer_km": 12000.0, "notes": " After the service ", "idempotency_key": "odo-1"}
		for k, v := range extra {
			a[k] = v
		}
		return a
	}
	start := count()

	if res, _ := call(f.readTok, args(nil)); !res.IsError || !strings.Contains(reason(res), "can only read") {
		t.Fatalf("a read connection must not record: %+v", res)
	}

	res, preview := call(writeTok, args(map[string]any{"dry_run": true}))
	if res.IsError || preview.Recorded || preview.Reading.ID != "" || preview.Reading.Odometer.Value != 12000 || preview.Reading.Notes == nil || *preview.Reading.Notes != "After the service" {
		t.Fatalf("a dry run shows the reading and records nothing: %+v %+v", res, preview)
	}
	if count() != start {
		t.Fatal("a dry run recorded a reading")
	}

	res, first := call(writeTok, args(nil))
	if res.IsError || !first.Recorded || first.Reading.ID == "" || count() != start+1 {
		t.Fatalf("the call records: %+v %+v", res, first)
	}
	list, err := f.repo.ListOdometerCheckpoints(ctx, f.mine.ID)
	if err != nil {
		t.Fatal(err)
	}
	var stored bool
	for _, c := range list {
		if c.ID == first.Reading.ID && c.Odometer == 12000 && c.Date.Format("2006-01-02") == "2026-09-01" && c.Notes != nil && *c.Notes == "After the service" {
			stored = true
		}
	}
	if !stored {
		t.Fatalf("stored readings: %+v", list)
	}

	// A retry returns the first answer; a reading is not recorded twice
	if res, again := call(writeTok, args(nil)); res.IsError || again.Reading.ID != first.Reading.ID || count() != start+1 {
		t.Fatalf("the retry returns the first answer: %+v %+v", res, again)
	}

	// What contradicts the readings around it is refused, and so is what the form refuses
	for name, bad := range map[string]map[string]any{
		"lower mileage on a later day": {"date": "2026-09-10", "odometer_km": 11000.0, "idempotency_key": "odo-2"},
		"negative odometer":            {"odometer_km": -1.0, "idempotency_key": "odo-3"},
		"beyond the range":             {"odometer_km": 2000001.0, "idempotency_key": "odo-4"},
		"not a day":                    {"date": "01/09/2026", "idempotency_key": "odo-5"},
		"long note":                    {"notes": strings.Repeat("n", 501), "idempotency_key": "odo-6"},
	} {
		if res, _ := call(writeTok, args(bad)); !res.IsError {
			t.Errorf("%s is refused", name)
		}
	}
	if count() != start+1 {
		t.Fatalf("readings after the refused calls: %d", count())
	}
}
