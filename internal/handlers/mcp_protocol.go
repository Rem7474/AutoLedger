package handlers

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/docs"
)

// Resource addresses. The vehicle template takes the id a tool returns as vehicle_id.
const (
	mcpGlossaryURI        = "autoledger://glossary"
	mcpAccountURI         = "autoledger://account"
	mcpVehicleURIPrefix   = "autoledger://vehicles/"
	mcpVehicleURITemplate = mcpVehicleURIPrefix + "{vehicle_id}"
	mcpResourceJSON       = "application/json"
	mcpResourceMarkdown   = "text/markdown"
)

func mcpTokenUser(extra *mcp.RequestExtra) string {
	if extra != nil && extra.TokenInfo != nil {
		return extra.TokenInfo.UserID
	}
	return ""
}

func jsonResource(uri string, v any) (*mcp.ReadResourceResult, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: mcpResourceJSON, Text: string(body)}}}, nil
}

// addResources registers what a client can read without calling a tool: the glossary, how the account reads its
// figures and one document per vehicle.
func (h *MCPHandler) addResources() {
	h.server.AddResource(&mcp.Resource{
		URI: mcpGlossaryURI, Name: "glossary", Title: "Cost and consumption terms", MIMEType: mcpResourceMarkdown,
		Description: "What cash paid, economic cost, full cost, running cost, consumption at the wheel and at the plug, estimated energy and drive cost mean. Read it before comparing or explaining two figures.",
	}, func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: mcpGlossaryURI, MIMEType: mcpResourceMarkdown, Text: docs.CostTerms}}}, nil
	})

	h.server.AddResource(&mcp.Resource{
		URI: mcpAccountURI, Name: "account", Title: "Units, calendar and currencies", MIMEType: mcpResourceJSON,
		Description: "The distance and volume units the owner reads, the reporting timezone, the month in progress and the currency of each vehicle.",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		account, err := h.read.Account(ctx, mcpTokenUser(req.GetExtra()))
		if err != nil {
			return nil, mcpFailure(ctx, "resource:account", err, "Could not read the account")
		}
		return jsonResource(mcpAccountURI, account)
	})

	h.server.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: mcpVehicleURITemplate, Name: "vehicle", Title: "A vehicle", MIMEType: mcpResourceJSON,
		Description: "One vehicle of the account: name, make, model, powertrain, currency, odometer, role, how its records arrive and its ownership.",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		id, ok := strings.CutPrefix(uri, mcpVehicleURIPrefix)
		if !ok || id == "" {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		vehicles, err := h.read.Vehicles(ctx, mcpTokenUser(req.GetExtra()))
		if err != nil {
			return nil, mcpFailure(ctx, "resource:vehicle", err, "Could not read the vehicle")
		}
		for _, v := range vehicles {
			if v.ID == id {
				return jsonResource(uri, v)
			}
		}
		return nil, mcp.ResourceNotFoundError(uri)
	})
}

// complete suggests values for the arguments a client fills in: the vehicle of the template and of the prompts.
func (h *MCPHandler) complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	result := &mcp.CompleteResult{Completion: mcp.CompletionResultDetails{Values: []string{}}}
	p := req.Params
	if p == nil || p.Ref == nil {
		return result, nil
	}
	vehicleArgument := (p.Ref.Type == "ref/resource" && p.Ref.URI == mcpVehicleURITemplate && p.Argument.Name == "vehicle_id") ||
		(p.Ref.Type == "ref/prompt" && p.Argument.Name == "vehicle")
	if !vehicleArgument {
		return result, nil
	}
	vehicles, err := h.read.Vehicles(ctx, mcpTokenUser(req.GetExtra()))
	if err != nil {
		return nil, mcpFailure(ctx, "completion", err, "Could not list the vehicles")
	}
	prefix := strings.ToLower(strings.TrimSpace(p.Argument.Value))
	for _, v := range vehicles {
		// A resource takes the id, a prompt takes the name or the id: the name reads better
		value := v.ID
		if p.Ref.Type == "ref/prompt" {
			value = v.Name
		}
		if strings.HasPrefix(strings.ToLower(value), prefix) {
			result.Completion.Values = append(result.Completion.Values, value)
		}
	}
	result.Completion.Total = len(result.Completion.Values)
	return result, nil
}

// mcpPromptSpec is a prompt that sends the model through the tools in a fixed order. It carries no figure: the
// numbers come from the tools when the prompt is used.
type mcpPromptSpec struct {
	name, title, description string
	arguments                []*mcp.PromptArgument
	text                     func(args map[string]string) string
}

