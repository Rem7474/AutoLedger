package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpMaxResultBytes is the largest result a tool returns. The lists have their own caps and say when they are cut; this
// bounds the rest (a long period, a vehicle with many places) so one call cannot flood the model's context.
const mcpMaxResultBytes = 100 * 1024

// capToolResults replaces a result over mcpMaxResultBytes by an error that tells the model how to ask for less. The
// size is the larger of the structured output and the text content, which carries the same data a second time.
func capToolResults(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		result, ok := res.(*mcp.CallToolResult)
		if method != "tools/call" || err != nil || !ok || result == nil || result.IsError {
			return res, err
		}
		size := mcpResultSize(result)
		if size <= mcpMaxResultBytes {
			return res, err
		}
		tool := ""
		if params, isParams := req.GetParams().(*mcp.CallToolParamsRaw); isParams {
			tool = params.Name
		}
		slog.WarnContext(ctx, "MCP result too large", "component", "mcp", "tool", tool, "bytes", size, "max_bytes", mcpMaxResultBytes)
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(
				"The result is too large (%d KB, the limit is %d KB). Ask for less: a shorter period, a filter, or a lower limit.", size/1024, mcpMaxResultBytes/1024)}},
		}, nil
	}
}

func mcpResultSize(result *mcp.CallToolResult) int {
	text := 0
	for _, c := range result.Content {
		if t, ok := c.(*mcp.TextContent); ok {
			text += len(t.Text)
		}
	}
	structured := 0
	if result.StructuredContent != nil {
		if raw, err := json.Marshal(result.StructuredContent); err == nil {
			structured = len(raw)
		}
	}
	return max(text, structured)
}

// guardToolCalls is the middleware of the tool calls: the log line covers the outcome of the size cap too.
func (h *MCPHandler) guardToolCalls(next mcp.MethodHandler) mcp.MethodHandler {
	return h.logToolCalls(capToolResults(next))
}
