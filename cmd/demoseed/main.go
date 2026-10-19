// Command demoseed fills a running AutoLedger instance with a deterministic demo dataset through its public
// HTTP API: a user, an electric vehicle and a combustion vehicle with about eight months of history.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"github.com/teslacost/teslacost/internal/demodata"
)

type client struct {
	base  string
	http  *http.Client
	token string
}

func (c *client) do(method, path string, body, out any, bearer string) error {
	var rd io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rd = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, c.base+path, rd)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer == "" {
		bearer = c.token
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if res.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %d %s", method, path, res.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		return json.Unmarshal(data, out)
	}
	return nil
}

func (c *client) login(email, password string) error {
	creds := map[string]string{"email": email, "password": password}
	// The account may already exist (a re-run): registration then fails and the login below decides.
	_ = c.do("POST", "/api/auth/register", creds, nil, "")
	var res struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := c.do("POST", "/api/auth/login", creds, &res, ""); err != nil {
		return err
	}
	c.token = res.AccessToken
	if c.token == "" {
		c.token = res.Token
	}
	if c.token == "" {
		return fmt.Errorf("login returned no token")
	}
	return nil
}

// The API reads and writes amounts in currency units, not cents.
func units(cents int64) float64 { return float64(cents) / 100 }

func day(t time.Time) string { return t.Format("2006-01-02") }

func (c *client) seed(ds demodata.Dataset, apiToken string) error {
	var v struct {
		ID string `json:"id"`
	}
	err := c.do("POST", "/api/vehicles", map[string]any{
		"name": ds.Vehicle.Name, "make": ds.Vehicle.Make, "model": ds.Vehicle.Model,
		"powertrain": ds.Vehicle.Powertrain, "current_odometer": ds.Vehicle.StartOdometer,
		"currency": ds.Vehicle.Currency, "estimated_kwh_100km": ds.Vehicle.KWhPer100Km,
		"estimated_price_per_kwh": ds.Vehicle.PricePerKWh,
	}, &v, "")
	if err != nil {
		return err
	}
	if o := ds.Ownership; o.PriceCents > 0 {
		if err := c.do("PUT", "/api/vehicles/"+v.ID+"/ownership", ownershipPayload(o), nil, ""); err != nil {
			return err
		}
	}
	if ds.MonthlyBudgetCents > 0 {
		if err := c.do("PUT", "/api/fleet/budget", map[string]any{"amount": units(ds.MonthlyBudgetCents)}, nil, ""); err != nil {
			return err
		}
	}
	for _, e := range ds.Events {
		err := c.do("POST", "/api/integrations/homeassistant/event", map[string]any{
			"event_type": e.Type, "vehicle_id": v.ID, "event_id": e.ID,
			"timestamp": e.At.Format(time.RFC3339), "data": e.Data,
		}, nil, apiToken)
		if err != nil {
			return err
		}
	}
	if err := c.seedPlaces(v.ID, ds.Places); err != nil {
		return err
	}
	if err := c.seedDrivers(v.ID, ds.Drivers); err != nil {
		return err
	}
	if cp := ds.Carpool; cp != nil {
		if err := c.do("POST", "/api/vehicles/"+v.ID+"/carpools", carpoolPayload(cp), nil, ""); err != nil {
			return err
		}
	}
	for _, t := range ds.Tires {
		err := c.do("POST", "/api/vehicles/"+v.ID+"/tires", map[string]any{
			"brand": t.Brand, "model": t.Model, "dimension": t.Dimension, "season": t.Season,
			"purchase_date": day(t.PurchaseDate), "purchase_price": units(t.PriceCents), "current_position": t.Position,
			"initial_depth_mm": 8.0, "min_legal_depth_mm": 1.6, "mounted_odometer": t.MountedOdometer,
			"estimated_lifespan_km": t.LifespanKm,
		}, nil, "")
		if err != nil {
			return err
		}
	}
	for _, x := range ds.Expenses {
		var err error
		if x.Kind == "toll" {
			err = c.do("POST", "/api/vehicles/"+v.ID+"/expenses", map[string]any{
				"type": "TOLL", "amount": units(x.AmountCents), "currency": ds.Vehicle.Currency,
				"date": day(x.Date), "notes": x.Description,
			}, nil, "")
		} else {
			err = c.do("POST", "/api/vehicles/"+v.ID+"/maintenance", map[string]any{
				"category": x.Category, "amount": units(x.AmountCents), "currency": ds.Vehicle.Currency,
				"date": day(x.Date), "description": x.Description, "amortization_mode": "NONE",
			}, nil, "")
		}
		if err != nil {
			return err
		}
	}
	for _, r := range ds.Reminders {
		err := c.do("POST", "/api/vehicles/"+v.ID+"/reminders", map[string]any{
			"title": r.Title, "due_date": day(r.Due),
		}, nil, "")
		if err != nil {
			return err
		}
	}
	if cmp := ds.Comparison; cmp != nil {
		err := c.do("POST", "/api/comparison-scenarios/", map[string]any{
			"vehicle_id": v.ID, "name": cmp.Name, "mode": "RETROSPECTIVE", "annual_km": cmp.AnnualKm, "years": cmp.Years,
			"ice": map[string]any{
				"fuel_type": cmp.FuelType, "l_per_100km": cmp.LPer100Km, "fuel_price": cmp.FuelPrice,
				"purchase_price": units(cmp.PurchaseCents), "resale_value": units(cmp.ResaleCents),
				"maintenance_yearly": units(cmp.MaintenanceYearlyCents), "insurance_yearly": units(cmp.InsuranceYearlyCents),
				"tax_yearly": units(cmp.TaxYearlyCents),
			},
		}, nil, "")
		if err != nil {
			return err
		}
	}
	log.Printf("%s: %d events, %d tires, %d expenses, %d reminders", ds.Vehicle.Name, len(ds.Events), len(ds.Tires), len(ds.Expenses), len(ds.Reminders))
	return nil
}

