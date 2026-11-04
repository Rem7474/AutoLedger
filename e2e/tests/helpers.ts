import { expect, type APIRequestContext, type Page } from '@playwright/test'

export const email = process.env.DEMO_EMAIL ?? 'demo@autoledger.example'
export const password = process.env.DEMO_PASSWORD ?? ''

export interface Vehicle {
  id: string
  name: string
  powertrain: 'EV' | 'ICE' | 'PHEV' | 'REEV'
}

// The login route answers 10 requests a minute per address, and the suite asks for a token in most of its tests: one
// login per worker, kept for less than the life of the token, so that a burst of tests cannot be refused.
const TOKEN_REUSE_MS = 10 * 60 * 1000
let cachedHeaders: { headers: { Authorization: string }; at: number } | null = null

export async function authHeaders(request: APIRequestContext): Promise<{ Authorization: string }> {
  if (cachedHeaders && Date.now() - cachedHeaders.at < TOKEN_REUSE_MS) return cachedHeaders.headers
  const login = await request.post('/api/auth/login', { data: { email, password } })
  expect(login.ok()).toBeTruthy()
  const { token, access_token: accessToken } = await login.json()
  const headers = { Authorization: `Bearer ${token ?? accessToken}` }
  cachedHeaders = { headers, at: Date.now() }
  return headers
}

export async function vehicles(request: APIRequestContext): Promise<Vehicle[]> {
  const res = await request.get('/api/vehicles', { headers: await authHeaders(request) })
  const body = await res.json()
  return Array.isArray(body) ? body : body.vehicles
}

export const authFile = '.auth/state.json'

export async function login(page: Page) {
  await page.addInitScript(() => localStorage.setItem('teslacost_locale', 'en'))
  await page.goto('/login')
  await page.fill('#login-email', email)
  await page.fill('#login-password', password)
  await page.locator('button[type=submit]').click()
  await page.waitForURL((url) => !url.pathname.includes('login'))
}

// Opens the app as the signed-in demo user with the given vehicle active (the key is read at startup).
export async function useVehicle(page: Page, vehicleId: string, route = '/') {
  await page.goto('/login')
  await page.evaluate(([id]) => {
    localStorage.setItem('teslacost_locale', 'en')
    localStorage.setItem('teslacost_active_vehicle', id)
  }, [vehicleId])
  await page.goto(route)
  await page.waitForLoadState('networkidle')
}

export async function expectNoHorizontalScroll(page: Page) {
  const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
  expect(overflow).toBeLessThanOrEqual(0)
}
