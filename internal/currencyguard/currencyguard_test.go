// Package currencyguard holds the check that no Go source assumes a currency.
package currencyguard

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// allowed lists the sources that may name the euro, with the reason. Everywhere else an amount carries the
// currency of its vehicle (or of the record that was entered), never an assumed one.
var allowed = map[string]string{
	"internal/services/toll_apply.go":          "the toll dataset (French autoroutes) is priced in euros; tolls are only applied to euro vehicles",
	"internal/services/comparison_service.go":  "the built-in indicative prices are written for the euro only",
	"internal/demodata/demodata.go":            "the demonstration vehicles are euro vehicles",
	"internal/database/repository_vehicles.go": "a vehicle stored without a currency (test fixtures; the API requires one) keeps the column default",
}

func TestNoAssumedCurrencyInSources(t *testing.T) {
	root := filepath.Join("..", "..")
	var found []string
	for _, dir := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if _, ok := allowed[rel]; ok {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			for i, line := range strings.Split(string(data), "\n") {
				code := line
				if at := strings.Index(code, "//"); at >= 0 {
					code = code[:at]
				}
				if strings.Contains(code, `"EUR"`) {
					found = append(found, rel+":"+strconv.Itoa(i+1))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(found) > 0 {
		t.Errorf("a source names the euro as a default currency; take the currency of the vehicle or of the request:\n%s", strings.Join(found, "\n"))
	}
}
