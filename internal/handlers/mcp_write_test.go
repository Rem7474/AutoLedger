package handlers

import (
	"slices"
	"testing"
)

// A tool that is not read-only changes the owner's records, so it has to go through addMCPWriteTool, which refuses a
// read-only grant before anything else. A new writing tool registered with mcp.AddTool directly would skip that.
func TestMCPWritingToolsAreRegisteredThroughTheBase(t *testing.T) {
	h := NewMCPHandler(nil, nil, "https://ledger.example", "test")
	var writing []string
	for _, tool := range listMCPTools(t) {
		readOnly := tool.Annotations != nil && tool.Annotations.ReadOnlyHint
		if !readOnly && !slices.Contains(h.writeTools, tool.Name) {
			t.Errorf("%s is not read-only but is not registered through addMCPWriteTool", tool.Name)
		}
		if !readOnly {
			writing = append(writing, tool.Name)
		}
	}
	for _, name := range h.writeTools {
		if !slices.Contains(writing, name) {
			t.Errorf("%s is registered through addMCPWriteTool but advertised as read-only", name)
		}
	}
	if len(writing) != len(h.writeTools) || len(writing) == 0 {
		t.Errorf("writing tools: advertised %v, registered %v", writing, h.writeTools)
	}
}
