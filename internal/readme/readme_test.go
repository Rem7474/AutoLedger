package readme

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
)

var (
	headingRe = regexp.MustCompile(`(?m)^(#{1,6}) (\S+)`)
	linkRe    = regexp.MustCompile(`\]\((?:\./)?([^)#\s:]+)(?:#[^)]*)?\)`)
	srcRe     = regexp.MustCompile(`src="([^"]+)"`)
)

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// outline lists each heading as its level plus its first word, which is the emoji for the sections that have one.
func outline(doc string) []string {
	var out []string
	inFence := false
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
		}
		if inFence {
			continue
		}
		if m := headingRe.FindStringSubmatch(line); m != nil {
			first := m[2]
			if len(m[1]) == 1 || !regexp.MustCompile(`^[A-Za-z]`).MatchString(first) {
				out = append(out, m[1]+" "+first)
			} else {
				out = append(out, m[1])
			}
		}
	}
	return out
}

func TestReadmeHeadingsMatch(t *testing.T) {
	en, fr := outline(read(t, "README.md")), outline(read(t, "README.fr.md"))
	if len(en) != len(fr) {
		t.Fatalf("README.md has %d headings, README.fr.md has %d", len(en), len(fr))
	}
	for i := range en {
		if en[i] != fr[i] {
			t.Errorf("heading %d: README.md %q, README.fr.md %q", i+1, en[i], fr[i])
		}
	}
}

func TestReadmesLinkToEachOther(t *testing.T) {
	if !strings.Contains(read(t, "README.md"), "(README.fr.md)") {
		t.Error("README.md does not link to README.fr.md")
	}
	if !strings.Contains(read(t, "README.fr.md"), "(README.md)") {
		t.Error("README.fr.md does not link to README.md")
	}
}

func TestReadmeLocalTargetsExist(t *testing.T) {
	for _, name := range []string{"README.md", "README.fr.md"} {
		doc := read(t, name)
		var targets []string
		for _, m := range linkRe.FindAllStringSubmatch(doc, -1) {
			targets = append(targets, m[1])
		}
		for _, m := range srcRe.FindAllStringSubmatch(doc, -1) {
			targets = append(targets, m[1])
		}
		for _, target := range targets {
			if strings.HasPrefix(target, "http") {
				continue
			}
			if _, err := os.Stat(filepath.Join("..", "..", target)); err != nil {
				t.Errorf("%s: %s does not exist", name, target)
			}
		}
	}
}

var anchorRe = regexp.MustCompile(`href="#([^"]+)"|\]\(#([^)]+)\)`)

// slug follows GitHub's rule: lower case, letters, digits, hyphens and underscores kept, spaces become hyphens.
func slug(heading string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(heading) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

func TestReadmeAnchorsResolve(t *testing.T) {
	for _, name := range []string{"README.md", "README.fr.md"} {
		doc := read(t, name)
		slugs := map[string]bool{}
		for _, m := range regexp.MustCompile(`(?m)^#{1,6} (.+)$`).FindAllStringSubmatch(doc, -1) {
			slugs[slug(m[1])] = true
		}
		for _, m := range anchorRe.FindAllStringSubmatch(doc, -1) {
			a := m[1] + m[2]
			if !slugs[a] {
				t.Errorf("%s: #%s matches no heading", name, a)
			}
		}
	}
}

// TestComposeForwardsTheDocumentedSettings fails when a setting the README documents never reaches the container: the
// Compose file lists the environment explicitly, so a variable set in .env is ignored without a line for it, and the
// feature it switches silently stays off.
func TestComposeForwardsTheDocumentedSettings(t *testing.T) {
	compose := read(t, "docker-compose.yml")
	// Not forwarded on purpose: the database host and port are fixed by the Compose network, and the OIDC block is
	// opt-in text to uncomment.
	notForwarded := map[string]bool{"AUTOLEDGER_DB_HOST": true, "AUTOLEDGER_DB_PORT": true, "OIDC_SCOPES": true}
	row := regexp.MustCompile("(?m)^\\| `([A-Z][A-Z0-9_]+)`(?: / `([A-Z][A-Z0-9_]+)`)? \\|")
	rows := row.FindAllStringSubmatch(read(t, "README.md"), -1)
	if len(rows) < 20 {
		t.Fatalf("found %d settings in the README table, the pattern no longer matches it", len(rows))
	}
	for _, m := range rows {
		for _, name := range m[1:] {
			if name != "" && !notForwarded[name] && !strings.Contains(compose, name) {
				t.Errorf("%s is documented in the README but docker-compose.yml does not pass it to the container", name)
			}
		}
	}
}
