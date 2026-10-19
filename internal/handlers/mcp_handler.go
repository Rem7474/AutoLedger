package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/teslacost/teslacost/internal/apierror"
	appAuth "github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/middleware"
)

// mcpInstructions is what a client shows its model about the server. It states the conventions every tool's output
// follows, so a model does not have to infer them from field names.
const mcpInstructions = `AutoLedger tracks what vehicles cost. These tools only read.
Amounts are decimal strings in the currency of their vehicle and are never converted between currencies: do not add amounts of different currencies together.
Distances are in kilometres. A null value means the figure is unknown, not zero.
Costs are the cash spent, the purchase of the vehicle excluded. Call get_vehicle_data_gaps before drawing conclusions from a vehicle's costs: it says what the records are missing.
list_vehicle_trips returns the addresses the owner recorded for each trip: they are personal, so repeat them only as far as the question needs.`

// MCPHandler serves the Model Context Protocol endpoint (Streamable HTTP, stateless) on top of the read API: the same
// narrow views, so an assistant sees no more than a read token does through /api/integrations/v1. It is read-only.
type MCPHandler struct {
	read    *IntegrationReadHandler
	tokens  middleware.APITokenValidator
	issuer  string
	server  *mcp.Server
	handler http.Handler
}

// NewMCPHandler builds the endpoint. baseURL is the instance's public URL: the resource is baseURL + "/mcp" and the
// authorization server is the instance itself.
func NewMCPHandler(read *IntegrationReadHandler, tokens middleware.APITokenValidator, baseURL, version string) *MCPHandler {
	h := &MCPHandler{read: read, tokens: tokens, issuer: trimTrailingSlash(baseURL)}
	h.server = mcp.NewServer(&mcp.Implementation{Name: "autoledger", Version: version}, &mcp.ServerOptions{Instructions: mcpInstructions})
	h.addTools()

	streamable := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return h.server }, &mcp.StreamableHTTPOptions{
		Stateless:    true,
		JSONResponse: true,
		// The SDK refuses a request that reaches a loopback listener with another Host, which is what a reverse proxy
		// on the same machine sends. The check guards against DNS rebinding of a server that trusts the caller's
		// network position; this one trusts only a bearer token, which a rebinding page does not have.
		DisableLocalhostProtection: true,
	})
	h.handler = auth.RequireBearerToken(h.verifyToken, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: h.issuer + "/.well-known/oauth-protected-resource/mcp",
		Scopes:              []string{appAuth.ScopeRead},
		// The validator already refuses an expired token and does not report the expiry.
		AllowMissingExpiration: true,
	})(streamable)
	return h
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

// Endpoint serves MCP requests. A request without a valid read-scope token gets a 401 whose WWW-Authenticate header
// points to the protected resource metadata, which is how a client finds the authorization server.
func (h *MCPHandler) Endpoint() http.Handler { return h.handler }

// ResourceMetadata serves the protected resource metadata (RFC 9728).
func (h *MCPHandler) ResourceMetadata() http.Handler {
	return auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               h.issuer + "/mcp",
		AuthorizationServers:   []string{h.issuer},
		ScopesSupported:        []string{appAuth.ScopeRead},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "AutoLedger",
	})
}

// verifyToken accepts an API token only: a session access token is not meant to leave the browser.
func (h *MCPHandler) verifyToken(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	if !appAuth.IsAPIToken(token) {
		return nil, auth.ErrInvalidToken
	}
	userID, _, scope, err := h.tokens.ValidateAPIToken(ctx, appAuth.HashAPIToken(token))
	if err != nil {
		return nil, auth.ErrInvalidToken
	}
	return &auth.TokenInfo{UserID: userID, Scopes: []string{scope}}, nil
}

type mcpNoInput struct{}

type mcpVehicleInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles"`
}

type mcpTripsInput struct {
	VehicleID string `json:"vehicle_id" jsonschema:"the id of a vehicle, as returned by list_vehicles"`
	Search    string `json:"search,omitempty" jsonschema:"text to find in the departure or arrival address, for instance a city or a street"`
	From      string `json:"from,omitempty" jsonschema:"first day to include, YYYY-MM-DD (UTC)"`
	To        string `json:"to,omitempty" jsonschema:"last day to include, YYYY-MM-DD (UTC)"`
	Limit     int    `json:"limit,omitempty" jsonschema:"how many trips to return, 20 by default and 50 at most"`
}

