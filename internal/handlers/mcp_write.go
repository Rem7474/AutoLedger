package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/teslacost/teslacost/internal/auth"
	"github.com/teslacost/teslacost/internal/database"
)

// mcpCanWrite reports whether the grant the call was made with may write.
func mcpCanWrite(req *mcp.CallToolRequest) bool {
	extra := req.GetExtra()
	return extra != nil && extra.TokenInfo != nil && slices.Contains(extra.TokenInfo.Scopes, auth.ScopeWrite)
}

// errMCPReadOnlyGrant is what a call made with a read-only grant gets from a writing tool.
var errMCPReadOnlyGrant = errors.New("this connection can only read: the owner has to allow writing when connecting the assistant to AutoLedger")

const (
	mcpMaxIdempotencyKey = 100
	// mcpIdempotentMethod is what an MCP call stores in the method column of idempotency_keys, which the HTTP routes
	// fill with their own method: a key reused across the two is refused instead of replayed.
	mcpIdempotentMethod = "MCP"
)

// mcpWriteConfig is what addMCPWriteTool is told beyond the tool itself.
type mcpWriteConfig[In any] struct {
	idempotencyKey func(In) string
	dryRun         func(In) bool
}

// mcpWriteOption adjusts addMCPWriteTool.
type mcpWriteOption[In any] func(*mcpWriteConfig[In])

// withIdempotencyKey makes a tool that creates a record replayable: a call that carries a key is answered, when the key
// was already used by the same call, with the answer it got the first time and records nothing. A dry run stores no key.
func withIdempotencyKey[In any](key func(In) string, dryRun func(In) bool) mcpWriteOption[In] {
	return func(c *mcpWriteConfig[In]) { c.idempotencyKey, c.dryRun = key, dryRun }
}

// mcpCallFingerprint identifies a call by its tool and its arguments, the key and the dry run left out, so that the
// same key with other arguments is told from a retry.
func mcpCallFingerprint(tool string, in any) (string, error) {
	raw, err := json.Marshal(in)
	if err != nil {
		return "", err
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", err
	}
	delete(args, "idempotency_key")
	delete(args, "dry_run")
	canonical, err := json.Marshal(args) // map keys are sorted
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return "mcp:" + tool + ":" + hex.EncodeToString(sum[:]), nil
}

// addMCPWriteTool registers a tool that changes a vehicle's records. What every such tool does first lives here, so
// that a new one cannot leave it out: the grant must hold the write scope, and the vehicle the call names must be one
// of the account's (a vehicle the grant is limited away from is not). The role on that vehicle is checked by run, in
// the service the app's own routes share, as is each tool's own validation and its dry run.
//
// vehicleRef reads the vehicle out of the input and failure is the message the model sees when run fails without a
// reason of its own. Every tool whose annotations say it is not read-only has to be registered here:
// TestMCPWritingToolsAreRegisteredThroughTheBase fails for one that is not.
func addMCPWriteTool[In, Out any](h *MCPHandler, tool *mcp.Tool, failure string, vehicleRef func(In) string,
	run func(ctx context.Context, userID, vehicleID string, in In) (*Out, error), options ...mcpWriteOption[In]) {
	var cfg mcpWriteConfig[In]
	for _, option := range options {
		option(&cfg)
	}
	h.writeTools = append(h.writeTools, tool.Name)
	mcp.AddTool(h.server, tool, func(ctx context.Context, req *mcp.CallToolRequest, in In) (*mcp.CallToolResult, Out, error) {
		var none Out
		if !mcpCanWrite(req) {
			return nil, none, errMCPReadOnlyGrant
		}
		userID := mcpUserID(req)
		vehicleID, err := h.resolveVehicle(ctx, userID, vehicleRef(in))
		if err != nil {
			return nil, none, mcpFailure(ctx, tool.Name, err, failure)
		}
		key := ""
		if cfg.idempotencyKey != nil && !cfg.dryRun(in) {
			key = strings.TrimSpace(cfg.idempotencyKey(in))
		}
		if len(key) > mcpMaxIdempotencyKey {
			return nil, none, fmt.Errorf("idempotency_key is %d characters, %d at most", len(key), mcpMaxIdempotencyKey)
		}
		var fingerprint string
		if key != "" {
			var err error
			if fingerprint, err = mcpCallFingerprint(tool.Name, in); err != nil {
				return nil, none, mcpFailure(ctx, tool.Name, err, failure)
			}
			// A retry that arrives while the first call is still running waits for it, then reads its answer
			release, err := h.idempotencyLocks.acquire(ctx, idempotencyRequestKey{userID: userID, key: key})
			if err != nil {
				return nil, none, mcpFailure(ctx, tool.Name, err, failure)
			}
			defer release()
			stored, err := h.read.repo.GetIdempotentResponse(ctx, userID, key)
			if err != nil {
				return nil, none, mcpFailure(ctx, tool.Name, err, failure)
			}
			if stored != nil {
				if stored.Method != mcpIdempotentMethod || stored.Path != fingerprint {
					return nil, none, errors.New("this idempotency_key was already used for another call: use a new key for a different record")
				}
				var replay Out
				if err := json.Unmarshal(stored.Body, &replay); err != nil {
					return nil, none, mcpFailure(ctx, tool.Name, err, failure)
				}
				return nil, replay, nil
			}
		}
		out, err := run(ctx, userID, vehicleID, in)
		if err != nil {
			return nil, none, mcpFailure(ctx, tool.Name, err, failure)
		}
		if key != "" {
			if body, err := json.Marshal(out); err == nil {
				if err := h.read.repo.SaveIdempotentResponse(ctx, userID, key, database.StoredResponse{Method: mcpIdempotentMethod, Path: fingerprint, StatusCode: 200, Body: body}); err != nil {
					slog.WarnContext(ctx, "MCP idempotency store failed", "component", "mcp", "tool", tool.Name, "error", err)
				}
			}
		}
		return nil, *out, nil
	})
}
