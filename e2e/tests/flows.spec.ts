import { expect, test } from '@playwright/test'
import { authHeaders, expectNoHorizontalScroll, useVehicle, vehicles, type Vehicle } from './helpers'

test('what-if allows confirming an unusually high energy price without changing the saved scenario', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Price warning reference', powertrain: 'EV', currency: 'EUR' } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  let scenarioId: string | undefined
  try {
    for (let i = 0; i < 5; i++) {
      const charge = await request.post(`/api/vehicles/${vehicle.id}/charges`, { headers, data: { date: '2026-01-01T12:00:00Z', kwh_added: 10, cost: 2, currency: 'EUR' } })
      expect(charge.ok()).toBeTruthy()
    }
    const name = 'Relative price warning'
    const saved = await request.post('/api/comparison-scenarios/', { headers, data: {
      name, mode: 'PROJECTION', currency: 'EUR', annual_km: 12000, years: 5,
      ice: { fuel_type: 'SP95_E10', l_per_100km: 6.5, fuel_price: 1.7 },
      tracked: { kwh_per_100km: 16, eur_per_kwh: 0.2 },
    } })
    expect(saved.ok()).toBeTruthy()
    scenarioId = (await saved.json()).id
    await useVehicle(page, vehicle.id, '/comparison')
    const history = page.waitForResponse(r => r.url().includes('/comparison-scenarios/price-history'))
    await page.getByText(name, { exact: true }).click()
    expect((await history).ok()).toBeTruthy()
    await page.locator('#wi-elec').fill('0.8')
    await expect(page.getByText(/more than 3 times/)).toBeVisible()
    const save = page.getByRole('button', { name: 'Save as a new comparison' })
    await expect(save).toBeDisabled()
    await page.setViewportSize({ width: 320, height: 800 })
    await expectNoHorizontalScroll(page)
    const preview = page.waitForResponse(r => r.url().endsWith('/comparison-scenarios/preview'))
    await page.getByRole('button', { name: 'Keep this price' }).click()
    expect((await preview).ok()).toBeTruthy()
    await expect(save).toBeEnabled()
    await expect(page.locator('#wi-elec')).toHaveValue('0.8')
    await page.locator('#wi-elec').fill('0.9')
    await expect(page.getByRole('button', { name: 'Keep this price' })).toBeVisible()
    const list = await request.get('/api/comparison-scenarios/', { headers })
    const original = (await list.json()).find((s: any) => s.id === scenarioId)
    expect(original.tracked.eur_per_kwh).toBe(0.2)
  } finally {
    if (scenarioId) await request.delete(`/api/comparison-scenarios/${scenarioId}`, { headers })
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
  }
})

test('what-if asks before keeping an unusually high fuel price', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Fuel warning reference', powertrain: 'ICE', currency: 'EUR' } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  let scenarioId: string | undefined
  try {
    for (let i = 0; i < 5; i++) {
      const fill = await request.post(`/api/vehicles/${vehicle.id}/fuel-logs`, { headers, data: { date: '2026-01-01T12:00:00Z', amount: 17, liters: 10, fuel_type: 'SP95_E10' } })
      expect(fill.ok()).toBeTruthy()
    }
    const name = 'Relative fuel warning'
    const saved = await request.post('/api/comparison-scenarios/', { headers, data: {
      name, mode: 'PROJECTION', currency: 'EUR', annual_km: 12000, years: 5,
      ice: { fuel_type: 'SP95_E10', l_per_100km: 6.5, fuel_price: 1.7 },
      tracked: { kwh_per_100km: 16, eur_per_kwh: 0.2 },
    } })
    expect(saved.ok()).toBeTruthy()
    scenarioId = (await saved.json()).id
    await useVehicle(page, vehicle.id, '/comparison')
    const history = page.waitForResponse(r => r.url().includes('/comparison-scenarios/price-history'))
    await page.getByText(name, { exact: true }).click()
    expect((await history).ok()).toBeTruthy()
    await page.locator('#wi-fuel').fill('6')
    await expect(page.getByText(/more than 3 times/)).toBeVisible()
    await expect(page.getByRole('button', { name: 'Save as a new comparison' })).toBeDisabled()
    await page.getByRole('button', { name: 'Keep this price' }).click()
    await expect(page.getByRole('button', { name: 'Save as a new comparison' })).toBeEnabled()
    await expect(page.locator('#wi-fuel')).toHaveValue('6')
  } finally {
    if (scenarioId) await request.delete(`/api/comparison-scenarios/${scenarioId}`, { headers })
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
  }
})

