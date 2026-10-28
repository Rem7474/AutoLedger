package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/config"
	"github.com/teslacost/teslacost/internal/crypto"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/demodata"
	"github.com/teslacost/teslacost/internal/demoseed"
)

var updateMCPContract = flag.Bool("update-mcp-contract", false, "rewrite testdata/mcp_contract.golden.json")

type bearer struct{ token string }

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

// contractServer runs the real application on a fresh database and seeds it with the demo datasets and an empty
// vehicle, as the screenshots and the end-to-end suite do. It returns the base URL, a client signed in as the
// owner and the read token an assistant would hold.
func contractServer(t *testing.T) (string, *demoseed.Client, string, map[string]string) {
	t.Helper()
	base := os.Getenv("TEST_DATABASE_URL")
	if base == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	t.Setenv("DATABASE_URL", dedicatedDatabaseURL(t, base, "contract"))
	t.Setenv("AUTOLEDGER_JWT_SECRET", "contract-secret-contract-secret-contract-00")
	t.Setenv("AUTOLEDGER_ENCRYPTION_KEY", "contract-encryption-key-contract-key-0")
	t.Setenv("AUTOLEDGER_MCP_ENABLED", "true")
	t.Setenv("AUTOLEDGER_BASE_URL", "http://127.0.0.1")
	t.Setenv("ENVIRONMENT", "development")
	cfg := config.Load()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Pool.Close)
	if _, err := db.Pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		t.Fatal(err)
	}
	if err := db.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	repo := database.NewRepository(db.Pool)
	encryptor, err := crypto.NewEncryptor(cfg.AppEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	router, _, err := newApplication(cfg, nil, db.Pool, repo, encryptor)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	c := demoseed.New(srv.URL)
	if err := c.Login("contract@example.com", "Contract-Pass-123!"); err != nil {
		t.Fatal(err)
	}
	var ingest struct {
		Token string `json:"token"`
	}
	if err := c.Do("POST", "/api/auth/tokens/", map[string]string{"name": "contract seed", "scope": "ingest"}, &ingest, ""); err != nil {
		t.Fatal(err)
	}
	names := map[string]string{"empty": "Empty"}
	for _, key := range []string{"ev", "ice", "layout"} {
		ds, err := demodata.Build(key, contractSeedDay(time.Now()))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.Seed(ds, ingest.Token); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
		names[key] = ds.Vehicle.Name
	}
	var empty struct {
		ID string `json:"id"`
	}
	if err := c.Do("POST", "/api/vehicles", map[string]any{"name": "Empty", "powertrain": "ICE", "currency": "CHF", "current_odometer": 0}, &empty, ""); err != nil {
		t.Fatal(err)
	}

	var read struct {
		Token string `json:"token"`
	}
	if err := c.Do("POST", "/api/auth/tokens/", map[string]string{"name": "contract read", "scope": auth.ScopeRead}, &read, ""); err != nil {
		t.Fatal(err)
	}
	var vehicles []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := c.Do("GET", "/api/vehicles", nil, &vehicles, ""); err != nil {
		t.Fatal(err)
	}
	byName := map[string]string{}
	for _, v := range vehicles {
		byName[v.Name] = v.ID
	}
	ids := map[string]string{}
	for label, name := range names {
		if ids[label] = byName[name]; ids[label] == "" {
			t.Fatalf("vehicle %q (%s) is not seeded: %v", name, label, byName)
		}
	}
	return srv.URL, c, read.Token, ids
}

func dedicatedDatabaseURL(t *testing.T, baseURL, suffix string) string {
	t.Helper()
	cfg, err := pgxpool.ParseConfig(baseURL)
	if err != nil {
		t.Fatal(err)
	}
	name := cfg.ConnConfig.Database + "_" + suffix
	admin, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var exists bool
	if err := admin.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		if _, err := admin.Exec(context.Background(), `CREATE DATABASE "`+name+`"`); err != nil {
			t.Fatal(err)
		}
	}
	return strings.Replace(baseURL, "/"+cfg.ConnConfig.Database, "/"+name, 1)
}

// shape reduces a JSON value to its structure: the keys, the type of each value, whether it is null, and whether a
// list is empty. The demo data is dated from the day it is seeded, so the figures change with the clock while the
// shape of an answer is the contract a client codes against. A list shows the union of its elements: a field that is
// null on some records and filled on others reads as filled, whichever record comes first.
func shape(v any) any {
	switch x := v.(type) {
	case map[string]any:
		out := map[string]any{}
		for k, val := range x {
			out[k] = shape(val)
		}
		dropDayDependentParts(x, out)
		return out
	case []any:
		if len(x) == 0 {
			return []any{}
		}
		merged := shape(x[0])
		for _, el := range x[1:] {
			merged = mergeShapes(merged, shape(el))
		}
		return []any{merged}
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case nil:
		return nil
	}
	return fmt.Sprintf("%T", v)
}

