# Ingestion API

Home Assistant, Node-RED, n8n or any script can push events to AutoLedger over HTTP. Français : [ingestion-api.fr.md](ingestion-api.fr.md).

## Authentication

Create an API token in the account page (`al_live_...`). It is account-level and only opens the `/api/integrations/**` routes. Send it as a bearer token:

```
Authorization: Bearer al_live_xxxxxxxx
```

### Scope

A token has a scope, chosen when it is created and never changed afterwards (generate another token instead):

| Scope | Can do |
|---|---|
| Send and read (`ingest`) | Push events (`POST /api/integrations/homeassistant/event`) and read the vehicles and their metrics. This is what Home Assistant needs. The read API is refused. |
| Read only (`read`) | Read the vehicles and their metrics, and the [read API](#read-api). A push is refused with `403`. |

Use a read-only token for anything that only looks at the figures (a dashboard, a script, an assistant): a leaked or misused token cannot write to the account.

## Endpoint

`POST /api/integrations/homeassistant/event` with a JSON body.

| Field | Type | Notes |
|---|---|---|
| `event_type` | string | `charging_session_end` (default when empty), `odometer`, `drive`, `fuel`. `odometer_update` and `telemetry_update` are older names of `odometer`. |
| `vehicle_id` | string | Required for `odometer`, `drive`, `fuel`. Optional for charging sessions. The account needs edit access to the vehicle. |
| `event_id` | string | Your own identifier for the event, used for idempotence (see below). |
| `timestamp` | RFC 3339 | Used when `data.start_time` / `data.end_time` is absent. |
| `distance_unit` | `km` (default) or `mi` | Unit of the distances and odometer values of the event. Values are stored in km. |
| `data` | object | Fields of the event type, below. |

### `charging_session_end`

`data`: `start_time`, `end_time`, `energy_kwh` (or `energy_added_kwh`; above 0, at most 500), `cost` (currency units) or `cost_cents`, `location`, `charger_name`, `odometer_km`, `soc_start`, `soc_end`.

Without a cost, the vehicle's tariff plan is applied when there is one, otherwise the cost is left to complete. Electric vehicles only.

A session without `vehicle_id` goes to the home charger's default vehicle, otherwise to the only electric vehicle the account can edit, otherwise it waits in the pending charges (a charger shared by several vehicles) until you assign it in the app.

### `odometer`

`data`: `odometer` (in `distance_unit`) or `odometer_km`. The vehicle's odometer only moves forward.

### `drive`

`data`: `start_time` (or the event `timestamp`), `end_time` or `duration_min`, `distance` (in `distance_unit`, 0.1 to 3000 km), `start_odometer`, `end_odometer`, `start_address`, `end_address`, `energy_kwh`.

`distance` is optional. When it is absent, it is the difference of `end_odometer` and `start_odometer` if both are sent. Otherwise the drive is stored without distance (`distance_km` is `null` in the API, a dash in the app, an empty cell in the CSV export), with no energy estimate and no consumption figure; a `distance` sent as 0 is refused. `start_lat`/`start_lon` and `end_lat`/`end_lon` give the positions: they stand in for `start_address` / `end_address` when these are absent (stored as `lat, lon` text). When `GEOCODING_ENABLED` is set on the server, the positions are then replaced in the background by a street address looked up on OpenStreetMap Nominatim (an address edited in the meantime is kept; on failure the coordinates stay).

Without `energy_kwh`, the energy is estimated from the vehicle's average consumption and flagged as estimated.

### `fuel`

Combustion vehicles only. `data`: `amount` (or `cost`), `liters`, `price_per_liter`, `fuel_type`, `is_full_tank`, `odometer` or `odometer_km`. The amount can be derived from litres and price per litre. The time is the event `timestamp` (or `data.start_time`), now when absent.

## Responses

| Status | `status` | Meaning |
|---|---|---|
| 201 | `recorded` | Stored. The body carries `vehicle_id` and the id of the new record. |
| 201 | `pending_qualification` | A charging session waiting for a vehicle (`pending_id`). |
| 200 | `duplicate` | Already recorded or already pending; nothing was written. |
| 200 | (odometer) | Reading applied. |

Errors answer `{"error": "...", "code": "...", "params": {...}}`. Frequent codes:

| Code | Cause |
|---|---|
| `request.invalid_body` | The body is not valid JSON. |
| `integration.unknown_event_type` | Misspelt `event_type`. |
| `integration.invalid_distance_unit` | `distance_unit` is neither `km` nor `mi`. |
| `vehicle.not_specified` | `vehicle_id` missing on an event that needs it. |
| `telemetry.missing_odometer` | No valid odometer reading. |
| `charge.invalid_energy`, `charge.invalid_period`, `charge.electric_only` | Charging session rejected. |
| `drive.missing_start_time`, `drive.invalid_distance` | Drive rejected. |
| `fuel.combustion_only`, `fuel.amount_required` | Fill-up rejected. |

## Idempotence

Sending the same event again is safe. A record is recognised as already stored:

1. by `event_id` (charging sessions and drives), when you send one;
2. otherwise by tolerance: a charge starting within 30 min with an energy within 0.5 kWh, a drive starting within 15 min with a distance within 1 km, a fill-up within 30 min with the same amount (fill-ups are only recognised this way).

Use a stable `event_id` (a session id, a timestamp-based key) whenever the source has one.

## Read API

A read-only token (`read`) also opens a narrow read API under `/api/integrations/v1`, meant for assistants and reports. A token with the `ingest` scope is refused on it: a token that can push events cannot also read the costs.

| Route | Returns |
|---|---|
| `GET /api/integrations/v1/vehicles` | The account's vehicles: `id`, `name`, `make`, `model`, `powertrain`, `currency`, `odometer`, `role`, `tracking_mode` and, when the acquisition is entered, `acquisition_type` and `owned_since`. |
| `GET /api/integrations/v1/vehicles/{vehicleId}` | One vehicle, same fields. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/costs` | What the vehicle cost over its last 12 months (`last_12_months`) and since it was bought (`since_purchase`), by category, with the cost per km and a `completeness` summary. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/data-gaps` | What the history is missing: odometer gaps and anomalies (the 20 most recent are listed), charges without a cost, unqualified drives, untracked distance, missing insurance or acquisition, the details of what the acquisition lacks, `suspicious_entries` (recent entries that look mistyped, at most one per kind), and `missing`: what to complete, ranked as the application ranks it, each with a rough worth (`estimated_amount`, `null` when it cannot be estimated) and the `score_dimensions` of the completeness score. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/trips` | The drives, most recent first, with their `from` and `to` addresses, distance, duration, energy, `travel_expenses` (tolls, parking) and tags. Query: `search` (text in an address), `from` and `to` (days `YYYY-MM-DD` in UTC, or RFC 3339 instants), `limit` (20 by default, 50 at most). `matching` counts the drives that fit and `truncated` says more than the listed ones exist. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/monthly-costs` | The cost of each calendar month, oldest first, by category, with the month's distance and cost per km, and the listed months added up (`total`). Query: `from` and `to` (months, `YYYY-MM`, both included); without them the last 12 months, at most 120 months otherwise. A month without a record is listed with zero amounts, and the month in progress has `in_progress: true`. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/reminders` | The maintenance reminders of the vehicle, most urgent first, each with its status (`overdue`, `due_soon`, `ok`), the distance and days left, its due odometer and date and its last service, plus the counts of overdue and due-soon reminders. Query: `status` (`overdue`, `due_soon` or `ok`) to keep only those; the counts ignore it. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/ownership` | The economic cost of owning the vehicle (`economic_cost`: full cost, depreciation and whether it is known, per-km rates, carpool revenue), the acquisition, the lease or loan (`contract`, with the mileage allowance and the projected excess cost) and the estimated resale value (`resale`, with the reason when it cannot be estimated). |
| `GET /api/integrations/v1/vehicles/{vehicleId}/energy-efficiency` | Consumption, energy prices and costs per 100 km, charging mix, effect of cold weather and battery health for the electric side (`electric`, null when the vehicle cannot be plugged in), and litres per 100 km and fuel prices for the fuel side (`fuel`, null when it burns no fuel). |
| `GET /api/integrations/v1/vehicles/{vehicleId}/expenses` | The entries of the vehicle's cost ledger (charges, fuel, tolls, maintenance, insurance, tires, loan and lease instalments), newest first, with the total and the per-category sums of all the matching entries. Query: `from`, `to` (days, `YYYY-MM-DD`), `category` (repeatable or comma separated), `min_amount`, `sort` (`date` or `amount`), `order` (`asc`), `limit` (50, at most 200) and `cursor` (the `next_cursor` of the previous page). The purchase is not an entry. |
| `GET /api/integrations/v1/fleet` | The vehicles of the account side by side (month cost, running and full cost per km, annual cost, consumption, cost rank among the comparable vehicles of a currency), the month's totals per currency, the monthly budget and its use, the last 12 months of household cost in the fleet currency, and the share of the distance each named driver drove. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/tags` | The drives of a vehicle split by tag over the whole history: distance, energy, tolls, allocated cost and cost per km of each tag and of the untagged drives. A drive with several tags counts in each. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/carpool` | The carpool totals of a vehicle and its most recent shared trips (`limit`, 10 by default, 30 at most) with their stops, costs and named passengers. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/trip-groups` | The journeys of a vehicle (`limit`, 20 by default, 50 at most): name, period, drives, distance, tolls and parking, carpool trips. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/tires` | The tires of a vehicle and their wear: position, distance, lifespan used, tread depth, condition, distance left, expected replacement day, cost per km. |
| `GET /api/integrations/v1/vehicles/{vehicleId}/charging-places` | The charging of a vehicle by place (`months`, 12 by default, `0` for the whole history): named places with their addresses, sessions, energy, cost and price per kWh, the comparison with the home price and the most used locations linked to no place. |

Conventions:

- An amount is `{"amount": "12.34", "currency": "GBP"}`: a decimal string, in the vehicle's currency. Amounts are never converted between currencies.
- A distance is `{"value": 4200, "unit": "km"}`. The unit is always kilometres.
- A figure that cannot be known (a cost per km with no distance, a score with nothing to score) is `null`, never `0`.
- `months_covered` says how many calendar months a period spans: `last_12_months` of a vehicle with a shorter history spans fewer than 12.
- The costs are cash spent, the purchase of the vehicle excluded. The month in progress counts with the instalments that fall due later in it.
- The responses carry no VIN, plate, coordinate, document or drive identifier. Addresses and names the account recorded are returned (trip addresses, drivers, carpool stops and passengers, charging places); an address stored only as coordinates is `null`.
- A vehicle the account cannot see answers `404`.

## Examples

curl:

```bash
curl -X POST https://autoledger.example.com/api/integrations/homeassistant/event \
  -H "Authorization: Bearer al_live_xxxxxxxx" -H "Content-Type: application/json" \
  -d '{"event_type":"charging_session_end","event_id":"wallbox-2026-05-01-1","vehicle_id":"<id>",
       "data":{"start_time":"2026-05-01T22:00:00Z","end_time":"2026-05-02T05:30:00Z","energy_kwh":38.4,"cost":5.76}}'
```

A drive in miles:

```bash
curl -X POST https://autoledger.example.com/api/integrations/homeassistant/event \
  -H "Authorization: Bearer al_live_xxxxxxxx" -H "Content-Type: application/json" \
  -d '{"event_type":"drive","vehicle_id":"<id>","distance_unit":"mi",
       "data":{"start_time":"2026-05-01T08:00:00Z","duration_min":25,"distance":12.4,"end_odometer":42310}}'
```

Home Assistant (`rest_command`):

```yaml
rest_command:
  autoledger_odometer:
    url: "https://autoledger.example.com/api/integrations/homeassistant/event"
    method: POST
    headers:
      Authorization: "Bearer al_live_xxxxxxxx"
      Content-Type: "application/json"
    payload: >
      {"event_type":"odometer","vehicle_id":"<id>","data":{"odometer_km":{{ states('sensor.car_odometer') | float }}}}
```

Node-RED / n8n: an HTTP Request node, method `POST`, the URL above, the `Authorization` header and the JSON body of the event.