// seedPlaces creates the charging places and links the location texts the charges report to them.
func (c *client) seedPlaces(vehicleID string, places []demodata.ChargingPlace) error {
	for _, p := range places {
		var created struct {
			ID string `json:"id"`
		}
		if err := c.do("POST", "/api/vehicles/"+vehicleID+"/charging-places", map[string]any{"name": p.Name, "category": p.Category}, &created, ""); err != nil {
			return err
		}
		for _, text := range p.Aliases {
			if err := c.do("PUT", "/api/vehicles/"+vehicleID+"/charging-places/aliases", map[string]any{"text": text, "place_id": created.ID}, nil, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// seedDrivers adds each driver to the vehicle and gives them every EveryNth drive, oldest first.
func (c *client) seedDrivers(vehicleID string, drivers []demodata.Driver) error {
	if len(drivers) == 0 {
		return nil
	}
	var ids []string
	for page := 1; ; page++ {
		var res struct {
			Drives []struct {
				ID string `json:"id"`
			} `json:"drives"`
			Total int `json:"total"`
		}
		path := fmt.Sprintf("/api/vehicles/%s/drives?limit=200&page=%d", vehicleID, page)
		if err := c.do("GET", path, nil, &res, ""); err != nil {
			return err
		}
		for _, d := range res.Drives {
			ids = append(ids, d.ID)
		}
		if len(res.Drives) == 0 || len(ids) >= res.Total {
			break
		}
	}
	// The list is newest first: reverse it so the share does not depend on when the seed runs
	for i, j := 0, len(ids)-1; i < j; i, j = i+1, j-1 {
		ids[i], ids[j] = ids[j], ids[i]
	}
	for _, drv := range drivers {
		var person struct {
			ID string `json:"id"`
		}
		if err := c.do("POST", "/api/vehicles/"+vehicleID+"/people", map[string]any{"name": drv.Name}, &person, ""); err != nil {
			return err
		}
		var mine []string
		for i, id := range ids {
			if i%drv.EveryNth == 0 {
				mine = append(mine, id)
			}
		}
		for start := 0; start < len(mine); start += 500 {
			end := min(start+500, len(mine))
			if err := c.do("PUT", "/api/vehicles/"+vehicleID+"/drives/driver", map[string]any{"drive_ids": mine[start:end], "driver_id": person.ID}, nil, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

func main() {
	url := flag.String("url", "http://localhost:8080", "base URL of the AutoLedger instance")
	email := flag.String("email", "demo@autoledger.example", "demo account email (created when absent)")
	password := flag.String("password", os.Getenv("DEMO_PASSWORD"), "demo account password (default: $DEMO_PASSWORD)")
	sets := flag.String("datasets", "ev,ice", "comma-separated datasets: ev, ice, layout (a hybrid with long names and another currency, for layout checks)")
	flag.Parse()
	if *password == "" {
		log.Fatal("a password is required (-password or $DEMO_PASSWORD)")
	}

	jar, _ := cookiejar.New(nil)
	c := &client{base: strings.TrimRight(*url, "/"), http: &http.Client{Jar: jar, Timeout: 30 * time.Second}}
	if err := c.login(*email, *password); err != nil {
		log.Fatalf("login: %v", err)
	}
	var tok struct {
		Token string `json:"token"`
	}
	if err := c.do("POST", "/api/auth/tokens/", map[string]string{"name": "demo seed", "scope": "ingest"}, &tok, ""); err != nil {
		log.Fatalf("api token: %v", err)
	}
	for _, key := range strings.Split(*sets, ",") {
		ds, err := demodata.Build(strings.TrimSpace(key), time.Now())
		if err != nil {
			log.Fatal(err)
		}
		if err := c.seed(ds, tok.Token); err != nil {
			log.Fatalf("%s: %v", key, err)
		}
	}
}

// ownershipPayload is the ownership of a demo vehicle: a LOA when it has a lease, a cash purchase otherwise.
func ownershipPayload(o demodata.Ownership) map[string]any {
	if l := o.Lease; l != nil {
		return map[string]any{
			"acquisition_type": "LOA", "start_date": day(o.Date), "start_odometer": o.Odometer,
			"purchase_price": units(o.PriceCents), "expected_resale_value": units(l.EndValueCents),
			"lease_monthly_rent": units(l.MonthlyRentCents), "lease_duration_months": l.DurationMonths,
			"lease_down_payment": units(l.DownPaymentCents), "lease_km_allowance_per_year": l.KmAllowancePerYear,
			"lease_excess_km_price": l.ExcessKmPrice, "lease_purchase_option_price": units(l.OptionPriceCents),
		}
	}
	payload := map[string]any{
		"acquisition_type": "CASH", "start_date": day(o.Date), "start_odometer": o.Odometer,
		"purchase_price": units(o.PriceCents), "purchase_fees": units(o.FeesCents),
	}
	if o.ResaleCents > 0 && o.HoldingMonths > 0 {
		payload["expected_resale_value"], payload["expected_holding_months"] = units(o.ResaleCents), o.HoldingMonths
	}
	return payload
}

// carpoolPayload is the API body of a demo carpooled trip, dated DaysAgo days back.
func carpoolPayload(cp *demodata.Carpool) map[string]any {
	legs := make([]map[string]any, 0, len(cp.Legs))
	total := 0.0
	for _, l := range cp.Legs {
		legs = append(legs, map[string]any{
			"start_label": l.From, "end_label": l.To, "distance_km": l.DistanceKm,
			"electricity_cost": units(l.ElectricityCents), "tolls_cost": units(l.TollsCents),
		})
		total += l.DistanceKm
	}
	passengers := make([]map[string]any, 0, len(cp.Passengers))
	for _, p := range cp.Passengers {
		passengers = append(passengers, map[string]any{
			"passenger_name": p.Name, "seats": p.Seats, "amount_paid": units(p.PaidCents),
			"board_stop_index": p.Board, "alight_stop_index": p.Alight + 1,
		})
	}
	return map[string]any{
		"title": cp.Title, "date": day(time.Now().AddDate(0, 0, -cp.DaysAgo)), "distance_km": total,
		"legs": legs, "passengers": passengers,
	}
}