var (
	mcpVehicleArgument = &mcp.PromptArgument{Name: "vehicle", Description: "the vehicle's name or id; leave empty to choose among the account's vehicles"}
	mcpMonthArgument   = &mcp.PromptArgument{Name: "month", Description: "a month, YYYY-MM; the previous full month by default"}
	mcpYearArgument    = &mcp.PromptArgument{Name: "year", Description: "a calendar year, YYYY; the previous year by default"}
)

func vehicleClause(args map[string]string) string {
	if v := strings.TrimSpace(args["vehicle"]); v != "" {
		return "the vehicle " + v
	}
	return "the account's vehicle (call list_vehicles and ask which one if there are several)"
}

func periodClause(args map[string]string, key, fallback string) string {
	if v := strings.TrimSpace(args[key]); v != "" {
		return v
	}
	return fallback
}

var mcpPrompts = []mcpPromptSpec{
	{
		name: "monthly_review", title: "Review a month", description: "Explain what a month cost, what changed against the previous ones and what to complete.",
		arguments: []*mcp.PromptArgument{mcpVehicleArgument, mcpMonthArgument},
		text: func(a map[string]string) string {
			return "Review " + periodClause(a, "month", "the previous full month") + " for " + vehicleClause(a) + ". " +
				"Read the autoledger://glossary and autoledger://account resources first. Call get_monthly_costs for the last twelve months and compare that month with the others, category by category; " +
				"call get_reminders for what is due; call get_vehicle_data_gaps and say how far the figures can be trusted. " +
				"Point out the deviations and what the owner should complete, without repeating every figure."
		},
	},
	{
		name: "what_is_missing", title: "What is missing", description: "List what to complete in the records, the most valuable first.",
		arguments: []*mcp.PromptArgument{mcpVehicleArgument},
		text: func(a map[string]string) string {
			return "Tell what is missing from the records of " + vehicleClause(a) + ". " +
				"Call get_vehicle_data_gaps and present its ranked list, the biggest gap first, with what completing each one changes. " +
				"Call get_data_sources if recent records are missing, and say whether the cause is the synchronization or entries not made."
		},
	},
	{
		name: "lease_check", title: "Check a lease or loan", description: "Compare what a lease or loan costs with the distance driven and project it to the end of the contract.",
		arguments: []*mcp.PromptArgument{mcpVehicleArgument},
		text: func(a map[string]string) string {
			return "Check the contract of " + vehicleClause(a) + ". " +
				"Call get_vehicle_ownership for the contract, the allowed distance and the resale estimate, get_odometer for where the vehicle stands, and get_monthly_costs for the recent pace. " +
				"Project the distance and the cost to the end of the contract at that pace and say whether the allowance will be exceeded and what that would cost. State your assumptions."
		},
	},
	{
		name: "receipt_check", title: "Check the receipts", description: "List the expenses that have no receipt attached.",
		arguments: []*mcp.PromptArgument{mcpVehicleArgument, mcpYearArgument},
		text: func(a map[string]string) string {
			return "List the expenses of " + periodClause(a, "year", "the previous year") + " for " + vehicleClause(a) + " that have no receipt. " +
				"Call list_expenses over that year, sorted by amount, and keep the entries whose has_receipt is false; ignore those where it is null, they cannot carry one. " +
				"Group them by category with their total and say which are worth chasing first."
		},
	},
	{
		name: "tax_year", title: "Professional and personal split of a year", description: "Prepare the split of a year between professional and personal use from tags.",
		arguments: []*mcp.PromptArgument{mcpVehicleArgument, mcpYearArgument},
		text: func(a map[string]string) string {
			return "Prepare the professional and personal split of " + periodClause(a, "year", "the previous year") + " for " + vehicleClause(a) + ". " +
				"Call get_drive_summary from the first to the last day of the year for the distance and allocated cost of each tag and of the untagged drives, and list_vehicle_trips over the same days for the trips behind them. " +
				"Say how much of the year is untagged (untagged_drives) and how that limits the split, and that drives_to_qualify counts motorway drives whose tolls may be missing. A drive with several tags counts in each: do not add the rows when shared_drives is above zero. Do not apply any tax scale of your own; when the owner wants the mileage allowance, call get_mileage_report over the same days and say that its scales are the rates the owner typed, not an official table."
		},
	},
}

func (h *MCPHandler) addPrompts() {
	for _, spec := range mcpPrompts {
		h.server.AddPrompt(&mcp.Prompt{Name: spec.name, Title: spec.title, Description: spec.description, Arguments: spec.arguments},
			func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				return &mcp.GetPromptResult{
					Description: spec.description,
					Messages:    []*mcp.PromptMessage{{Role: "user", Content: &mcp.TextContent{Text: spec.text(req.Params.Arguments)}}},
				}, nil
			})
	}
}