type mcpVehiclesOutput struct {
	Vehicles []IntegrationVehicleV1 `json:"vehicles"`
}

func (h *MCPHandler) addTools() {
	readOnly := &mcp.ToolAnnotations{ReadOnlyHint: true, OpenWorldHint: ptr(false)}

	mcp.AddTool(h.server, &mcp.Tool{
		Name:        "list_vehicles",
		Description: "List the vehicles of the account: id, name, make, model, powertrain, currency and odometer. Start here to get a vehicle id.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ mcpNoInput) (*mcp.CallToolResult, mcpVehiclesOutput, error) {
		vehicles, err := h.read.Vehicles(ctx, mcpUserID(req))
		if err != nil {
			return nil, mcpVehiclesOutput{}, mcpFailure(ctx, "list_vehicles", err, "Could not list the vehicles")
		}
		return nil, mcpVehiclesOutput{Vehicles: vehicles}, nil
	})

	mcp.AddTool(h.server, &mcp.Tool{
		Name: "get_vehicle_costs",
		Description: "What a vehicle cost over its last 12 months and since it was bought, by category (energy, tolls, maintenance, insurance, financing, tires, other), " +
			"with the cost per kilometre and a completeness score. Amounts are in the vehicle's currency.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpVehicleInput) (*mcp.CallToolResult, IntegrationCosts, error) {
		costs, err := h.read.Costs(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationCosts{}, mcpFailure(ctx, "get_vehicle_costs", err, "Could not compute the costs")
		}
		return nil, *costs, nil
	})

	mcp.AddTool(h.server, &mcp.Tool{
		Name: "list_vehicle_trips",
		Description: "The trips of a vehicle, most recent first: when, from and to (the addresses the owner recorded, null when unknown), distance, duration, energy, " +
			"and what tolls and parking cost. Filter with search (text in an address, to find a trip \"to Lyon\") and a period. Only the first trips are listed: matching says how many fit.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpTripsInput) (*mcp.CallToolResult, IntegrationTrips, error) {
		trips, err := h.read.Trips(ctx, mcpUserID(req), in.VehicleID, IntegrationTripsQuery{Search: in.Search, From: in.From, To: in.To, Limit: in.Limit})
		if err != nil {
			return nil, IntegrationTrips{}, mcpFailure(ctx, "list_vehicle_trips", err, "Could not list the trips")
		}
		return nil, *trips, nil
	})

	mcp.AddTool(h.server, &mcp.Tool{
		Name: "get_vehicle_data_gaps",
		Description: "What is missing from a vehicle's records: odometer gaps and anomalies (the 20 most recent are listed), charges without a cost, " +
			"unqualified drives, untracked distance, missing insurance or acquisition. Use it to say how far the costs can be trusted.",
		Annotations: readOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in mcpVehicleInput) (*mcp.CallToolResult, IntegrationDataGaps, error) {
		gaps, err := h.read.DataGaps(ctx, mcpUserID(req), in.VehicleID)
		if err != nil {
			return nil, IntegrationDataGaps{}, mcpFailure(ctx, "get_vehicle_data_gaps", err, "Could not check the data")
		}
		return nil, *gaps, nil
	})
}

func ptr[T any](v T) *T { return &v }

// mcpUserID is the account the request's token belongs to. The endpoint refuses a request without one.
func mcpUserID(req *mcp.CallToolRequest) string {
	if info := req.GetExtra().TokenInfo; info != nil {
		return info.UserID
	}
	return ""
}

// mcpFailure turns a failed read into the error the model sees. A vehicle the account cannot see is reported as such;
// anything else is logged and answered generically, so no internal detail reaches the assistant.
func mcpFailure(ctx context.Context, tool string, err error, fallback string) error {
	if errors.Is(err, errIntegrationVehicleNotFound) {
		return errors.New("vehicle not found: use list_vehicles to get a valid vehicle_id")
	}
	if apiErr, ok := apierror.As(err); ok {
		return errors.New(apiErr.Error())
	}
	slog.ErrorContext(ctx, fallback, "component", "mcp", "tool", tool, "error", err)
	return errors.New(fallback)
}
