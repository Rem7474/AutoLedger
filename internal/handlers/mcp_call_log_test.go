package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
)

func (f *mcpFixture) tokenID(t *testing.T, userID string) string {
	t.Helper()
	tokens, err := f.repo.ListAPITokens(context.Background(), userID)
	if err != nil || len(tokens) == 0 {
		t.Fatalf("tokens of %s: %v %v", userID, tokens, err)
	}
	return tokens[0].ID
}

// The log is written after the answer is sent: wait for the number of calls the test expects.
func (f *mcpFixture) loggedCalls(t *testing.T, userID, tokenID string, want int) []models.APITokenCall {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		calls, err := f.repo.ListAPITokenCalls(context.Background(), userID, tokenID)
		if err != nil {
			t.Fatal(err)
		}
		if len(calls) >= want || time.Now().After(deadline) {
			if len(calls) != want {
				t.Fatalf("logged calls = %d, want %d: %+v", len(calls), want, calls)
			}
			return calls
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func TestMCPToolCallsAreLoggedForTheAccountPageWithoutWhatWasAsked(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	session := f.connect(t, f.readTok)

	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "list_vehicles"}); err != nil || res.IsError {
		t.Fatalf("list_vehicles: %v %+v", err, res)
	}
	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_vehicle_costs", Arguments: map[string]any{"vehicle_id": f.mine.ID}}); err != nil || res.IsError {
		t.Fatalf("get_vehicle_costs: %v %+v", err, res)
	}
	// A vehicle named by its name is looked up like an id but never written to the log.
	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_vehicle_costs", Arguments: map[string]any{"vehicle_id": "Daily"}}); err != nil || res.IsError {
		t.Fatalf("get_vehicle_costs by name: %v %+v", err, res)
	}
	if res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "get_vehicle_costs", Arguments: map[string]any{"vehicle_id": f.theirs.ID}}); err != nil || !res.IsError {
		t.Fatalf("another account's vehicle is refused: %v %+v", err, res)
	}

	calls := f.loggedCalls(t, f.owner.ID, f.tokenID(t, f.owner.ID), 4)
	byTool := map[string][]models.APITokenCall{}
	for _, c := range calls {
		byTool[c.Tool] = append(byTool[c.Tool], c)
	}
	if got := byTool["list_vehicles"]; len(got) != 1 || got[0].Outcome != "ok" || got[0].VehicleID != nil {
		t.Errorf("list_vehicles: %+v", got)
	}
	costs := byTool["get_vehicle_costs"]
	if len(costs) != 3 {
		t.Fatalf("get_vehicle_costs calls: %+v", costs)
	}
	// Most recent first: the refused call, the one by name, the one by id.
	if costs[0].Outcome != "error" || costs[0].VehicleID == nil || *costs[0].VehicleID != f.theirs.ID {
		t.Errorf("a refused call is logged as an error: %+v", costs[0])
	}
	if costs[1].Outcome != "ok" || costs[1].VehicleID != nil {
		t.Errorf("a vehicle named by its name is not logged: %+v", costs[1])
	}
	if costs[2].Outcome != "ok" || costs[2].VehicleID == nil || *costs[2].VehicleID != f.mine.ID {
		t.Errorf("a vehicle named by id is logged: %+v", costs[2])
	}
	raw, _ := json.Marshal(calls)
	for _, secret := range []string{"Daily", f.vin} {
		if strings.Contains(string(raw), secret) {
			t.Errorf("the log holds %q: %s", secret, raw)
		}
	}

	if _, err := f.repo.ListAPITokenCalls(ctx, f.theirs.UserID, f.tokenID(t, f.owner.ID)); !errors.Is(err, database.ErrNotFound) {
		t.Errorf("another account cannot read the calls of the token: %v", err)
	}
}