// dropDayDependentParts levels out the fields whose presence follows the day the test runs on, through what the demo
// data holds in each window (the month in progress, the last months): the contract checks their type where they are
// always present, and these are not. Their own tools are covered by the integration tests of their package.
func dropDayDependentParts(answer, shaped map[string]any) {
	// An amount that falls due later in the month, null from the last days of it
	if _, ok := answer["scheduled"]; ok {
		shaped["scheduled"] = nil
	}
	// A saving at home needs a priced charge away from home within the period; the reason is given otherwise
	if _, ok := answer["home_savings"]; ok {
		shaped["home_savings"] = nil
		shaped["home_savings_unavailable"] = nil
	}
	// Entries whose price stands out from the others of the period
	if _, ok := answer["suspicious_entries"]; ok {
		shaped["suspicious_entries"] = []any{}
	}
	// A warning carries the amount it measured only for some of its codes
	if _, ok := answer["code"]; ok {
		if params, ok := shaped["params"].(map[string]any); ok {
			delete(params, "amount")
		}
	}
}

// mergeShapes combines the shapes of two records of a list: the keys of both, and the type of a field wherever one of
// them knows it.
func mergeShapes(a, b any) any {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if aok && bok {
		out := map[string]any{}
		for k, v := range am {
			out[k] = v
		}
		for k, v := range bm {
			out[k] = mergeShapes(out[k], v)
		}
		return out
	}
	al, aok := a.([]any)
	bl, bok := b.([]any)
	if aok && bok {
		switch {
		case len(al) == 0:
			return bl
		case len(bl) == 0:
			return al
		}
		return []any{mergeShapes(al[0], bl[0])}
	}
	return a
}

// toolArguments gives a tool the arguments the contract calls it with: the vehicle, nothing else.
func toolArguments(t *testing.T, tool *mcp.Tool, vehicleID string) map[string]any {
	t.Helper()
	schema, _ := json.Marshal(tool.InputSchema)
	var parsed struct {
		Properties map[string]any `json:"properties"`
		Required   []string       `json:"required"`
	}
	_ = json.Unmarshal(schema, &parsed)
	args := map[string]any{}
	extra := map[string]any{"month": previousMonth(time.Now()), "category": "energy", "energy_kwh": 10, "duration_minutes": 60}
	for _, name := range parsed.Required {
		value, ok := extra[name]
		if name == "vehicle_id" {
			continue
		}
		if !ok {
			t.Fatalf("%s requires %q: give it a value in toolArguments", tool.Name, name)
		}
		args[name] = value
	}
	if _, ok := parsed.Properties["vehicle_id"]; ok {
		args["vehicle_id"] = vehicleID
	}
	return args
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) (map[string]any, bool) {
	t.Helper()
	res, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	raw, _ := json.Marshal(res.StructuredContent)
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil && !res.IsError {
		t.Fatalf("%s: structured content: %v", name, err)
	}
	return out, res.IsError
}

// Every tool answers for every kind of vehicle (electric with a lease, combustion with a purchase, a plug-in hybrid
// in another currency, a vehicle with no record) with the structure the golden file records, through the protocol
// and the real router.
func TestMCPContractOnTheDemoVehicles(t *testing.T) {
	url, c, token, ids := contractServer(t)
	session, err := mcp.NewClient(&mcp.Implementation{Name: "contract", Version: "1"}, nil).Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint: url + "/mcp", HTTPClient: &http.Client{Transport: bearer{token}}, DisableStandaloneSSE: true, MaxRetries: -1,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	t.Run("structure", func(t *testing.T) { checkStructure(t, session, ids) })
	t.Run("dashboard parity", func(t *testing.T) { checkDashboardParity(t, c, session, ids) })
	t.Run("month breakdown parity", func(t *testing.T) { checkMonthBreakdownParity(t, session, ids) })
}

