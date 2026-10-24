package handlers

import (
	"context"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// mcpSurface lists what the server advertises through the protocol: tool and prompt names, resource URIs and templates.
func mcpSurface(t *testing.T) (tools, prompts, resources []string) {
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

	toolList, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range toolList.Tools {
		tools = append(tools, tool.Name)
	}
	promptList, err := session.ListPrompts(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range promptList.Prompts {
		prompts = append(prompts, p.Name)
	}
	resourceList, err := session.ListResources(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range resourceList.Resources {
		resources = append(resources, r.URI)
	}
	templateList, err := session.ListResourceTemplates(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range templateList.ResourceTemplates {
		resources = append(resources, r.URITemplate)
	}
	sort.Strings(tools)
	sort.Strings(prompts)
	sort.Strings(resources)
	return tools, prompts, resources
}

var docTableName = regexp.MustCompile("(?m)^\\| `([^`]+)` \\|")

// documented returns the first column of the tables of a Markdown file, between two level-2 headings.
func documented(t *testing.T, path, heading string) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	start := strings.Index(text, "\n## "+heading+"\n")
	if start < 0 {
		t.Fatalf("%s has no section %q", path, heading)
	}
	section := text[start+1:]
	if end := strings.Index(section[3:], "\n## "); end >= 0 {
		section = section[:end+3]
	}
	names := map[string]bool{}
	for _, m := range docTableName.FindAllStringSubmatch(section, -1) {
		names[m[1]] = true
	}
	return names
}

// The tool reference is what a person reads before connecting a client: a tool, a prompt or a resource the server
// advertises and the page does not list is invisible to them, and a listed one that no longer exists misleads.
func TestMCPDocumentationListsExactlyWhatTheServerAdvertises(t *testing.T) {
	tools, prompts, resources := mcpSurface(t)
	if len(tools) == 0 || len(prompts) == 0 || len(resources) == 0 {
		t.Fatalf("the server advertises nothing: %d tools, %d prompts, %d resources", len(tools), len(prompts), len(resources))
	}
	pages := []struct{ path, tools, other string }{
		{"../../docs/mcp.md", "Tools", "Resources and prompts"},
		{"../../docs/mcp.fr.md", "Outils", "Ressources et prompts"},
	}
	for _, page := range pages {
		inTools := documented(t, page.path, page.tools)
		inOther := documented(t, page.path, page.other)
		for _, name := range tools {
			if !inTools[name] {
				t.Errorf("%s does not list the tool %s", page.path, name)
			}
		}
		for _, name := range append(append([]string{}, prompts...), resources...) {
			if !inOther[name] {
				t.Errorf("%s does not list %s", page.path, name)
			}
		}
		advertised := map[string]bool{}
		for _, name := range append(append(append([]string{}, tools...), prompts...), resources...) {
			advertised[name] = true
		}
		for name := range inTools {
			if !advertised[name] {
				t.Errorf("%s lists the tool %s, which the server does not advertise", page.path, name)
			}
		}
		for name := range inOther {
			if !advertised[name] {
				t.Errorf("%s lists %s, which the server does not advertise", page.path, name)
			}
		}
	}
}
