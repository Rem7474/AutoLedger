package handlers

import (
	"context"
	"encoding/json"
	"flag"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// listMCPTools returns the tools the server advertises, through the protocol, without an account or a database.
func listMCPTools(t *testing.T) []*mcp.Tool {
	t.Helper()
	h := NewMCPHandler(nil, nil, "https://ledger.example", "test")
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx := context.Background()
	serverSession, err := h.server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1"}, nil).Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = session.Close() })
	res, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	return res.Tools
}

// undescribed lists the paths of the properties of a JSON schema that carry no description.
func undescribed(schema map[string]any, path string) []string {
	var missing []string
	props, _ := schema["properties"].(map[string]any)
	for name, raw := range props {
		prop, _ := raw.(map[string]any)
		here := path + "." + name
		if strings.TrimSpace(asString(prop["description"])) == "" {
			missing = append(missing, here)
		}
		missing = append(missing, undescribed(prop, here)...)
		if items, ok := prop["items"].(map[string]any); ok {
			missing = append(missing, undescribed(items, here+"[]")...)
		}
	}
	return missing
}

func asString(v any) string { s, _ := v.(string); return s }

func schemaOf(t *testing.T, v any) map[string]any {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// A model reads a tool's schema, not the documentation: a field without a description is a figure it has to guess
// the meaning of.
func TestEveryFieldOfEveryMCPToolIsDescribed(t *testing.T) {
	tools := listMCPTools(t)
	if len(tools) == 0 {
		t.Fatal("no tool is advertised")
	}
	for _, tool := range tools {
		if strings.TrimSpace(tool.Title) == "" {
			t.Errorf("%s has no title", tool.Name)
		}
		for kind, schema := range map[string]any{"input": tool.InputSchema, "output": tool.OutputSchema} {
			if schema == nil {
				continue
			}
			missing := undescribed(schemaOf(t, schema), tool.Name+"."+kind)
			sort.Strings(missing)
			if len(missing) > 0 {
				t.Errorf("fields without a description:\n  %s", strings.Join(missing, "\n  "))
			}
		}
	}
}

var updateMCPGolden = flag.Bool("update-mcp-golden", false, "rewrite testdata/mcp_tools.golden.json")

// The tools and their schemas are a contract with every client: a rename, a removed field or a changed type breaks
// an assistant's saved prompts. This test makes such a change a visible diff in review instead of a side effect.
// Regenerate with: go test ./internal/handlers -run TestMCPToolListMatchesTheGoldenFile -update-mcp-golden
func TestMCPToolListMatchesTheGoldenFile(t *testing.T) {
	tools := listMCPTools(t)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Name < tools[j].Name })
	got, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')

	const path = "testdata/mcp_tools.golden.json"
	if *updateMCPGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (generate it with -update-mcp-golden)", err)
	}
	if string(want) != string(got) {
		t.Errorf("the advertised tools changed; review the diff and regenerate with -update-mcp-golden\n--- want\n%s\n--- got\n%s", want, got)
	}
}
