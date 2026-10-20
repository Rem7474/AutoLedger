import { expect, test } from '@playwright/test'
import { authHeaders, useVehicle, vehicles, type Vehicle } from './helpers'

// A phone: touch pointer, so the 44px hit areas and the 16px fields of main.css apply.
test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true })

let ev: Vehicle

test.beforeAll(async ({ request }) => {
  ev = (await vehicles(request)).find((v) => v.powertrain === 'EV')!
  expect(ev).toBeTruthy()
})

for (const route of ['/drives', '/carpools', '/comparison', '/tires', '/maintenance']) {
  test(`every checkbox has a 44px hit area on ${route}`, async ({ page }) => {
    await useVehicle(page, ev.id, route)
    const small = await page.evaluate(() =>
      [...document.querySelectorAll<HTMLInputElement>('input[type=checkbox]:not([role=switch])')]
        .filter((e) => e.getBoundingClientRect().width > 0 && !e.closest('.sr-only'))
        .map((e) => ({ id: e.id || e.getAttribute('aria-label') || '', box: (e.closest('label, .tap') ?? e).getBoundingClientRect() }))
        .filter((x) => x.box.height < 44)
        .map((x) => `${x.id} ${Math.round(x.box.width)}x${Math.round(x.box.height)}`),
    )
    expect(small).toEqual([])
  })
}

test('a date field inside a form is 16px so the page does not zoom on focus', async ({ page }) => {
  await useVehicle(page, ev.id, '/expenses')
  await page.getByRole('button', { name: /^Add/ }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  const sizes = await dialog.locator('input.dp--input').evaluateAll((els) => els.map((e) => getComputedStyle(e).fontSize))
  expect(sizes.length).toBeGreaterThan(0)
  expect(sizes).toEqual(sizes.map(() => '16px'))
})

test('a carpool is built in three steps on a phone: legs, passengers, then the recap', async ({ page, request }) => {
  // The carpool of an interrupted run is removed first
  const headers = await authHeaders(request)
  const existing = await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()
  for (const trip of (existing.trips ?? []).filter((c: { title: string }) => c.title === 'E2E phone carpool')) {
    await request.delete(`/api/vehicles/${ev.id}/carpools/${trip.id}`, { headers })
  }

  await useVehicle(page, ev.id, '/carpools')
  await page.getByRole('button', { name: 'New carpool' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'New carpool' })

  // Step 1: the legs; the title and the passengers are not on screen yet
  await expect(dialog.locator('#carpool-title')).toHaveCount(0)
  await dialog.locator('#leg-start-0').fill('Lyon')
  await dialog.locator('#leg-end-0').fill('Annecy')
  await dialog.locator('#leg-distance-0').fill('100')
  await dialog.locator('#leg-electricity_cost-0').fill('20')
  await dialog.locator('#leg-tolls_cost-0').fill('10')
  await dialog.getByRole('button', { name: 'Continue' }).click()

  // Step 2: who is on board, then what they paid; the recap bar already shows what is left to the driver
  await expect(dialog.getByRole('group', { name: 'Who is on board on each leg' })).toBeVisible()
  await expect(dialog.locator('#leg-start-0')).toHaveCount(0)
  await dialog.locator('#passenger-name-0').fill('Lea')
  await dialog.getByRole('button', { name: 'Exact amount' }).click()
  await expect(dialog.locator('#passenger-paid-0')).toHaveValue('15')
  await expect(dialog.getByText('Left to the driver')).toBeVisible()
  await dialog.getByRole('button', { name: 'Continue' }).click()

  // Step 3: the recap, the title, then save
  await dialog.locator('#carpool-title').fill('E2E phone carpool')
  await expect(dialog.getByText('€30.00').first()).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(390)
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByText('E2E phone carpool').first()).toBeVisible()

  const created = await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()
  for (const trip of created.trips.filter((c: { title: string }) => c.title === 'E2E phone carpool')) {
    await request.delete(`/api/vehicles/${ev.id}/carpools/${trip.id}`, { headers })
  }
})
