import { createHash, randomBytes } from 'node:crypto'
import { expect, test, type APIRequestContext, type Page } from '@playwright/test'
import { expectNoHorizontalScroll, login } from './helpers'

// The server runs with AUTOLEDGER_MCP_ENABLED=true (see CLAUDE.md): an assistant registers, sends the user to the
// consent page, exchanges the code for a read-only token and calls the MCP endpoint with it.
const CALLBACK = 'http://127.0.0.1:7777/callback'

function pkce() {
  const verifier = randomBytes(32).toString('base64url')
  return { verifier, challenge: createHash('sha256').update(verifier).digest('base64url') }
}

async function registerClient(request: APIRequestContext, name: string): Promise<string> {
  const res = await request.post('/oauth/register', { data: { client_name: name, redirect_uris: [CALLBACK], token_endpoint_auth_method: 'none' } })
  expect(res.status(), 'the server must run with AUTOLEDGER_MCP_ENABLED=true').toBe(201)
  return (await res.json()).client_id
}

function authorizeUrl(clientId: string, challenge: string, state: string) {
  const q = new URLSearchParams({ client_id: clientId, redirect_uri: CALLBACK, response_type: 'code', code_challenge: challenge, code_challenge_method: 'S256', state })
  return `/oauth/authorize?${q}`
}

// The assistant listens on a local address; the test answers for it and keeps what the browser was sent back with.
async function captureCallback(page: Page): Promise<() => URL> {
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

test('an assistant gets a read-only token once the user approves, and the user can end it', async ({ page, request }) => {
  const clientId = await registerClient(request, 'Test assistant')
  const { verifier, challenge } = pkce()
  const callback = await captureCallback(page)

  await page.goto(authorizeUrl(clientId, challenge, 'state-1'))
  await expect(page.getByRole('heading', { name: 'Authorize an application' })).toBeVisible()
  await expect(page.getByText('Test assistant asks to read your AutoLedger data.')).toBeVisible()
  await expect(page.getByText('127.0.0.1:7777')).toBeVisible()
  await expect(page.getByText('Change, add or delete anything.')).toBeVisible()
  await page.setViewportSize({ width: 320, height: 800 })
  await expectNoHorizontalScroll(page)

  await page.getByRole('button', { name: 'Approve' }).click()
  await expect.poll(() => { try { return callback().searchParams.get('code') } catch { return null } }).not.toBeNull()
  const returned = callback()
  expect(returned.searchParams.get('state')).toBe('state-1')

  const exchanged = await request.post('/oauth/token', {
    form: { grant_type: 'authorization_code', code: returned.searchParams.get('code')!, client_id: clientId, redirect_uri: CALLBACK, code_verifier: verifier },
  })
  expect(exchanged.ok()).toBeTruthy()
  const { access_token: token, refresh_token: refresh, scope } = await exchanged.json()
  expect(scope).toBe('read')
  const bearer = { Authorization: `Bearer ${token}` }

  const vehicles = await request.get('/api/integrations/v1/vehicles', { headers: bearer })
  expect(vehicles.ok()).toBeTruthy()
  const list = await vehicles.json()
  expect(list.length).toBeGreaterThan(0)
  expect(JSON.stringify(list)).not.toMatch(/"vin"|"plate"/)
  const costs = await request.get(`/api/integrations/v1/vehicles/${list[0].id}/costs`, { headers: bearer })
  expect(costs.ok()).toBeTruthy()
  expect((await costs.json()).last_12_months.costs.total.currency).toBe(list[0].currency)

  // The same token opens the MCP endpoint: tools see the same narrow views as the read API.
  const mcp = (headers: Record<string, string>, body: object) =>
    request.post('/mcp', { headers: { Accept: 'application/json, text/event-stream', ...headers }, data: body })
  const unauthenticated = await mcp({}, { jsonrpc: '2.0', id: 1, method: 'tools/list' })
  expect(unauthenticated.status()).toBe(401)
  expect(unauthenticated.headers()['www-authenticate']).toContain('/.well-known/oauth-protected-resource/mcp')
  const resource = await (await request.get('/.well-known/oauth-protected-resource/mcp')).json()
  expect(resource.authorization_servers).toHaveLength(1)
  const called = await mcp(bearer, { jsonrpc: '2.0', id: 2, method: 'tools/call', params: { name: 'list_vehicles', arguments: {} } })
  expect(called.ok()).toBeTruthy()
  const result = (await called.json()).result
  expect(result.isError).toBeFalsy()
  expect(result.structuredContent.vehicles.map((v: { id: string }) => v.id)).toEqual(list.map((v: { id: string }) => v.id))
  expect(JSON.stringify(result)).not.toMatch(/"vin"|"plate"/)

  const push = await request.post('/api/integrations/homeassistant/event', { headers: bearer, data: { event_type: 'odometer' } })
  expect(push.status(), 'a read grant cannot push events').toBe(403)

  // The grant is listed on the account page as a connected application and ends when revoked there.
  await page.setViewportSize({ width: 1280, height: 900 })
  await page.goto('/account')
  const row = page.locator('li', { hasText: 'Test assistant' })
  await expect(row.getByText('Connected app')).toBeVisible()
  await row.getByRole('button').click()
  await page.getByRole('dialog').getByRole('button', { name: 'Revoke' }).click()
  await expect(row).toHaveCount(0)
  expect((await request.get('/api/integrations/v1/vehicles', { headers: bearer })).status()).toBe(401)
  expect((await mcp(bearer, { jsonrpc: '2.0', id: 3, method: 'tools/list' })).status()).toBe(401)
  const renewed = await request.post('/oauth/token', { form: { grant_type: 'refresh_token', client_id: clientId, refresh_token: refresh } })
  expect(renewed.status()).toBe(400)
})

test('denying sends the assistant back without a code', async ({ page, request }) => {
  const clientId = await registerClient(request, 'Denied assistant')
  const callback = await captureCallback(page)
  await page.goto(authorizeUrl(clientId, pkce().challenge, 'state-2'))
  await page.getByRole('button', { name: 'Deny' }).click()
  await expect.poll(() => { try { return callback().searchParams.get('error') } catch { return null } }).toBe('access_denied')
  expect(callback().searchParams.get('code')).toBeNull()
})

test('an invalid request is explained and offers no approval', async ({ page, request }) => {
  const clientId = await registerClient(request, 'Tampered assistant')
  const url = authorizeUrl(clientId, pkce().challenge, 's').replace(encodeURIComponent(CALLBACK), encodeURIComponent('https://evil.example/cb'))
  await page.goto(url)
  await expect(page.getByRole('alert')).toContainText('does not match the one the application registered')
  await expect(page.getByRole('button', { name: 'Approve' })).toHaveCount(0)
})

test.describe('signed out', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('signing in brings the user back to the consent page', async ({ page, request }) => {
    const clientId = await registerClient(request, 'Resumed assistant')
    await page.goto(authorizeUrl(clientId, pkce().challenge, 'state-3'))
    await expect(page).toHaveURL(/\/login/)
    await login(page)
    await expect(page).toHaveURL(/\/oauth\/authorize\?/)
    await expect(page.getByText('Resumed assistant asks to read your AutoLedger data.')).toBeVisible()
  })
})
