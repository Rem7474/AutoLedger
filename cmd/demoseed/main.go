// Command demoseed fills a running AutoLedger instance with a deterministic demo dataset through its public
// HTTP API: a user, an electric vehicle and a combustion vehicle with about eight months of history.
package main

import (
	"flag"
	"log"
	"os"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/demodata"
	"github.com/teslacost/teslacost/internal/demoseed"
)

func main() {
	url := flag.String("url", "http://localhost:8080", "base URL of the AutoLedger instance")
	email := flag.String("email", "demo@autoledger.example", "demo account email (created when absent)")
	password := flag.String("password", os.Getenv("DEMO_PASSWORD"), "demo account password (default: $DEMO_PASSWORD)")
	sets := flag.String("datasets", "ev,ice", "comma-separated datasets: ev, ice, layout (a hybrid with long names and another currency, for layout checks)")
	flag.Parse()
	if *password == "" {
		log.Fatal("a password is required (-password or $DEMO_PASSWORD)")
	}

	c := demoseed.New(*url)
	if err := c.Login(*email, *password); err != nil {
		log.Fatalf("login: %v", err)
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := c.Do("POST", "/api/auth/tokens/", map[string]string{"name": "demo seed", "scope": "ingest"}, &tok, ""); err != nil {
		log.Fatalf("api token: %v", err)
	}
	for _, key := range strings.Split(*sets, ",") {
		ds, err := demodata.Build(strings.TrimSpace(key), time.Now())
		if err != nil {
			log.Fatal(err)
		}
		if err := c.Seed(ds, tok.Token); err != nil {
			log.Fatalf("%s: %v", key, err)
		}
	}
}
