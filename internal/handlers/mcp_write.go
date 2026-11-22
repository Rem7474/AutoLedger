package handlers

import (
	"context"
	"errors"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
)

// mcpCanWrite reports whether the grant the call was made with may write.
func mcpCanWrite(req *mcp.CallToolRequest) bool {
	extra := req.GetExtra()
	return extra != nil && extra.TokenInfo != nil && slices.Contains(extra.TokenInfo.Scopes, auth.ScopeWrite)
}

// errMCPReadOnlyGrant is what a call made with a read-only grant gets from a writing tool.
var errMCPReadOnlyGrant = errors.New("this connection can only read: the owner has to allow writing when connecting the assistant to AutoLedger")

// addMCPWriteTool registers a tool that changes a vehicle's records. What every such tool does first lives here, so
// that a new one cannot leave it out: the grant must hold the write scope, and the vehicle the call names must be one
// of the account's (a vehicle the grant is limited away from is not). The role on that vehicle is checked by run, in
// the service the app's own routes share, as is each tool's own validation and its dry run.
//
// vehicleRef reads the vehicle out of the input and failure is the message the model sees when run fails without a
// reason of its own. Every tool whose annotations say it is not read-only has to be registered here:
// TestMCPWritingToolsAreRegisteredThroughTheBase fails for one that is not.
func addMCPWriteTool[In, Out any](h *MCPHandler, tool *mcp.Tool, failure string, vehicleRef func(In) string,
	run func(ctx context.Context, userID, vehicleID string, in In) (*Out, error)) {
	h.writeTools = append(h.writeTools, tool.Name)
	mcp.AddTool(h.server, tool, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var none Out
		if !mcpCanWrite(req) {
			return nil, none, errMCPReadOnlyGrant
		}
		vehicleID, err := h.resolveVehicle(ctx, mcpUserID(req), vehicleRef(in))
		if err != nil {
			return nil, none, mcpFailure(ctx, tool.Name, err, failure)
		}
		out, err := run(ctx, mcpUserID(req), vehicleID, in)
		if err != nil {
			return nil, none, mcpFailure(ctx, tool.Name, err, failure)
		}
		return nil, *out, nil
	})
}
