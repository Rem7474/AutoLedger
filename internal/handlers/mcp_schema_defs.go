package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// The output schemas repeat the same object (an amount, a distance...) hundreds of times, which makes tools/list the
// heaviest thing a client loads. mcpSharedDefs moves each object that a schema repeats into the $defs of that schema
// and points at it with a $ref; a description written on the use stays on the use. The schemas the server validates
// its answers against are not touched: only what tools/list sends.

// mcpMinSharedBytes is the size under which an object is not worth a $defs entry.
const mcpMinSharedBytes = 150

func mcpSharedDefs(next mcp.MethodHandler) mcp.MethodHandler {
	var mu sync.Mutex
	cache := map[*mcp.Tool]*mcp.Tool{}
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		res, err := next(ctx, method, req)
		list, ok := res.(*mcp.ListToolsResult)
		if method != "tools/list" || err != nil || !ok {
			return res, err
		}
		out := *list
		out.Tools = make([]*mcp.Tool, len(list.Tools))
		mu.Lock()
		defer mu.Unlock()
		for i, tool := range list.Tools {
			shared, ok := cache[tool]
			if !ok {
				shared = withSharedDefs(tool)
				cache[tool] = shared
			}
			out.Tools[i] = shared
		}
		return &out, nil
	}
}

// withSharedDefs returns the tool with its output schema factored, or the tool itself when there is nothing to factor.
func withSharedDefs(tool *mcp.Tool) *mcp.Tool {
	if tool.OutputSchema == nil {
		return tool
	}
	raw, err := json.Marshal(tool.OutputSchema)
	if err != nil {
		return tool
	}
	var schema map[string]any
	if json.Unmarshal(raw, &schema) != nil {
		return tool
	}
	factored := factorSchema(schema)
	if factored == nil {
		return tool
	}
	copied := *tool
	copied.OutputSchema = factored
	return &copied
}

func isObjectSchema(n map[string]any) bool {
	_, hasProps := n["properties"]
	return n["type"] == "object" && hasProps
}

// withoutDescription is the schema as it matters to a validator: two uses that differ only by their description share one entry.
func withoutDescription(n map[string]any) map[string]any {
	out := make(map[string]any, len(n))
	for k, v := range n {
		if k != "description" {
			out[k] = v
		}
	}
	return out
}

func schemaKey(n map[string]any) string {
	b, _ := json.Marshal(withoutDescription(n)) // map keys are sorted by encoding/json
	return string(b)
}

// factorSchema moves the objects a schema repeats into its $defs; nil when none is worth it.
func factorSchema(schema map[string]any) map[string]any {
	counts := map[string]int{}
	var count func(n any)
	count = func(n any) {
		switch v := n.(type) {
		case map[string]any:
			if isObjectSchema(v) {
				counts[schemaKey(v)]++
			}
			for _, c := range v {
				count(c)
			}
		case []any:
			for _, c := range v {
				count(c)
			}
		}
	}
	count(schema)
	repeated := map[string]bool{}
	for k, c := range counts {
		if c >= 2 && len(k) >= mcpMinSharedBytes {
			repeated[k] = true
		}
	}
	if len(repeated) == 0 {
		return nil
	}

	names := map[string]string{}
	defs := map[string]any{}
	var rewrite func(n any, top bool) any
	rewrite = func(n any, top bool) any {
		switch v := n.(type) {
		case map[string]any:
			if !top && isObjectSchema(v) {
				if key := schemaKey(v); repeated[key] {
					name, seen := names[key]
					if !seen {
						name = fmt.Sprintf("shape%d", len(names)+1)
						names[key] = name
						defs[name] = nil // reserved before the body is rewritten, which may reach other shapes
						defs[name] = rewrite(withoutDescription(v), true)
					}
					ref := map[string]any{"$ref": "#/$defs/" + name}
					if d, ok := v["description"]; ok {
						ref["description"] = d
					}
					return ref
				}
			}
			out := make(map[string]any, len(v))
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				out[k] = rewrite(v[k], false)
			}
			return out
		case []any:
			out := make([]any, len(v))
			for i, c := range v {
				out[i] = rewrite(c, false)
			}
			return out
		}
		return n
	}
	out := rewrite(schema, true).(map[string]any)
	out["$defs"] = defs
	return out
}