test('an adjusted scenario is saved in place or as another comparison from the result page', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Adjust scenario reference', powertrain: 'EV', currency: 'EUR' } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  const ids: string[] = []
  try {
    const name = 'Adjusted in place'
    const saved = await request.post('/api/comparison-scenarios/', { headers, data: {
      name, mode: 'PROJECTION', currency: 'EUR', annual_km: 12000, years: 5,
      ice: { fuel_type: 'SP95_E10', l_per_100km: 6.5, fuel_price: 1.7 },
      tracked: { kwh_per_100km: 16, eur_per_kwh: 0.2 },
    } })
    expect(saved.ok()).toBeTruthy()
    const scenarioId = (await saved.json()).id
    ids.push(scenarioId)
    await useVehicle(page, vehicle.id, '/comparison')
    await page.getByText(name, { exact: true }).click()
    const save = page.getByTestId('what-if-save')
    await expect(save).toBeDisabled()

    // A field outside the three sliders is part of the same draft
    await page.getByText(/More assumptions/).click()
    const preview = page.waitForResponse(r => r.url().endsWith('/comparison-scenarios/preview'))
    await page.locator('#wi-years').fill('8')
    expect((await preview).ok()).toBeTruthy()
    await expect(save).toBeEnabled()
    await save.click()
    await expect(save).toBeDisabled()
    const list = await request.get('/api/comparison-scenarios/', { headers })
    const updated = (await list.json()).find((s: any) => s.id === scenarioId)
    expect(updated.years).toBe(8)

    // Save as another comparison leaves the first one untouched
    await page.getByText(/More assumptions/).click()
    await page.locator('#wi-years').fill('3')
    await page.getByRole('button', { name: 'Save as a new comparison' }).click()
    await page.locator('#wi-new-name').fill('Adjusted copy')
    await page.getByRole('button', { name: 'Create' }).click()
    await expect(page.getByText('Adjusted copy').first()).toBeVisible()
    const after = await (await request.get('/api/comparison-scenarios/', { headers })).json()
    const copy = after.find((s: any) => s.name === 'Adjusted copy')
    expect(copy?.years).toBe(3)
    if (copy) ids.push(copy.id)
    expect(after.find((s: any) => s.id === scenarioId).years).toBe(8)
  } finally {
    for (const id of ids) await request.delete(`/api/comparison-scenarios/${id}`, { headers })
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
  }
})

let ev: Vehicle
let ice: Vehicle

test.beforeAll(async ({ request }) => {
  const all = await vehicles(request)
  ev = all.find((v) => v.powertrain === 'EV')!
  ice = all.find((v) => v.powertrain === 'ICE')!
  expect(ev && ice).toBeTruthy()
})

// A loading skeleton is also a status region: only the live announcement is a toast
const toast = (page: import('@playwright/test').Page) =>
  page.locator('[role="status"]:not([aria-busy="true"])').filter({ hasText: /\S/ })

test('quick-add records a charge typed with its cost', async ({ page }) => {
  await useVehicle(page, ev.id, '/energy')
  await page.getByRole('button', { name: 'Quick add' }).first().click()
  await page.fill('#qc-kwh', '37.5')
  await page.fill('#qc-cost', '9.80')
  await page.getByRole('button', { name: 'Save the charge' }).click()
  await expect(toast(page)).toContainText('37.5')
})

test('quick-add records a fill-up on a combustion vehicle', async ({ page }) => {
  await useVehicle(page, ice.id, '/energy')
  await page.getByRole('button', { name: 'Quick add' }).first().click()
  await page.fill('#qf-liters', '40')
  await page.fill('#qf-amount', '72')
  await page.getByRole('button', { name: /^save/i }).click()
  await expect(toast(page)).toContainText('72')
})

test('energy tabs follow the powertrain and the legacy fuel URL redirects', async ({ page }) => {
  await useVehicle(page, ice.id, '/manual?tab=FUEL')
  await expect(page).toHaveURL(/\/energy\?tab=FUEL/)
  await expect(page.getByRole('button', { name: 'New fill-up' })).toBeVisible()
  await expect(page.getByRole('tab', { name: 'Charges' })).toHaveCount(0)

  await useVehicle(page, ev.id, '/energy')
  await expect(page.getByRole('tab', { name: 'Charges' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'New fill-up' })).toHaveCount(0)
})

