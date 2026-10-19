# Authorizing an assistant (OAuth)

An assistant or any other application that speaks OAuth 2.1 can ask for read-only access to an account without the user copying a token around. The instance has to enable it with `AUTOLEDGER_MCP_ENABLED=true` (off by default). The assistant then calls the [MCP endpoint](mcp.md). Français : [oauth.fr.md](oauth.fr.md).

## What the user sees

The application sends the browser to `/oauth/authorize` on the instance. After signing in, the user reads what the application asks for and approves or denies it:

- It can see the vehicles (name, make, model, powertrain, currency, odometer), what each one costs over the last 12 months and since purchase, and what is missing from the records. It also sees the names and places recorded: trip addresses, charging places, drivers and carpool passengers, tags and notes.
- It cannot change, add or delete anything, and it never sees VINs, plates, GPS positions or documents.

The application name is given by the application itself and is not verified: approve only an application you just started. The page shows where the user is sent back to.

Approving creates a grant that appears in the API tokens list of the account page, marked as an OAuth grant. Deleting it there ends the access at once: the access token and the refresh token stop working.

## What the application gets

An access token of the form `al_live_...` with the `read` scope, valid for one hour, and a refresh token (`al_refresh_...`) valid for 90 days, renewed at every use. The access token opens the [read API](ingestion-api.md#read-api) and nothing else. There is one grant per user and application: authorizing again replaces the previous one.

A refresh token works once. Presenting an already used refresh token revokes the whole grant, so a stolen copy cannot be used quietly alongside the real one.

## Endpoints

| Endpoint | Purpose |
|---|---|
| `GET /.well-known/oauth-authorization-server` | Server metadata (RFC 8414). |
| `POST /oauth/register` | Dynamic client registration (RFC 7591). Public clients only, at most 1000 clients per instance and 10 registrations per hour and address. A client with no grant is deleted after 30 days. |
| `/oauth/authorize` | Consent page (served by the web app). |
| `POST /oauth/token` | Exchanges an authorization code or a refresh token. |
| `POST /oauth/revoke` | Revokes a grant from a token (RFC 7009). |

Rules enforced by the server:

- Authorization code flow with PKCE, method `S256` only. The code is valid for 60 seconds and is used once.
- Public clients only (`token_endpoint_auth_method` `none`); the only scope is `read`.
- A redirect URI is `https`, a loopback address with any port (`http://127.0.0.1:port/...`), or a private-use scheme of a desktop application (`cursor://...`). Fragments, wildcards and credentials in the URI are refused, and so are `javascript:`, `data:` and `file:`.
- When the request carries a `resource` parameter (RFC 8707), it must be the instance's MCP URL, `<base URL>/mcp`.
- The redirect carries an `iss` parameter (RFC 9207) with the instance URL.

The metadata, registration, token and revocation endpoints and the MCP endpoint answer a request from any origin (`Access-Control-Allow-Origin: *`, no credentials), so a client that runs in a browser can use them. They read no cookie: the credential is the bearer token or the authorization code with its PKCE verifier. The rest of the API keeps refusing requests from another origin.
