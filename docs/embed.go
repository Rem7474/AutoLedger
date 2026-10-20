// Package docs embeds the reference documents the application serves itself.
package docs

import _ "embed"

// CostTerms is the English glossary of the cost and consumption terms (docs/cost-terms.md), the one source of the
// definitions the interface shows and the MCP glossary resource returns.
//
//go:embed cost-terms.md
var CostTerms string
