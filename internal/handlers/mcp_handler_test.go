package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
	"github.com/teslacost/teslacost/internal/models"
	"github.com/teslacost/teslacost/internal/services"
)

// bearerTransport sends a fixed token, the way an MCP client does once it has been authorized.
type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if b.token != "" {
		r.Header.Set("Authorization", "Bearer "+b.token)
	}
	return http.DefaultTransport.RoundTrip(r)
}

type mcpFixture struct {
	server   *httptest.Server
	repo     *database.Repository
	mine     *models.Vehicle
	theirs   *models.Vehicle
	vin      string
	readTok  string
	owner    *models.User
	otherTok string
}

func newMCPFixture(t *testing.T) *mcpFixture {
	t.Helper()
	repo := authTestRepo(t)
	ctx := context.Background()
	owner, err := repo.CreateUser(ctx, "mcp-owner@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	other, err := repo.CreateUser(ctx, "mcp-other@example.com", "hash")
	if err != nil {
		t.Fatal(err)
	}
	f := &mcpFixture{repo: repo, vin: "VF1SECRETVIN000001", owner: owner}
	f.mine = &models.Vehicle{UserID: owner.ID, Name: "Daily", Make: "Some make", Model: "Some model", Vin: &f.vin, Currency: "GBP",
		CurrentOdometer: 4200, TeslaMateAuthType: models.AuthModeNone}
	f.theirs = &models.Vehicle{UserID: other.ID, Name: "Not mine", Currency: "EUR", TeslaMateAuthType: models.AuthModeNone}
	for _, v := range []*models.Vehicle{f.mine, f.theirs} {
		if err := repo.CreateVehicle(ctx, v); err != nil {
			t.Fatal(err)
		}
	}
	f.readTok = f.newToken(t, owner.ID, auth.ScopeRead)
	f.otherTok = f.newToken(t, other.ID, auth.ScopeRead)

	read := NewIntegrationReadHandler(repo, services.NewTCOService(repo.Pool(), "UTC"))
	f.server = httptest.NewServer(NewMCPHandler(read, repo, "https://ledger.example/", "test").Endpoint())
	t.Cleanup(f.server.Close)
	return f
}

func (f *mcpFixture) newToken(t *testing.T, userID, scope string) string {
	t.Helper()
	raw, prefix, hash, err := auth.GenerateAPIToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.repo.CreateAPIToken(context.Background(), userID, "mcp test", scope, hash, prefix, nil); err != nil {
		t.Fatal(err)
	}
	return raw
}

func (f *mcpFixture) connect(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{
		Endpoint:             f.server.URL,
		HTTPClient:           &http.Client{Transport: bearerTransport{token: token}},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func (f *mcpFixture) post(token string) *http.Response { return f.postAs(token, "") }

// postAs sends the request with another Host header, as a reverse proxy on the same machine does.
func (f *mcpFixture) postAs(token, host string) *http.Response {
	req, _ := http.NewRequest(http.MethodPost, f.server.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if host != "" {
		req.Host = host
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func TestMCPEndpointOnlyOpensToAReadTokenAndSaysWhereToAuthorize(t *testing.T) {
	f := newMCPFixture(t)
	ingest := f.newToken(t, f.owner.ID, auth.ScopeIngest)

	for name, token := range map[string]string{"no credential": "", "unknown token": "al_live_" + strings.Repeat("0", 48), "not an API token": "eyJhbGciOiJIUzI1NiJ9.e30.x"} {
		resp := f.post(token)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status %d, want 401", name, resp.StatusCode)
		}
		if got := resp.Header.Get("WWW-Authenticate"); !strings.Contains(got, `resource_metadata="https://ledger.example/.well-known/oauth-protected-resource/mcp"`) {
			t.Errorf("%s: a client finds the authorization server through WWW-Authenticate, got %q", name, got)
		}
	}

	resp := f.post(ingest)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("a token that can push events must not read through MCP: status %d, want 403", resp.StatusCode)
	}

	resp = f.post(f.readTok)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("a read token must be accepted: status %d", resp.StatusCode)
	}

	resp = f.postAs(f.readTok, "ledger.example")
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("a request a reverse proxy forwards to a loopback listener keeps its public Host: status %d", resp.StatusCode)
	}
}

func TestMCPToolsAreReadOnlyAndServeTheAccountsVehiclesWithoutIdentifiers(t *testing.T) {
	f := newMCPFixture(t)
	ctx := context.Background()
	session := f.connect(t, f.readTok)

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
		if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
			t.Errorf("%s must be declared read-only", tool.Name)
		}
	}
	if strings.Join(names, ",") != "get_vehicle_costs,get_vehicle_data_gaps,list_vehicle_trips,list_vehicles" {
		t.Errorf("tools: %v", names)
	}

	call := func(name string, args map[string]any) *mcp.CallToolResult {
		res, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return res
	}
	text := func(res *mcp.CallToolResult) string {
		raw, _ := json.Marshal(res.StructuredContent)
		return string(raw)
	}

	list := call("list_vehicles", nil)
	var listed mcpVehiclesOutput
	if err := json.Unmarshal([]byte(text(list)), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Vehicles) != 1 || listed.Vehicles[0].ID != f.mine.ID || listed.Vehicles[0].Currency != "GBP" ||
		listed.Vehicles[0].Odometer != (IntegrationDistance{Value: 4200, Unit: "km"}) {
		t.Errorf("only the account's own vehicle is listed: %+v", listed.Vehicles)
	}

	for _, tool := range []string{"get_vehicle_costs", "get_vehicle_data_gaps"} {
		res := call(tool, map[string]any{"vehicle_id": f.mine.ID})
		if res.IsError {
			t.Fatalf("%s on an own vehicle failed: %+v", tool, res.Content)
		}
		if body := text(res); strings.Contains(body, f.vin) || strings.Contains(body, f.theirs.ID) {
			t.Errorf("%s leaked an identifier: %s", tool, body)
		}
		res = call(tool, map[string]any{"vehicle_id": f.theirs.ID})
		if !res.IsError {
			t.Errorf("%s on another account's vehicle must be an error", tool)
		}
	}

	var costs IntegrationCosts
	if err := json.Unmarshal([]byte(text(call("get_vehicle_costs", map[string]any{"vehicle_id": f.mine.ID}))), &costs); err != nil {
		t.Fatal(err)
	}
	if costs.Currency != "GBP" || costs.SincePurchase.Costs.Total.Currency != "GBP" {
		t.Errorf("every amount carries the vehicle's currency: %+v", costs)
	}

	// The other account sees none of it.
	otherList, err := f.connect(t, f.otherTok).CallTool(ctx, &mcp.CallToolParams{Name: "list_vehicles"})
	if err != nil {
		t.Fatal(err)
	}
	var theirs mcpVehiclesOutput
	if err := json.Unmarshal([]byte(text(otherList)), &theirs); err != nil {
		t.Fatal(err)
	}
	if len(theirs.Vehicles) != 1 || theirs.Vehicles[0].ID != f.theirs.ID {
		t.Errorf("the other account's list: %+v", theirs.Vehicles)
	}
}

func TestMCPRevokedTokenStopsWorking(t *testing.T) {
	f := newMCPFixture(t)
	tokens, err := f.repo.ListAPITokens(context.Background(), f.owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range tokens {
		if err := f.repo.RevokeAPIToken(context.Background(), f.owner.ID, tok.ID); err != nil {
			t.Fatal(err)
		}
	}
	resp := f.post(f.readTok)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("a revoked token: status %d, want 401", resp.StatusCode)
	}
}

func TestMCPResourceMetadataNamesTheAuthorizationServer(t *testing.T) {
	h := NewMCPHandler(nil, nil, "https://ledger.example/", "test")
	rec := httptest.NewRecorder()
	h.ResourceMetadata().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/mcp", nil))

	var meta struct {
		Resource             string   `json:"resource"`
		AuthorizationServers []string `json:"authorization_servers"`
		ScopesSupported      []string `json:"scopes_supported"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Resource != "https://ledger.example/mcp" || len(meta.AuthorizationServers) != 1 || meta.AuthorizationServers[0] != "https://ledger.example" ||
		len(meta.ScopesSupported) != 1 || meta.ScopesSupported[0] != "read" {
		t.Errorf("metadata: %s", rec.Body.String())
	}
}