for (const width of [320, 360, 375, 390, 414]) {
  test(`no horizontal scroll at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 800 })
    for (const route of ['/', '/energy', '/expenses', '/maintenance', '/account']) {
      await useVehicle(page, ev.id, route)
      await expectNoHorizontalScroll(page)
    }
  })
}

const modal = (page: import('@playwright/test').Page) => page.getByRole('dialog', { name: 'Add tires' })

test.describe('a modal with edits does not close on a backdrop click', () => {
  test.beforeEach(async ({ page }) => {
    await useVehicle(page, ev.id, '/tires')
    await page.getByRole('button', { name: 'Add tires' }).first().click()
    await expect(modal(page)).toBeVisible()
  })

  test('typed input is kept and Escape still closes', async ({ page }) => {
    await page.fill('#tire-add-tire-brand', 'Acme')
    await page.mouse.click(4, 4)
    await expect(modal(page)).toBeVisible()
    await expect(page.locator('#tire-add-tire-brand')).toHaveValue('Acme')
    await expect(toast(page)).toContainText('Unsaved changes')
    await page.keyboard.press('Escape')
    await expect(modal(page)).toHaveCount(0)
  })

  test('an untouched modal closes on a backdrop click', async ({ page }) => {
    await page.mouse.click(4, 4)
    await expect(modal(page)).toHaveCount(0)
  })
})

test('deleting a tire asks whether to delete it or scrap it', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Tire dialog reference', powertrain: 'EV', currency: 'EUR' } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  try {
    const tire = await request.post(`/api/vehicles/${vehicle.id}/tires`, { headers, data: {
      brand: 'Zeta', model: 'Grip', dimension: '205/55R16', season: 'SUMMER', purchase_date: '2026-01-01', purchase_price: 9000,
      current_position: 'STORAGE', initial_depth_mm: 8, min_legal_depth_mm: 1.6, estimated_lifespan_km: 40000,
    } })
    expect(tire.ok()).toBeTruthy()
    await page.setViewportSize({ width: 390, height: 844 })
    await useVehicle(page, vehicle.id, '/tires')
    await page.getByRole('tab', { name: /Catalogue and garage stock/ }).click()
    await page.getByRole('button', { name: 'Open Zeta Grip' }).click()
    await page.getByRole('button', { name: 'Delete (entry error)' }).click()

    const dialog = page.getByRole('dialog', { name: 'Delete or scrap this tire?' })
    await expect(dialog.getByRole('button', { name: 'Delete permanently' })).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Cancel' })).toBeVisible()
    await dialog.getByRole('button', { name: 'Scrap it instead' }).click()

    // Scrapping opens the scrap form and keeps the tire
    await expect(page.getByRole('dialog', { name: /Scrap/ })).toBeVisible()
    const still = await request.get(`/api/vehicles/${vehicle.id}/tires`, { headers })
    expect(JSON.stringify(await still.json())).toContain('Zeta')
  } finally {
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
  }
})

test('a vehicle card selects its vehicle and keeps the setup in its menu', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ev.id, '/vehicles')
  const card = page.locator('div.card', { has: page.getByRole('heading', { name: ice.name }) })
  await expect(card.getByRole('button', { name: 'Acquisition and financing' })).toHaveCount(0)
  await card.getByRole('button', { name: `Select ${ice.name}` }).click()
  await expect(card.getByText('Active', { exact: true })).toBeVisible()

  await card.getByRole('button', { name: 'More actions' }).click()
  await card.getByRole('menuitem', { name: 'Data sources' }).click()
  await expect(page.getByRole('dialog', { name: 'Data sources' })).toBeVisible()
  await page.keyboard.press('Escape')

  // The disclosure of the form tells assistive technology whether it is open
  await useVehicle(page, ev.id, '/vehicles')
  const evCard = page.locator('div.card', { has: page.getByRole('heading', { name: ev.name }) })
  await evCard.getByRole('button', { name: 'Edit', exact: true }).click()
  const options = page.getByRole('button', { name: /^Additional options/ })
  await expect(options).toHaveAttribute('aria-expanded', 'false')
  await options.click()
  await expect(options).toHaveAttribute('aria-expanded', 'true')
})

test('the dashboard follow-up zone is closed on a phone and opens on demand', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await useVehicle(page, ev.id, '/')
  const toggle = page.getByRole('button', { name: /^Follow-up/ })
  await expect(toggle).toHaveAttribute('aria-expanded', 'false')
  await expect(page.locator('#dashboard-follow-up')).toBeHidden()
  await toggle.click()
  await expect(toggle).toHaveAttribute('aria-expanded', 'true')
  await expect(page.locator('#dashboard-follow-up')).toBeVisible()
})

test('the dashboard follow-up zone is open on a wide screen', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ev.id, '/')
  await expect(page.getByRole('button', { name: /^Follow-up/ })).toHaveAttribute('aria-expanded', 'true')
})

test('the odometer page lists the mileage typed on fill-ups, read-only', async ({ page }) => {
  await useVehicle(page, ice.id, '/odometer')
  await expect(page.getByText('Fill-up', { exact: true }).first()).toBeVisible()
  const row = page.locator('div.rounded-2xl', { has: page.getByText('Fill-up', { exact: true }) }).first()
  await expect(row.getByRole('button')).toHaveCount(0)
  await expect(row.getByRole('link', { name: 'See the fill-ups' })).toHaveAttribute('href', /\/energy\?tab=FUEL/)
})

test('an electric vehicle lists no fill-up on the odometer page', async ({ page }) => {
  await useVehicle(page, ev.id, '/odometer')
  await expect(page.getByText('History of recorded readings')).toBeVisible()
  await expect(page.getByText('Fill-up', { exact: true })).toHaveCount(0)
})

test('a selectable odometer reading keeps its mileage next to the checkbox on a wide screen', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const notes = `Layout reading ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/odometer-checkpoints`, {
    headers,
    data: { date: new Date().toISOString().slice(0, 10), odometer: 876543, notes },
  })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await useVehicle(page, ev.id, '/odometer')
    const row = page.locator('div.rounded-2xl', { hasText: notes }).last()
    await row.scrollIntoViewIfNeeded()
    const check = (await row.getByRole('checkbox').boundingBox())!
    const text = (await row.getByText(notes).boundingBox())!
    // The content follows the checkbox: it is not pushed to the middle of the row by the space between the two
    expect(text.x - (check.x + check.width)).toBeLessThan(80)
  } finally {
    await request.delete(`/api/vehicles/${ev.id}/odometer-checkpoints/${id}`, { headers })
  }
})

test('the readings of the odometer list line up whether or not they can be selected', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const notes = `Alignment reading ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ice.id}/odometer-checkpoints`, {
    headers,
    data: { date: new Date().toISOString().slice(0, 10), odometer: 876543, notes },
  })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await useVehicle(page, ice.id, '/odometer')
    const manual = page.locator('div.rounded-2xl', { hasText: notes }).last()
    const fillUp = page.locator('div.rounded-2xl', { has: page.getByText('Fill-up', { exact: true }) }).last()
    await fillUp.scrollIntoViewIfNeeded()
    const fillUpText = (await fillUp.locator('.tabular-nums').first().boundingBox())!
    await manual.scrollIntoViewIfNeeded()
    const manualText = (await manual.locator('.tabular-nums').first().boundingBox())!
    // A fill-up has no checkbox but keeps its column, so the mileages start at the same place
    expect(Math.abs(fillUpText.x - manualText.x)).toBeLessThanOrEqual(1)
  } finally {
    await request.delete(`/api/vehicles/${ice.id}/odometer-checkpoints/${id}`, { headers })
  }
})

test('the drive filters sit in a sheet on a phone and in the bar on a wide screen', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await useVehicle(page, ev.id, '/drives')
  const toggle = page.getByRole('button', { name: /^Filters/ })
  const withToll = page.getByRole('button', { name: /With toll/ })
  await expect(toggle).toBeVisible()
  await expect(withToll).toBeHidden()
  await expect(page.getByRole('searchbox', { name: /Search for a city/ })).toBeVisible()
  await toggle.click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await withToll.click()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toBeHidden()
  await expect(withToll).toBeHidden()
  await expect(toggle).toContainText('1')

  await page.setViewportSize({ width: 1280, height: 900 })
  await expect(toggle).toBeHidden()
  await expect(withToll).toBeVisible()
})

test('a modal covers the bottom bar of a phone', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 700 })
  await useVehicle(page, ice.id, '/energy?tab=FUEL')
  await page.getByRole('button', { name: 'New fill-up' }).click()
  await expect(page.getByRole('dialog', { name: 'New fill-up' })).toBeVisible()
  const barIsReachable = await page.evaluate(() => {
    const bar = document.querySelector('nav.fixed')!
    const { left, top, width, height } = bar.getBoundingClientRect()
    const hit = document.elementFromPoint(left + width * 0.1, top + height / 2)
    return !!hit && bar.contains(hit)
  })
  expect(barIsReachable).toBe(false)
})

test.describe('the tire modals share one frame', () => {
  test('add and history open named, close on Escape and on the close button', async ({ page }) => {
    await useVehicle(page, ev.id, '/tires')
    await page.getByRole('button', { name: 'Add tires' }).first().click()
    const add = page.getByRole('dialog', { name: 'Add tires' })
    await expect(add).toBeVisible()
    await page.keyboard.press('Escape')
    await expect(add).toHaveCount(0)

    await page.getByRole('button', { name: /^Open Michelin Pilot Sport 4/ }).first().click()
    const history = page.getByRole('dialog', { name: /Michelin Pilot Sport 4/ })
    await expect(history).toBeVisible()
    await expect(history.getByRole('button', { name: 'Edit the tire' })).toBeVisible()
    await history.getByRole('button', { name: 'Close', exact: true }).first().click()
    await expect(history).toHaveCount(0)
  })
})

test('the expense and vehicle modals open named and close on Escape', async ({ page }) => {
  await useVehicle(page, ev.id, '/expenses')
  await page.getByRole('button', { name: 'Add Toll / parking' }).click()
  const toll = page.getByRole('dialog', { name: 'Add a toll / parking fee' })
  await expect(toll).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(toll).toHaveCount(0)

  await page.getByRole('tab', { name: 'Fixed costs' }).click()
  await page.getByRole('button', { name: 'Add Fixed costs' }).click()
  const maintenance = page.getByRole('dialog', { name: /Add maintenance \/ fixed expense/ })
  await expect(maintenance).toBeVisible()
  await maintenance.getByRole('button', { name: 'Close', exact: true }).first().click()
  await expect(maintenance).toHaveCount(0)

  await useVehicle(page, ev.id, '/vehicles')
  await page.locator('button.btn-primary', { has: page.locator('svg') }).first().click()
  const vehicle = page.getByRole('dialog').first()
  await expect(vehicle).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
})

test('a toll added from the cost modal of a drive is listed, edited and deleted', async ({ page, request }) => {
  // A run that stopped half way must not leave its toll behind for the next one
  const headers = await authHeaders(request)
  const leftovers = await (await request.get(`/api/vehicles/${ev.id}/expenses`, { headers })).json()
  for (const e of leftovers.filter((x: { notes?: string }) => x.notes === 'e2e cost modal')) {
    await request.delete(`/api/vehicles/${ev.id}/expenses/${e.id}`, { headers })
  }

  await useVehicle(page, ev.id, '/drives')
  await page.getByRole('button', { name: /^Open the cost breakdown of the drive of/ }).nth(4).click()
  const dialog = page.getByRole('dialog').last()
  await expect(dialog.getByText('0 cost(s) assigned')).toBeVisible()

  await dialog.getByRole('button', { name: 'Add a toll or parking fee to this drive' }).click()
  await dialog.locator('#drive-inline-toll-amount').fill('3.2')
  await dialog.locator('#drive-inline-toll-notes').fill('e2e cost modal')
  await dialog.getByRole('button', { name: 'Confirm', exact: true }).click()
  await expect(dialog.getByText('1 cost(s) assigned')).toBeVisible()
  await expect(dialog.getByText('€3.20').first()).toBeVisible()

  await dialog.getByRole('button', { name: 'Edit this cost' }).click()
  await dialog.locator('[id^="drive-expense-amount-"]').fill('4.5')
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog.getByText('€4.50').first()).toBeVisible()

  await dialog.getByRole('button', { name: 'Delete this cost' }).click()
  await page.getByRole('dialog', { name: 'Delete the cost' }).getByRole('button', { name: 'Delete', exact: true }).click()
  await expect(dialog.getByText('0 cost(s) assigned')).toBeVisible()
})

test('a carpool is entered with its legs and passengers, priced and saved, then deleted', async ({ page, request }) => {
  // The demo carpool of an interrupted run is removed first
  const headers = await authHeaders(request)
  const existing = await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()
  for (const trip of (existing.trips ?? []).filter((c: { title: string }) => c.title === 'E2E carpool')) {
    await request.delete(`/api/vehicles/${ev.id}/carpools/${trip.id}`, { headers })
  }

  await useVehicle(page, ev.id, '/carpools')
  await page.getByRole('button', { name: 'New carpool' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'New carpool' })
  await dialog.locator('#carpool-title').fill('E2E carpool')
  await dialog.locator('#leg-start-0').fill('Lyon')
  await dialog.locator('#leg-end-0').fill('Annecy')
  await dialog.locator('#leg-distance-0').fill('100')
  await dialog.locator('#leg-electricity_cost-0').fill('20')
  await dialog.locator('#leg-tolls_cost-0').fill('10')

  // 30 of actual cost shared between the driver and one passenger: the fair share is 15
  await expect(dialog.locator('#passenger-name-0')).toHaveAttribute('placeholder', 'Passenger 1')
  await dialog.getByRole('button', { name: 'Exact amount' }).click()
  await expect(dialog.locator('#passenger-paid-0')).toHaveValue('15')
  await dialog.getByRole('button', { name: 'Calculation detail' }).click()
  await expect(dialog.getByText('Hide the calculation')).toBeVisible()
  await expect(dialog.getByText('€30.00').first()).toBeVisible()
  await expect(dialog.getByText('€15.00').first()).toBeVisible()

  await dialog.getByRole('button', { name: 'Add a passenger' }).click()
  await expect(dialog.locator('#passenger-name-1')).toBeVisible()
  await dialog.getByRole('button', { name: 'Remove this passenger' }).last().click()
  await expect(dialog.locator('#passenger-name-1')).toHaveCount(0)

  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByText('E2E carpool').first()).toBeVisible()

  await page.locator('div.rounded-2xl', { hasText: 'E2E carpool' }).getByRole('button', { name: 'More actions' }).first().click()
  await page.getByRole('menuitem', { name: 'Delete', exact: true }).click()
  // No confirmation: the carpool leaves the list and a snackbar offers to take it back until the server is asked
  await expect(page.getByText('E2E carpool')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Undo' })).toBeVisible()
  const listed = async () => ((await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()).trips ?? []).filter((c: { title: string }) => c.title === 'E2E carpool').length
  expect(await listed()).toBe(1)
  await expect.poll(listed, { timeout: 12_000 }).toBe(0)
})

test('carpools are listed 30 at a time and the totals cover all of them', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const mine = async () => {
    const res = await (await request.get(`/api/vehicles/${ev.id}/carpools?limit=100`, { headers })).json()
    return (res.trips ?? []).filter((c: { title: string }) => c.title.startsWith('E2E paged '))
  }
  // Left over by an interrupted run
  for (const trip of await mine()) await request.delete(`/api/vehicles/${ev.id}/carpools/${trip.id}`, { headers })

  const summaryBefore = await (await request.get(`/api/vehicles/${ev.id}/carpools/summary`, { headers })).json()
  // 35 carpools dated far ahead, so they are the latest ones whatever the demo holds
  const count = 35
  for (let i = 0; i < count; i++) {
    const day = String(28 - Math.floor(i / 2)).padStart(2, '0')
    const res = await request.post(`/api/vehicles/${ev.id}/carpools`, {
      headers,
      data: { title: `E2E paged ${String(i).padStart(2, '0')}`, date: `2098-01-${day}`, legs: [{ distance_km: 10, electricity_cost: 1 }], passengers: [{ passenger_name: 'Paged', seats: 1, amount_paid: 0.5 }] },
    })
    expect(res.ok()).toBeTruthy()
  }
  try {
    const summaryAfter = await (await request.get(`/api/vehicles/${ev.id}/carpools/summary`, { headers })).json()
    expect(summaryAfter.total_trips - summaryBefore.total_trips).toBe(count)

    await useVehicle(page, ev.id, '/carpools')
    const rows = page.getByText(/^E2E paged \d\d$/)
    await expect(rows).toHaveCount(30)
    await expect(page.getByText('E2E paged 34')).toHaveCount(0)

    // The sentinel asks for the next page by itself when it nears the screen; its button does the same
    const more = page.getByRole('button', { name: 'Show more carpools' })
    await more.scrollIntoViewIfNeeded()
    await expect(page.getByText('E2E paged 34')).toBeVisible()
    await expect(rows).toHaveCount(count)
    await expectNoHorizontalScroll(page)
  } finally {
    for (const trip of await mine()) await request.delete(`/api/vehicles/${ev.id}/carpools/${trip.id}`, { headers })
  }
})

test('a charges CSV is previewed, imported, then undone from the import history', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const undoLeftovers = async () => {
    const batches = await (await request.get(`/api/vehicles/${ev.id}/import/batches`, { headers })).json()
    for (const b of batches ?? []) await request.delete(`/api/vehicles/${ev.id}/import/batches/${b.id}`, { headers })
  }
  // A run that stopped half way must not leave its batch behind for the next one
  await undoLeftovers()

  await useVehicle(page, ev.id, '/energy?tab=CHARGES')
  await page.getByRole('button', { name: 'Import CSV' }).first().click()
  const dialog = page.getByRole('dialog', { name: 'Import data (CSV)' })
  await dialog.locator('#csv-file-input').setInputFiles({
    name: 'charges.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from('date,kwh,cost,currency,location\n2020-03-04 18:30,12.5,2.5,EUR,E2E csv import\n'),
  })
  await dialog.getByRole('button', { name: 'Preview' }).click()
  await expect(dialog.getByText(/1 valid/i)).toBeVisible()

  await dialog.getByRole('button', { name: 'Import rows' }).click()
  await expect(dialog.getByText('Import completed successfully')).toBeVisible()
  await dialog.getByRole('button', { name: 'Close', exact: true }).last().click()
  await expect(page.getByRole('dialog')).toHaveCount(0)

  const charges = await (await request.get(`/api/vehicles/${ev.id}/charges?limit=500`, { headers })).json()
  expect(JSON.stringify(charges)).toContain('E2E csv import')

  // The import history of the dialog undoes the batch
  await page.getByRole('button', { name: 'Import CSV' }).first().click()
  const history = page.getByRole('dialog', { name: 'Import data (CSV)' })
  await history.getByText('Import history').click()
  await history.getByRole('button', { name: 'Undo this import' }).first().click()
  await page.getByRole('dialog').last().getByRole('button', { name: 'Undo this import' }).click()
  await expect.poll(async () => JSON.stringify(await (await request.get(`/api/vehicles/${ev.id}/charges?limit=500`, { headers })).json())).not.toContain('E2E csv import')
})

test.describe('a charge priced from a tariff plan', () => {
  // The server reads tariff hours in its reporting timezone (Europe/Paris by default): the browser uses the same one
  test.use({ timezoneId: 'Europe/Paris' })

  test('is priced hour by hour when it ends after midnight', async ({ page, request }) => {
    const headers = await authHeaders(request)
    const vehicleUrl = `/api/vehicles/${ev.id}`
    const vehicle = await (await request.get(vehicleUrl, { headers })).json()
    const plansUrl = '/api/tariffs/plans'
    const dropPlans = async () => {
      await request.put(vehicleUrl, { headers, data: { ...vehicle, tariff_plan_id: null } })
      const { plans } = await (await request.get(plansUrl, { headers })).json()
      for (const p of plans.filter((x: { name: string }) => x.name === 'E2E night')) await request.delete(`${plansUrl}/${p.id}`, { headers })
    }
    const dropCharges = async () => {
      const list = await (await request.get(`${vehicleUrl}/charges?limit=500`, { headers })).json()
      for (const c of (list.charges ?? list).filter((x: { notes?: string }) => x.notes === 'e2e tariff midnight')) {
        await request.delete(`${vehicleUrl}/charges/${c.id}`, { headers })
      }
    }
    // A run that stopped half way must not leave its plan or charge behind for the next one
    await dropCharges()
    await dropPlans()

    try {
      // 0.30 a kWh by day, 0.10 from midnight to 06:00
      const created = await request.post(plansUrl, {
        headers,
        data: {
          name: 'E2E night',
          plan_type: 'BANDS',
          currency: 'EUR',
          default_band: 'day',
          bands: [{ name: 'day', rate_cents: 0.3 }, { name: 'night', rate_cents: 0.1 }],
          rules: [{ start: '00:00', end: '06:00', band: 'night' }],
        },
      })
      expect(created.ok()).toBeTruthy()
      const plan = await created.json()
      expect((await request.put(vehicleUrl, { headers, data: { ...vehicle, tariff_plan_id: plan.id } })).ok()).toBeTruthy()

      // The form starts at 23:00 on 4 March 2020
      await page.clock.setFixedTime(new Date('2020-03-04T23:00:00+01:00'))
      await useVehicle(page, ev.id, '/energy?tab=CHARGES')
      await page.getByRole('button', { name: 'Add a charge' }).first().click()
      const dialog = page.getByRole('dialog', { name: 'New charge' })
      await expect(dialog.locator('input[aria-label="Datepicker input"]').first()).toHaveValue('04/03/2020 23:00')

      // It ends at 01:00 the next day
      await dialog.locator('input[aria-label="Datepicker input"]').nth(1).click()
      const picker = page.locator('.dp--menu')
      await picker.getByLabel('March 5th, 2020').click()
      await picker.getByLabel('Open time picker').click()
      await picker.getByLabel('Increment hours').click({ clickCount: 2 })
      await expect(picker.getByText('05/03/2020 01:00')).toBeVisible()
      await picker.getByRole('button', { name: 'Apply' }).click()

      // One hour on each side of midnight: 10 kWh at 0.30 and 10 kWh at 0.10
      await dialog.locator('#charge-form-kwh').fill('20')
      await expect(dialog.locator('#charge-form-cost')).toHaveValue('4')
      await expect(dialog.getByText('E2E night').first()).toBeVisible()

      await dialog.locator('#charge-form-notes').fill('e2e tariff midnight')
      await dialog.getByRole('button', { name: 'Save', exact: true }).click()
      await expect(dialog).toHaveCount(0)
      const list = await (await request.get(`${vehicleUrl}/charges?limit=500`, { headers })).json()
      const saved = (list.charges ?? list).find((x: { notes?: string }) => x.notes === 'e2e tariff midnight')
      expect(saved.cost).toBe(4)
    } finally {
      await dropCharges()
      await dropPlans()
    }
  })
})

test('a session from a shared charger is qualified onto a vehicle', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const dropSecondVehicle = async () => {
    const body = await (await request.get('/api/vehicles', { headers })).json()
    for (const v of (Array.isArray(body) ? body : body.vehicles).filter((x: Vehicle) => x.name === 'E2E second EV')) {
      await request.delete(`/api/vehicles/${v.id}`, { headers })
    }
  }
  const dropPending = async () => {
    const { pending_charges: pending } = await (await request.get('/api/pending-charges', { headers })).json()
    for (const p of (pending ?? []).filter((x: { energy_kwh: number }) => x.energy_kwh === 17.25)) {
      await request.delete(`/api/pending-charges/${p.id}`, { headers })
    }
  }
  // A run that stopped half way must not leave its vehicle or its session behind for the next one
  await dropSecondVehicle()
  await dropPending()

  // With two electric vehicles, a session that names none waits for the household to say whose it is
  const second = await request.post('/api/vehicles', { headers, data: { name: 'E2E second EV', powertrain: 'EV', currency: 'EUR', current_odometer: 0 } })
  expect(second.ok()).toBeTruthy()
  try {
    const sent = await request.post('/api/integrations/homeassistant/event', {
      headers,
      data: {
        event_id: `e2e-wallbox:${Date.now()}`,
        event_type: 'charging_session_end',
        data: { start_time: '2020-03-04T20:00:00Z', end_time: '2020-03-04T22:00:00Z', energy_added_kwh: 17.25 },
      },
    })
    expect((await sent.json()).status).toBe('pending_qualification')

    await useVehicle(page, ev.id, '/energy?tab=CHARGES')
    await page.getByRole('button', { name: 'Qualify' }).click()
    const dialog = page.getByRole('dialog', { name: 'Charges to Qualify' })
    await expect(dialog.getByText('17.25 kWh')).toBeVisible()
    await dialog.getByLabel('Select vehicle').selectOption({ label: ev.name })
    await dialog.getByRole('button', { name: 'Assign' }).click()
    await expect(dialog.getByText('17.25 kWh')).toHaveCount(0)

    // The session is now a charge of the chosen vehicle
    const list = await (await request.get(`/api/vehicles/${ev.id}/charges?limit=500`, { headers })).json()
    const charge = (list.charges ?? list).find((c: { kwh_added: number }) => c.kwh_added === 17.25)
    expect(charge).toBeTruthy()
    await request.delete(`/api/vehicles/${ev.id}/charges/${charge.id}`, { headers })
  } finally {
    await dropPending()
    await dropSecondVehicle()
  }
})

test('a carpool is started from the selected drives without leaving the drives page', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ev.id, '/drives')
  const search = page.getByPlaceholder(/Search for a city/)
  await search.fill('Office')
  const cards = page.getByRole('checkbox', { name: 'Select this drive' })
  await expect(cards.first()).toBeVisible()
  await cards.first().check()

  // The selection bar's button, not the quiet icon of a row
  const bulkCarpool = page.locator('button.btn', { hasText: 'Carpool' })
  await bulkCarpool.click()
  const dialog = page.getByRole('dialog', { name: 'New carpool' })
  await expect(dialog).toBeVisible()
  expect(new URL(page.url()).pathname).toBe('/drives')

  // Closing the editor leaves the page as it was: the search and the selection are still there
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  await expect(search).toHaveValue('Office')
  await expect(page.locator('input.select-box:checked')).toHaveCount(1)
  await expect(bulkCarpool).toBeVisible()
})

test('the TeslaMate connection is set, edited and removed from Data sources', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Connection reference', powertrain: 'EV', currency: 'EUR' } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  try {
    await page.setViewportSize({ width: 390, height: 844 })
    await useVehicle(page, vehicle.id, '/vehicles')
    const card = page.locator('div.card', { has: page.getByRole('heading', { name: vehicle.name }) })

    // The vehicle form is only its identity
    await card.getByRole('button', { name: 'More actions' }).click()
    await page.getByRole('menuitem', { name: 'Edit' }).click()
    await expect(page.locator('#vehicle-teslamate-api-url')).toHaveCount(0)
    await page.keyboard.press('Escape')

    await card.getByRole('button', { name: 'More actions' }).click()
    await page.getByRole('menuitem', { name: 'Data sources' }).click()
    const dialog = page.getByRole('dialog', { name: 'Data sources' })
    await dialog.getByRole('button', { name: 'Connect TeslaMate' }).click()
    await dialog.locator('#vehicle-teslamate-api-url').fill('http://127.0.0.1:9')
    await dialog.getByRole('button', { name: 'Save the connection' }).click()
    await expect(dialog.getByRole('button', { name: 'Edit connection' })).toBeVisible()
    const saved = await (await request.get(`/api/vehicles/${vehicle.id}`, { headers })).json()
    expect(saved.teslamate_api_url).toBe('http://127.0.0.1:9')
    expect(saved.name).toBe(vehicle.name)

    // A second save works on the new version, then the connection is removed after a confirmation
    await dialog.getByRole('button', { name: 'Edit connection' }).click()
    await expect(dialog.locator('#vehicle-teslamate-api-url')).toHaveValue('http://127.0.0.1:9')
    await dialog.getByRole('button', { name: 'Disconnect' }).click()
    await page.getByRole('dialog', { name: 'Disconnect TeslaMate?' }).getByRole('button', { name: 'Disconnect' }).click()
    await expect(dialog.getByRole('button', { name: 'Connect TeslaMate' })).toBeVisible()
    const after = await (await request.get(`/api/vehicles/${vehicle.id}`, { headers })).json()
    expect(after.teslamate_api_url ?? '').toBe('')
  } finally {
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
  }
})
