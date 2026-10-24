import { createHash, randomBytes } from 'node:crypto'
import { expect, type APIRequestContext, type Page } from '@playwright/test'

// The server runs with AUTOLEDGER_MCP_ENABLED=true (see CLAUDE.md): an assistant registers, sends the user to the
// consent page, exchanges the code for a read-only token and calls the MCP endpoint with it.
export const CALLBACK = 'http://127.0.0.1:7777/callback'

export function pkce() {
  const verifier = randomBytes(32).toString('base64url')
  return { verifier, challenge: createHash('sha256').update(verifier).digest('base64url') }
}

export async function registerClient(request: APIRequestContext, name: string): Promise<string> {
  const res = await request.post('/oauth/register', { data: { client_name: name, redirect_uris: [CALLBACK], token_endpoint_auth_method: 'none' } })
  expect(res.status(), 'the server must run with AUTOLEDGER_MCP_ENABLED=true').toBe(201)
  return (await res.json()).client_id
}

export function authorizeUrl(clientId: string, challenge: string, state: string) {
  const q = new URLSearchParams({ client_id: clientId, redirect_uri: CALLBACK, response_type: 'code', code_challenge: challenge, code_challenge_method: 'S256', state })
  return `/oauth/authorize?${q}`
}

// The assistant listens on a local address; the test answers for it and keeps what the browser was sent back with.
export async function captureCallback(page: Page): Promise<() => URL> {
  let seen: URL | undefined
  await page.route(`${CALLBACK}**`, (route) => {
    seen = new URL(route.request().url())
    return route.fulfill({ status: 200, contentType: 'text/html', body: 'ok' })
  })
  return () => {
    expect(seen, 'the browser was sent back to the assistant').toBeDefined()
    return seen!
  }
}