func checkStructure(t *testing.T, session *mcp.ClientSession, ids map[string]string) {
	vehicles := []string{"ev", "ice", "layout", "empty"}
	listed, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]any{}
	for _, tool := range listed.Tools {
		for _, vehicle := range vehicles {
			args := toolArguments(t, tool, ids[vehicle])
			if _, perVehicle := args["vehicle_id"]; !perVehicle && vehicle != vehicles[0] {
				continue
			}
			out, isErr := callTool(t, session, tool.Name, args)
			key := tool.Name + " / " + vehicle
			if _, perVehicle := args["vehicle_id"]; !perVehicle {
				key = tool.Name
			}
			if isErr {
				t.Errorf("%s answered with an error", key)
				continue
			}
			got[key] = shape(out)
		}
	}
	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')

	const path = "testdata/mcp_contract.golden.json"
	if *updateMCPContract {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (generate it with -update-mcp-contract)", err)
	}
	if string(want) != string(encoded) {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		t.Errorf("the structure of a tool answer changed (%d answers); review the diff of %s and regenerate with -update-mcp-contract", len(keys), path)
	}
}

func amountOf(t *testing.T, v any) float64 {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("not an amount: %v", v)
	}
	f, err := strconv.ParseFloat(fmt.Sprint(m["amount"]), 64)
	if err != nil {
		t.Fatalf("amount %v: %v", m["amount"], err)
	}
	return f
}

// The figures an assistant reads are the ones the dashboard shows for the same vehicle.
func checkDashboardParity(t *testing.T, c *demoseed.Client, session *mcp.ClientSession, ids map[string]string) {
	for label, id := range ids {
		name := label
		var tco map[string]any
		if err := c.Do("GET", "/api/vehicles/"+id+"/tco", nil, &tco, ""); err != nil {
			t.Fatalf("%s: tco: %v", name, err)
		}
		costs, isErr := callTool(t, session, "get_vehicle_costs", map[string]any{"vehicle_id": id})
		if isErr {
			t.Fatalf("%s: get_vehicle_costs failed", name)
		}
		since := costs["since_purchase"].(map[string]any)["costs"].(map[string]any)
		if total, _ := tco["total_cost"].(float64); label != "empty" && total == 0 {
			t.Fatalf("%s: the demo vehicle has no cost, so the comparison proves nothing", label)
		}
		for mcpField, tcoField := range map[string]string{"total": "total_cost", "energy": "energy_cost", "tolls": "tolls_cost", "insurance": "insurance_cost", "financing": "financing_cost", "tires": "tires_cost"} {
			want, _ := tco[tcoField].(float64)
			if diff := amountOf(t, since[mcpField]) - want; diff > 0.005 || diff < -0.005 {
				t.Errorf("%s: %s is %v for the assistant and %v on the dashboard", name, mcpField, since[mcpField], want)
			}
		}
	}
}

// The records get_month_breakdown lists add up to the figure get_monthly_costs gives for the same month.
func checkMonthBreakdownParity(t *testing.T, session *mcp.ClientSession, ids map[string]string) {
	month := previousMonth(time.Now())
	for label, id := range ids {
		series, isErr := callTool(t, session, "get_monthly_costs", map[string]any{"vehicle_id": id, "from": month, "to": month})
		months, _ := series["months"].([]any)
		if isErr || (len(months) != 1 && label != "empty") {
			t.Fatalf("%s: monthly costs: %v", label, series)
		}
		if len(months) == 0 {
			// A vehicle with no record has no month: its breakdown is empty too.
			breakdown, _ := callTool(t, session, "get_month_breakdown", map[string]any{"vehicle_id": id, "month": month, "category": "energy"})
			if entries, _ := breakdown["entries"].([]any); len(entries) != 0 || amountOf(t, breakdown["total"]) != 0 {
				t.Errorf("%s: a vehicle with no record has nothing to break down: %v", label, breakdown)
			}
			continue
		}
		for category, field := range map[string]string{"energy": "energy", "insurance": "insurance"} {
			breakdown, isErr := callTool(t, session, "get_month_breakdown", map[string]any{"vehicle_id": id, "month": month, "category": category})
			if isErr {
				t.Fatalf("%s: month breakdown %s failed", label, category)
			}
			want := amountOf(t, months[0].(map[string]any)["costs"].(map[string]any)[field])
			if got := amountOf(t, breakdown["total"]); got != want {
				t.Errorf("%s %s %s: breakdown says %v, monthly costs say %v", label, month, category, got, want)
			}
		}
	}
}

// contractSeedDay is the latest Sunday on or before now. The demo data follows the weekday of its last day (which
// days carry a charge, a trip, a journey), and so does the structure of some answers (a home saving needs a priced
// charge away from home): seeding from a fixed weekday keeps the contract the same whatever day it runs.
func contractSeedDay(now time.Time) time.Time {
	now = now.UTC()
	return now.AddDate(0, 0, -int(now.Weekday()))
}

// previousMonth is the month before now's as "2006-01": AddDate(0, -1, 0) from the 31st lands in the current month.
func previousMonth(now time.Time) string {
	now = now.UTC()
	return time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC).Format("2006-01")
}
