# Connecting an assistant (MCP)

AutoLedger can serve a [Model Context Protocol](https://modelcontextprotocol.io) endpoint, so an assistant (Claude, Cursor, any MCP client) can answer questions about the costs of your vehicles. It only reads: nothing it can call changes the account. Français : [mcp.fr.md](mcp.fr.md).

## Enable it

Set `AUTOLEDGER_MCP_ENABLED=true` (in `.env` with Docker Compose) and `AUTOLEDGER_BASE_URL` to the public address of the instance (the address the assistant will use, with `https`). The endpoint is `<base URL>/mcp`. Until the variable is set, the endpoint and the authorization pages do not exist.

The reverse proxy must forward the `Authorization` header and the original `Host` header, as it does for the rest of the application.

## Connect a client

Give the client the endpoint address. It discovers the authorization server by itself, registers, and opens the AutoLedger consent page in the browser; approving it creates the access (see [Authorizing an assistant](oauth.md)).

```
claude mcp add --transport http autoledger https://ledger.example/mcp
```

Then authenticate from the client (`/mcp` in Claude Code). Clients that ask for a token instead of running the authorization (a script, an automation) can use a read-only API token created on the account page, sent as `Authorization: Bearer al_live_...`. A token with the `ingest` scope is refused: it can push events, and an assistant that reads must not hold it.

The access is listed in the API tokens of the account page, and deleting it there cuts the assistant off at once.

## Check the endpoint

A client that cannot verify the server usually cannot reach one of these addresses. From a machine outside your network:

```
curl -i -X POST https://ledger.example/mcp
curl https://ledger.example/.well-known/oauth-protected-resource/mcp
curl https://ledger.example/.well-known/oauth-authorization-server
```

The first must answer `401` with a `WWW-Authenticate: Bearer resource_metadata="..."` header, and the other two a JSON document whose addresses start with your public URL. An HTML page, a `404` or an address such as `http://localhost:8080` means `AUTOLEDGER_MCP_ENABLED` is not set, the proxy does not forward the request, or `AUTOLEDGER_BASE_URL` is not the public address. A client that runs in a cloud service (a web connector, for instance) can only reach an address that is reachable from the internet.

## Tools

| Tool | Returns |
|---|---|
| `list_vehicles` | The vehicles of the account: `id`, `name`, `make`, `model`, `powertrain`, `currency`, `odometer`, `role` (`owner`, or `editor` / `viewer` for a vehicle shared with the account), `tracking_mode` (`connected`, `semi_auto` or `manual`) and, when the acquisition is entered, `acquisition_type` and `owned_since`. |
| `get_vehicle_costs` | What a vehicle cost over its last 12 months and since it was bought, by category, with the cost per km and a completeness score. Takes a `vehicle_id`. |
| `get_vehicle_data_gaps` | What is missing from the records: odometer gaps and anomalies, charges without a cost, unqualified drives, untracked distance, missing insurance or acquisition, and what to complete ranked by what it is worth. Takes a `vehicle_id`. |
| `list_vehicle_trips` | The trips of a vehicle, most recent first: start and end time, `from` and `to` addresses, distance, duration, energy, tolls and parking, tags. Takes a `vehicle_id` and optionally `search` (text in an address, to find "the trip to Lyon"), `from` and `to` (days, `YYYY-MM-DD`, UTC) and `limit` (20 by default, 50 at most); `matching` says how many trips fit. |

Wherever a tool takes a `vehicle_id`, the name of the vehicle works too when no other vehicle has the same; otherwise the error lists the vehicles of the account.

The tools return the same views as the [read API](ingestion-api.md#read-api), with the same conventions: an amount is a decimal string in the currency of its vehicle and is never converted, a distance is in kilometres, an unknown figure is `null`. VINs, plates, positions, documents and drive identifiers are never returned. The one place a street address appears is `list_vehicle_trips`: it returns the departure and arrival addresses the account recorded, so an assistant can talk about a trip by its destination. An address stored only as coordinates is returned as `null`. Anything you authorize an assistant to read can be repeated in its answers and kept by its provider: do not authorize one for an account whose trip addresses you would not share. A vehicle the account cannot see is reported as not found.

## Notes

- The endpoint is stateless: every request carries its own token and nothing is kept between requests.
- Requests are limited to 120 per minute and address.
- A read-only demo instance (`AUTOLEDGER_DEMO`) refuses the requests the endpoint needs.
