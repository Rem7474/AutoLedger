package handlers

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func callWith(result mcp.Result, method string) (mcp.Result, error) {
	next := func(context.Context, string, mcp.Request) (mcp.Result, error) { return result, nil }
	return capToolResults(next)(context.Background(), method, &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: "get_vehicle_costs"}})
}

func TestCapToolResultsKeepsAResultUnderTheLimitAndReplacesOneOverIt(t *testing.T) {
	small := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("a", mcpMaxResultBytes)}}}
	if res, err := callWith(small, "tools/call"); err != nil || res != small {
		t.Fatalf("a result at the limit must pass untouched, got %v %v", res, err)
	}

	big := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("a", mcpMaxResultBytes+1)}}}
	res, err := callWith(big, "tools/call")
	got, ok := res.(*mcp.CallToolResult)
	if err != nil || !ok || !got.IsError {
		t.Fatalf("a result over the limit must become an error, got %#v %v", res, err)
	}
	text := got.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "too large") || !strings.Contains(text, "100 KB") || strings.Contains(text, "aaaa") {
		t.Errorf("the error says what to do and carries none of the data: %q", text)
	}
}

func TestCapToolResultsMeasuresTheStructuredOutputToo(t *testing.T) {
	structured := &mcp.CallToolResult{StructuredContent: map[string]string{"rows": strings.Repeat("a", mcpMaxResultBytes)}}
	res, _ := callWith(structured, "tools/call")
	if got, ok := res.(*mcp.CallToolResult); !ok || !got.IsError {
		t.Fatalf("a structured output over the limit must be refused, got %#v", res)
	}
}

func TestCapToolResultsLeavesOtherMethodsAndErrorsAlone(t *testing.T) {
	big := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: strings.Repeat("a", mcpMaxResultBytes+1)}}}
	if res, _ := callWith(big, "resources/read"); res != big {
		t.Error("only tool calls are capped")
	}
	failed := &mcp.CallToolResult{IsError: true, Content: big.Content}
	if res, _ := callWith(failed, "tools/call"); res != failed {
		t.Error("an error result is left as it is")
	}
	if res, err := callWith(nil, "tools/call"); res != nil || err != nil {
		t.Errorf("a nil result passes, got %v %v", res, err)
	}
}
