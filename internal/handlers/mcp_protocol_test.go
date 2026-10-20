package handlers

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPServesTheGlossaryTheAccountAndOnlyItsOwnVehiclesAsResources(t *testing.T) {
	f := newMCPFixture(t)
	session := f.connect(t, f.readTok)
	ctx := context.Background()

	list, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	uris := []string{}
	for _, r := range list.Resources {
		uris = append(uris, r.URI)
	}
	if strings.Join(uris, ",") != "autoledger://account,autoledger://glossary" {
		t.Errorf("resources: %v", uris)
	}
	templates, err := session.ListResourceTemplates(ctx, nil)
	if err != nil || len(templates.ResourceTemplates) != 1 || templates.ResourceTemplates[0].URITemplate != "autoledger://vehicles/{vehicle_id}" {
		t.Errorf("resource templates: %+v %v", templates, err)
	}

	glossary, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "autoledger://glossary"})
	if err != nil || !strings.Contains(glossary.Contents[0].Text, "Full cost") || glossary.Contents[0].MIMEType != "text/markdown" {
		t.Fatalf("glossary: %+v %v", glossary, err)
	}

	account, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "autoledger://account"})
	if err != nil {
		t.Fatal(err)
	}
	var acc IntegrationAccount
	if err := json.Unmarshal([]byte(account.Contents[0].Text), &acc); err != nil {
		t.Fatal(err)
	}
	if acc.DistanceUnit == "" || acc.ReportingZone == "" || len(acc.MonthInProgress) != 7 || len(acc.Vehicles) != 1 || acc.Vehicles[0].ID != f.mine.ID || acc.Vehicles[0].Currency != "GBP" ||
		strings.Contains(account.Contents[0].Text, f.theirs.ID) || strings.Contains(account.Contents[0].Text, "@") {
		t.Errorf("account: %s", account.Contents[0].Text)
	}

	vehicle, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "autoledger://vehicles/" + f.mine.ID})
	if err != nil || !strings.Contains(vehicle.Contents[0].Text, f.mine.ID) || strings.Contains(vehicle.Contents[0].Text, f.vin) {
		t.Fatalf("vehicle: %+v %v", vehicle, err)
	}
	for _, uri := range []string{"autoledger://vehicles/" + f.theirs.ID, "autoledger://vehicles/", "autoledger://vehicles/unknown"} {
		if _, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri}); err == nil {
			t.Errorf("%s must not be readable", uri)
		}
	}
}

func TestMCPPromptsOnlySendTheModelThroughTheTools(t *testing.T) {
	f := newMCPFixture(t)
	session := f.connect(t, f.readTok)
	ctx := context.Background()

	list, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for _, p := range list.Prompts {
		names = append(names, p.Name)
	}
	if strings.Join(names, ",") != "lease_check,monthly_review,receipt_check,tax_year,what_is_missing" {
		t.Fatalf("prompts: %v", names)
	}

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	known := map[string]bool{}
	for _, tool := range tools.Tools {
		known[tool.Name] = true
	}
	for _, name := range names {
		got, err := session.GetPrompt(ctx, &mcp.GetPromptParams{Name: name, Arguments: map[string]string{"vehicle": "Daily"}})
		if err != nil || len(got.Messages) != 1 {
			t.Fatalf("%s: %+v %v", name, got, err)
		}
		text := got.Messages[0].Content.(*mcp.TextContent).Text
		if !strings.Contains(text, "the vehicle Daily") {
			t.Errorf("%s ignores its vehicle: %s", name, text)
		}
		// Every tool a prompt names exists
		for _, word := range strings.FieldsFunc(text, func(r rune) bool { return !(r == '_' || r >= 'a' && r <= 'z') }) {
			if (strings.HasPrefix(word, "get_") || strings.HasPrefix(word, "list_")) && !known[word] {
				t.Errorf("%s names the tool %q, which does not exist", name, word)
			}
		}
	}
	noVehicle, err := session.GetPrompt(ctx, &mcp.GetPromptParams{Name: "monthly_review", Arguments: map[string]string{"month": "2026-09"}})
	if err != nil || !strings.Contains(noVehicle.Messages[0].Content.(*mcp.TextContent).Text, "2026-09") || !strings.Contains(noVehicle.Messages[0].Content.(*mcp.TextContent).Text, "ask which one") {
		t.Errorf("an omitted vehicle is asked for: %+v %v", noVehicle, err)
	}
}

func TestMCPCompletesTheVehicleOfTheAccountOnly(t *testing.T) {
	f := newMCPFixture(t)
	session := f.connect(t, f.readTok)
	ctx := context.Background()

	byName, err := session.Complete(ctx, &mcp.CompleteParams{Ref: &mcp.CompleteReference{Type: "ref/prompt", Name: "monthly_review"}, Argument: mcp.CompleteParamsArgument{Name: "vehicle", Value: "da"}})
	if err != nil || len(byName.Completion.Values) != 1 || byName.Completion.Values[0] != "Daily" {
		t.Errorf("a prompt completes the name: %+v %v", byName, err)
	}
	byID, err := session.Complete(ctx, &mcp.CompleteParams{Ref: &mcp.CompleteReference{Type: "ref/resource", URI: "autoledger://vehicles/{vehicle_id}"}, Argument: mcp.CompleteParamsArgument{Name: "vehicle_id", Value: ""}})
	if err != nil || len(byID.Completion.Values) != 1 || byID.Completion.Values[0] != f.mine.ID {
		t.Errorf("the template completes the id of this account's vehicles: %+v %v", byID, err)
	}
	other, err := session.Complete(ctx, &mcp.CompleteParams{Ref: &mcp.CompleteReference{Type: "ref/prompt", Name: "monthly_review"}, Argument: mcp.CompleteParamsArgument{Name: "month", Value: "2026"}})
	if err != nil || len(other.Completion.Values) != 0 {
		t.Errorf("an argument without suggestion: %+v %v", other, err)
	}
}
