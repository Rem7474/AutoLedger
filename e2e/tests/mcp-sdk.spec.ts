import { Client } from '@modelcontextprotocol/sdk/client/index.js'
import { UnauthorizedError, type OAuthClientProvider } from '@modelcontextprotocol/sdk/client/auth.js'
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp.js'
import type { OAuthClientInformationMixed, OAuthClientMetadata, OAuthTokens } from '@modelcontextprotocol/sdk/shared/auth.js'
import { expect, test, type Page } from '@playwright/test'
import { CALLBACK, captureCallback } from './oauthFlow'

// The server runs with AUTOLEDGER_MCP_ENABLED=true and the demo dataset (see CLAUDE.md). The client here is the
// official MCP SDK, so discovery, dynamic registration, PKCE, token exchange and output validation are the ones a real
// assistant performs; only the browser step (the consent page) is driven by the test.

// The state a client keeps between its requests.
class MemoryProvider implements OAuthClientProvider {
  info?: OAuthClientInformationMixed
  saved?: OAuthTokens
  verifier = ''
  authorizationUrl?: URL

  constructor(private readonly name: string) {}

  get redirectUrl() { return CALLBACK }
  get clientMetadata(): OAuthClientMetadata {
    return { client_name: this.name, redirect_uris: [CALLBACK], token_endpoint_auth_method: 'none', grant_types: ['authorization_code', 'refresh_token'], response_types: ['code'], scope: 'read' }
  }
  clientInformation() { return this.info }
  saveClientInformation(info: OAuthClientInformationMixed) { this.info = info }
  tokens() { return this.saved }
  saveTokens(tokens: OAuthTokens) { this.saved = tokens }
  redirectToAuthorization(url: URL) { this.authorizationUrl = url }
  saveCodeVerifier(verifier: string) { this.verifier = verifier }
  codeVerifier() { return this.verifier }
}

// Connects the way an assistant does: the first request is refused, the user approves on the consent page, the client
// exchanges the code and connects again with its token. `choose` acts on the consent page before approval.
async function connect(page: Page, baseURL: string, name: string, choose?: (page: Page) => Promise<void>) {
  const provider = new MemoryProvider(name)
  const endpoint = new URL('/mcp', baseURL)
  const callback = await captureCallback(page)

  const first = new StreamableHTTPClientTransport(endpoint, { authProvider: provider })
  await expect(new Client({ name, version: '1.0.0' }).connect(first)).rejects.toBeInstanceOf(UnauthorizedError)
  expect(provider.authorizationUrl, 'the client was sent to the consent page').toBeDefined()
  expect(provider.info?.client_id, 'the client registered itself').toBeTruthy()

  await page.goto(provider.authorizationUrl!.toString())
  await expect(page.getByText(`${name} asks to read your AutoLedger data.`)).toBeVisible()
  await choose?.(page)
  await page.getByRole('button', { name: 'Approve' }).click()
  await expect.poll(() => { try { return callback().searchParams.get('code') } catch { return null } }).not.toBeNull()
  await first.finishAuth(callback().searchParams.get('code')!)
  expect(provider.saved?.access_token).toBeTruthy()

  const client = new Client({ name, version: '1.0.0' })
  await client.connect(new StreamableHTTPClientTransport(endpoint, { authProvider: provider }))
  return client
}

type Structured = Record<string, any>
type Call = { isError: boolean; data: Structured; text: string }

async function call(client: Client, name: string, args: Record<string, unknown> = {}): Promise<Call> {
  // The client validates structuredContent against the tool's outputSchema.
  const res = await client.callTool({ name, arguments: args })
  return { isError: res.isError === true, data: (res.structuredContent ?? {}) as Structured, text: JSON.stringify(res.content) }
}

// What each tool needs, read from its own input schema so that a tool added later is called without editing this file.
function argumentsFor(tool: { name: string; inputSchema: { required?: string[] } }, vehicleId: string): Record<string, unknown> {
  const known: Record<string, unknown> = { vehicle_id: vehicleId, month: new Date().toISOString().slice(0, 7), category: 'energy', energy_kwh: 10, duration_minutes: 60 }
  const args: Record<string, unknown> = {}
  for (const key of tool.inputSchema.required ?? []) {
    expect(known, `${tool.name} requires "${key}": give this test a value for it`).toHaveProperty(key)
    args[key] = known[key]
  }
  return args
}

test('an assistant built on the official SDK authorizes, lists and calls every tool on the demo dataset', async ({ page, baseURL }) => {
  test.setTimeout(120_000)
  const client = await connect(page, baseURL!, 'SDK assistant')

  const { tools } = await client.listTools()
  expect(tools.length).toBeGreaterThan(0)
  for (const tool of tools) {
    expect(tool.annotations?.readOnlyHint, `${tool.name} must be read-only`).toBe(true)
    expect(tool.outputSchema, `${tool.name} declares what it returns`).toBeDefined()
  }

  const listed = await call(client, 'list_vehicles')
  expect(listed.isError).toBe(false)
  const vehicles: { id: string; currency: string }[] = listed.data.vehicles
  expect(vehicles.length).toBeGreaterThan(0)

  for (const tool of tools) {
    const res = await call(client, tool.name, argumentsFor(tool, vehicles[0].id))
    expect(res.isError, `${tool.name}: ${res.text}`).toBe(false)
    expect(Object.keys(res.data).length, `${tool.name} answers with structured content`).toBeGreaterThan(0)
    expect(res.text, `${tool.name} never exposes a VIN or a plate`).not.toMatch(/"vin"|"plate"/i)
  }

  // A vehicle outside the account is not found, whatever the tool.
  const stranger = await call(client, 'get_vehicle_costs', { vehicle_id: '00000000-0000-0000-0000-000000000000' })
  expect(stranger.isError).toBe(true)

  const { prompts } = await client.listPrompts()
  expect(prompts.length).toBeGreaterThan(0)
  const { resources } = await client.listResources()
  const uris = resources.map((r) => r.uri)
  expect(uris).toEqual(expect.arrayContaining(['autoledger://glossary', 'autoledger://account']))
  for (const uri of ['autoledger://glossary', 'autoledger://account']) {
    const { contents } = await client.readResource({ uri })
    expect(contents.length, uri).toBeGreaterThan(0)
  }
  await client.close()
})

test('a grant limited to one vehicle sees only that vehicle', async ({ page, baseURL }) => {
  test.setTimeout(120_000)
  let chosen = ''
  const client = await connect(page, baseURL!, 'Limited assistant', async (consent) => {
    await consent.getByLabel('Only the vehicles I choose').check()
    const boxes = consent.locator('input[type=checkbox]')
    expect(await boxes.count(), 'the demo account has several vehicles to choose from').toBeGreaterThan(1)
    chosen = (await boxes.first().locator('xpath=ancestor::label').innerText()).trim()
    await boxes.first().check()
  })

  const listed = await call(client, 'list_vehicles')
  expect(listed.isError).toBe(false)
  expect(listed.data.vehicles).toHaveLength(1)
  const own = listed.data.vehicles[0]
  expect(chosen).toContain(own.name)

  expect((await call(client, 'get_vehicle_costs', { vehicle_id: own.id })).isError).toBe(false)
  // The household view adds up every vehicle, so a limited grant is refused it.
  const fleet = await call(client, 'get_fleet_summary')
  expect(fleet.isError).toBe(true)
  expect(fleet.text).toContain('limited to some vehicles')
  await client.close()
})
