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
| `list_vehicles` | The vehicles of the account: `id`, `name`, `make`, `model`, `powertrain`, `currency`, `odometer`. |
| `get_vehicle_costs` | What a vehicle cost over its last 12 months and since it was bought, by category, with the cost per km and a completeness score. Takes a `vehicle_id`. |
| `get_vehicle_data_gaps` | What is missing from the records: odometer gaps and anomalies, charges without a cost, unqualified drives, untracked distance, missing insurance or acquisition. Takes a `vehicle_id`. |

The tools return the same views as the [read API](ingestion-api.md#read-api), with the same conventions: an amount is a decimal string in the currency of its vehicle and is never converted, a distance is in kilometres, an unknown figure is `null`. VINs, plates, addresses, positions, documents and drive identifiers are never returned. A vehicle the account cannot see is reported as not found.

## Notes

- The endpoint is stateless: every request carries its own token and nothing is kept between requests.
- Requests are limited to 120 per minute and address.
- A read-only demo instance (`AUTOLEDGER_DEMO`) refuses the requests the endpoint needs.
