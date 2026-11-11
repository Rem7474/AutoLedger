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
    await page.getByRole('button', { name: `Open ${name}` }).click()
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
    await page.getByRole('button', { name: `Open ${name}` }).click()
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
    await page.getByRole('button', { name: `Open ${name}` }).click()
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
  await useVehicle(page, ev.id, '/consumption')
  await page.getByRole('button', { name: 'Quick add' }).first().click()
  await page.fill('#qc-kwh', '37.5')
  await page.fill('#qc-cost', '9.80')
  await page.getByRole('button', { name: 'Save the charge' }).click()
  await expect(toast(page)).toContainText('37.5')
})

test('quick-add records a fill-up on a combustion vehicle', async ({ page }) => {
  await useVehicle(page, ice.id, '/consumption')
  await page.getByRole('button', { name: 'Quick add' }).first().click()
  await page.fill('#qf-liters', '40')
  await page.fill('#qf-amount', '72')
  await page.getByRole('button', { name: /^save/i }).click()
  await expect(toast(page)).toContainText('72')
})

test('consumption tabs follow the powertrain and the legacy fuel URL redirects', async ({ page }) => {
  await useVehicle(page, ice.id, '/manual?tab=FUEL')
  await expect(page).toHaveURL(/\/consumption\?tab=FUEL/)
  await expect(page.getByRole('button', { name: 'New fill-up' })).toBeVisible()
  await expect(page.getByRole('tab', { name: 'Charges' })).toHaveCount(0)

  await useVehicle(page, ev.id, '/consumption')
  await expect(page.getByRole('tab', { name: 'Charges' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'New fill-up' })).toHaveCount(0)
})

for (const width of [320, 360, 375, 390, 414]) {
  test(`no horizontal scroll at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 800 })
    for (const route of ['/', '/consumption', '/drives?tab=CARPOOLS', '/expenses', '/maintenance', '/account']) {
      await useVehicle(page, ev.id, route)
      await expectNoHorizontalScroll(page)
    }
  })
}

const modal = (page: import('@playwright/test').Page) => page.getByRole('dialog', { name: 'Add tires' })

test.describe('a modal with edits does not close on a backdrop click', () => {
  test.beforeEach(async ({ page }) => {
    // Below 1024 px the form is a sheet over a backdrop; from there up it is a panel with none
    await page.setViewportSize({ width: 800, height: 900 })
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

test('the home is the present: the month so far, what is coming up, a line to the vehicle, and no period bar', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 812 })
  await useVehicle(page, ev.id, '/')
  const main = page.getByRole('main')
  await expect(main.getByTestId('today-hero')).toBeVisible()
  await expect(main.getByTestId('today-per-km')).toBeVisible()
  await expect(main.getByRole('heading', { name: 'Coming up' })).toBeVisible()
  await expect(main.getByTestId('upcoming')).toBeVisible()
  await expect(main.getByRole('link', { name: /Vehicle sheet/ })).toHaveAttribute('href', '/vehicle')
  await expect(page.getByTestId('period-bar')).toHaveCount(0)
})

test('the home hero leads to the journal of costs on a wide screen too', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ev.id, '/')
  await page.getByTestId('today-hero').getByRole('link').first().click()
  await expect(page).toHaveURL(/\/costs/)
})

test('the old fleet address lands on the home page with every vehicle, one line per currency', async ({ page, request }) => {
  const headers = await authHeaders(request)
  // A vehicle in another currency with one cost: the household view gives it its own line
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Pound reference', powertrain: 'EV', currency: 'GBP' } })
  expect(created.ok()).toBeTruthy()
  const pound = await created.json()
  try {
    const now = new Date().toISOString()
    const charge = await request.post(`/api/vehicles/${pound.id}/charges`, { headers, data: { date: now, kwh_added: 10, cost: 2, currency: 'GBP' } })
    expect(charge.ok()).toBeTruthy()
    await useVehicle(page, ev.id, '/fleet')
    await expect(page).toHaveURL(/\/\?scope=all$/)
    await expect(page.getByRole('heading', { name: 'All vehicles' })).toBeVisible()
    const totals = page.getByTestId('fleet-currency-total')
    // Nothing is added across currencies: the euros and the pounds are two lines
    await expect(totals).toHaveCount(2)
    await expect(totals.first()).toContainText('€')
    await expect(totals.nth(1)).toContainText('£')
    await expect(page.getByTestId('fleet-overview').getByRole('heading', { name: 'Vehicles side by side' })).toBeVisible()
    await expect(page.getByRole('navigation').getByRole('link', { name: 'Household Fleet' })).toHaveCount(0)
  } finally {
    await request.delete(`/api/vehicles/${pound.id}`, { headers })
  }
})

test('the vehicle selector opens the household view and a vehicle leaves it', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  const select = page.getByLabel('Active vehicle')
  await select.selectOption({ label: 'All vehicles' })
  await expect(page).toHaveURL(/\/\?scope=all$/)
  await expect(page.getByTestId('fleet-overview')).toBeVisible()
  await page.reload()
  await expect(page.getByTestId('fleet-overview')).toBeVisible()
  await expect(select).toHaveValue('all')
  await select.selectOption(ev.id)
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByTestId('today-hero')).toBeVisible()
  // Another page shows the vehicle, never the household entry
  await useVehicle(page, ev.id, '/drives')
  await expect(select).toHaveValue(ev.id)
})

test('the odometer page lists the mileage typed on fill-ups, read-only', async ({ page }) => {
  await useVehicle(page, ice.id, '/odometer')
  await expect(page.getByText('Fill-up', { exact: true }).first()).toBeVisible()
  const row = page.locator('div.rounded-2xl', { has: page.getByText('Fill-up', { exact: true }) }).first()
  await expect(row.getByRole('button')).toHaveCount(0)
  await expect(row.getByRole('link', { name: 'See the fill-ups' })).toHaveAttribute('href', /\/consumption\?tab=FUEL/)
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

test('a comparison is deleted from its detail with Undo, without a row menu', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const name = `Detail delete comparison ${Date.now()}`
  const saved = await request.post('/api/comparison-scenarios/', { headers, data: {
    name, mode: 'PROJECTION', currency: 'EUR', annual_km: 12000, years: 5,
    ice: { fuel_type: 'SP95_E10', l_per_100km: 6.5, fuel_price: 1.7 },
    tracked: { kwh_per_100km: 16, eur_per_kwh: 0.2 },
  } })
  expect(saved.ok()).toBeTruthy()
  const { id } = await saved.json()
  try {
    await page.setViewportSize({ width: 1280, height: 900 })
    await useVehicle(page, ice.id, '/comparison')
    const open = page.getByRole('button', { name: `Open ${name}` })
    await expect(open).toHaveCount(1)
    await expect(page.locator('[aria-haspopup="menu"]')).toHaveCount(0)
    await open.click()
    await expect(page.getByRole('button', { name: 'Edit', exact: true })).toBeVisible()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(open).toHaveCount(0)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(open).toHaveCount(1)
    await open.click()
    const deleted = page.waitForResponse((r) => r.request().method() === 'DELETE' && r.url().includes(`/comparison-scenarios/${id}`))
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    expect((await deleted).ok()).toBeTruthy()
    await expect(open).toHaveCount(0)
  } finally {
    await request.delete(`/api/comparison-scenarios/${id}`, { headers })
  }
})

test('a fill-up is deleted from its detail with Undo, without a row menu', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const notes = `Detail delete fill-up ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ice.id}/fuel-logs`, { headers, data: { date: '2026-01-01T12:00:00Z', amount: 17, liters: 10, fuel_type: 'SP95_E10', notes } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await page.setViewportSize({ width: 1280, height: 900 })
    await useVehicle(page, ice.id, '/consumption?tab=FUEL&period=all')
    const open = page.getByRole('button', { name: /^Edit the fill-up of 01 Jan 2026/ })
    await expect(open).toHaveCount(1)
    await expect(page.getByRole('button', { name: 'More actions' })).toHaveCount(0)
    await open.click()
    await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(open).toHaveCount(0)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(open).toHaveCount(1)
    await open.click()
    const deleted = page.waitForResponse((r) => r.request().method() === 'DELETE' && r.url().includes(`/fuel-logs/${id}`))
    await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click()
    expect((await deleted).ok()).toBeTruthy()
    await expect(open).toHaveCount(0)
  } finally {
    await request.delete(`/api/vehicles/${ice.id}/fuel-logs/${id}`, { headers })
  }
})

test('a fill-up opens from the cost journal in its sheet, is edited and shows its new amount', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const notes = `Journal fill-up ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ice.id}/fuel-logs`, { headers, data: { date: new Date().toISOString(), amount: 17, liters: 10, fuel_type: 'SP95_E10', notes } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await page.setViewportSize({ width: 390, height: 844 })
    await useVehicle(page, ice.id, '/costs')
    const row = page.getByRole('button', { name: notes })
    await row.click()
    // The sheet opens on the journal: no navigation to the consumption page
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await expect(page).toHaveURL(/\/costs/)
    await expect(dialog.getByRole('button', { name: 'Delete', exact: true })).toBeVisible()
    await dialog.locator('#fuel-amount').fill('18.5')
    const saved = page.waitForResponse((r) => r.request().method() === 'PUT' && r.url().includes(`/fuel-logs/${id}`))
    await dialog.getByRole('button', { name: 'Save', exact: true }).click()
    expect((await saved).ok()).toBeTruthy()
    await expect(dialog).toHaveCount(0)
    await expect(page.getByTestId('cost-entry').filter({ hasText: notes })).toContainText(/18[.,]50/)
  } finally {
    await request.delete(`/api/vehicles/${ice.id}/fuel-logs/${id}`, { headers })
  }
})

test('an odometer reading is deleted from its form with Undo, without a row menu', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const notes = `Detail delete reading ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ice.id}/odometer-checkpoints`, { headers, data: { date: new Date().toISOString().slice(0, 10), odometer: 876544, notes } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await page.setViewportSize({ width: 1280, height: 900 })
    await useVehicle(page, ice.id, '/odometer')
    const open = page.getByRole('button', { name: /^Edit the .*876,544/ })
    await expect(open).toHaveCount(1)
    await expect(page.getByRole('button', { name: 'More actions' })).toHaveCount(0)
    await open.click()
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(open).toHaveCount(0)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(open).toHaveCount(1)
    await open.click()
    const deleted = page.waitForResponse((r) => r.request().method() === 'DELETE' && r.url().includes(`/odometer-checkpoints/${id}`))
    await page.getByRole('button', { name: 'Delete', exact: true }).click()
    expect((await deleted).ok()).toBeTruthy()
    await expect(open).toHaveCount(0)
  } finally {
    await request.delete(`/api/vehicles/${ice.id}/odometer-checkpoints/${id}`, { headers })
  }
})

test('a manual drive is deleted from its detail with Undo, without a row menu', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const end = `Detail delete drive ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/drives`, { headers, data: { start_time: new Date().toISOString(), distance_km: 12.3, start_address: 'Probe start', end_address: end, tags: [] } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await page.setViewportSize({ width: 1280, height: 900 })
    await useVehicle(page, ev.id, '/drives')
    const row = page.getByTestId('drive-card').filter({ hasText: end })
    await expect(row).toHaveCount(1)
    await expect(row.getByRole('button', { name: 'More actions' })).toHaveCount(0)
    await row.getByRole('button', { name: /^Open the cost breakdown/ }).click()
    const dialog = page.getByRole('dialog')
    await expect(dialog.getByRole('button', { name: 'Edit', exact: true })).toBeVisible()
    await dialog.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(row).toHaveCount(0)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(row).toHaveCount(1)
    await row.getByRole('button', { name: /^Open the cost breakdown/ }).click()
    const deleted = page.waitForResponse((r) => r.request().method() === 'DELETE' && r.url().includes(`/drives/${id}`))
    await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click()
    expect((await deleted).ok()).toBeTruthy()
    await expect(row).toHaveCount(0)
  } finally {
    await request.delete(`/api/vehicles/${ev.id}/drives/${id}`, { headers })
  }
})

test('a manual charge is deleted from its detail with Undo, without a row menu', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const address = `Detail delete charge ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/charges`, { headers, data: { date: new Date().toISOString(), kwh_added: 12.34, cost: 3.21, currency: 'EUR', address } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  try {
    await page.setViewportSize({ width: 1280, height: 900 })
    await useVehicle(page, ev.id, '/costs?theme=energy')
    const open = page.getByRole('button', { name: address, exact: true })
    await expect(open).toHaveCount(1)
    await expect(page.getByRole('button', { name: 'More actions' })).toHaveCount(0)
    await open.click()
    const dialog = page.getByRole('dialog')
    await expect(dialog.getByText('Entered manually.')).toBeVisible()
    await dialog.getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(open).toHaveCount(0)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(open).toHaveCount(1)
    await open.click()
    const deleted = page.waitForResponse((r) => r.request().method() === 'DELETE' && r.url().includes(`/charges/${id}`))
    await page.getByRole('dialog').getByRole('button', { name: 'Delete', exact: true }).click()
    expect((await deleted).ok()).toBeTruthy()
    await expect(open).toHaveCount(0)
  } finally {
    await request.delete(`/api/vehicles/${ev.id}/charges/${id}`, { headers })
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
  await useVehicle(page, ice.id, '/consumption?tab=FUEL')
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
  await useVehicle(page, ev.id, '/consumption?tab=CHARGES')
  await page.getByRole('button', { name: 'Add a charge' }).click()
  const charge = page.getByRole('dialog').first()
  await expect(charge).toBeVisible()
  await charge.getByRole('button', { name: 'Close', exact: true }).first().click()
  await expect(charge).toHaveCount(0)

  // The quick entry sends a maintenance entry to the costs page, where its form opens
  await useVehicle(page, ev.id, '/costs?theme=maintenance&add=maintenance')
  const maintenance = page.getByRole('dialog', { name: /Add maintenance \/ fixed expense/ })
  await expect(maintenance).toBeVisible()
  await expect(page).not.toHaveURL(/add=/)
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

  await useVehicle(page, ev.id, '/drives?tab=CARPOOLS')
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

  // The row has no menu: it opens the carpool detail, which carries Edit in its header and a quiet Delete at the bottom
  await expect(page.getByRole('button', { name: 'More actions' })).toHaveCount(0)
  await page.getByRole('button', { name: 'Open E2E carpool' }).first().click()
  const detail = page.getByRole('dialog')
  await expect(detail.getByRole('button', { name: 'Edit', exact: true })).toBeVisible()
  await detail.getByRole('button', { name: 'Delete', exact: true }).click()
  // No confirmation: the carpool leaves the list and a snackbar offers to take it back until the server is asked
  await expect(page.getByText('E2E carpool')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Undo' })).toBeVisible()
  const listed = async () => ((await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()).trips ?? []).filter((c: { title: string }) => c.title === 'E2E carpool').length
  expect(await listed()).toBe(1)
  await expect.poll(listed, { timeout: 12_000 }).toBe(0)
})

test('a trip has no row menu: its detail carries Edit in the header and a quiet Delete at the bottom', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const name = `E2E trip ${Date.now()}`
  const drives = (await (await request.get(`/api/vehicles/${ev.id}/drives?limit=2`, { headers })).json()).drives as { id: string }[]
  const created = await request.post(`/api/vehicles/${ev.id}/trip-groups`, { headers, data: { name, notes: 'E2E trip notes', drive_ids: drives.map((d) => d.id) } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  const listed = async () => ((await (await request.get(`/api/vehicles/${ev.id}/trip-groups`, { headers })).json()) as { id: string }[]).filter((g) => g.id === id).length
  try {
    await useVehicle(page, ev.id, '/drives?tab=TRIPS&period=all')
    await expect(page.getByRole('button', { name: 'More actions' })).toHaveCount(0)
    await page.getByRole('button', { name: `Open ${name}` }).click({ position: { x: 20, y: 6 } })
    const detail = page.getByRole('dialog')
    await expect(detail.getByRole('button', { name: 'Edit', exact: true })).toBeVisible()
    await expect(detail.getByText('E2E trip notes')).toBeVisible()
    await detail.getByRole('button', { name: 'Delete', exact: true }).click()
    // Deleting a trip asks first (its costs are kept or deleted with it)
    await page.getByRole('dialog', { name: 'Delete the trip' }).getByRole('button', { name: 'Delete the trip', exact: true }).click()
    await expect.poll(listed, { timeout: 10_000 }).toBe(0)
  } finally {
    if (await listed()) await request.delete(`/api/vehicles/${ev.id}/trip-groups/${id}`, { headers })
  }
})

test('the legs of a trip are compact cards that open the leg; a leg is taken out of the trip from its detail', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const name = `E2E legs ${Date.now()}`
  const drives = (await (await request.get(`/api/vehicles/${ev.id}/drives?limit=3`, { headers })).json()).drives as { id: string }[]
  const created = await request.post(`/api/vehicles/${ev.id}/trip-groups`, { headers, data: { name, drive_ids: drives.map((d) => d.id) } })
  expect(created.ok()).toBeTruthy()
  const { id } = await created.json()
  const legCount = async () => (((await (await request.get(`/api/vehicles/${ev.id}/trip-groups`, { headers })).json()) as { id: string; drive_ids: string[] }[]).find((g) => g.id === id)?.drive_ids ?? []).length
  try {
    await useVehicle(page, ev.id, '/drives?tab=TRIPS&period=all')
    await page.locator('div.relative.flex', { has: page.getByRole('button', { name: `Open ${name}` }) }).getByRole('button', { name: 'Legs', exact: true }).click()
    const legs = page.locator('[data-testid="trip-legs"] [data-testid="drive-leg"]')
    await expect(legs).toHaveCount(3)
    await expect(legs.first()).toHaveClass(/card/)
    // no hidden control on a leg: the row is one open button, nothing else
    await expect(page.locator('[data-testid="trip-legs"] button')).toHaveCount(3)
    expect(await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)).toBe(0)
    await legs.first().getByRole('button').click()
    const detail = page.getByRole('dialog')
    await detail.getByRole('button', { name: 'Remove from this trip', exact: true }).click()
    await expect.poll(legCount, { timeout: 10_000 }).toBe(2)
  } finally {
    await request.delete(`/api/vehicles/${ev.id}/trip-groups/${id}`, { headers })
  }
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

    await useVehicle(page, ev.id, '/drives?tab=CARPOOLS&period=all')
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

test('the carpools open on the current month and follow the period bar', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const title = `E2E dated ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/carpools`, {
    headers,
    data: { title, date: '2019-05-10', legs: [{ distance_km: 10, electricity_cost: 1 }], passengers: [{ passenger_name: 'Old', seats: 1, amount_paid: 0.5 }] },
  })
  expect(created.ok()).toBeTruthy()
  const id = (await created.json()).id
  try {
    await useVehicle(page, ev.id, '/drives?tab=CARPOOLS')
    const bar = page.getByTestId('period-bar')
    await expect(bar).toBeVisible()
    await expect(page).toHaveURL(/period=1M/)
    // A carpool of 2019 is outside the month: the list says so and offers the other periods
    await expect(page.getByText('No carpool in this period')).toBeVisible()
    await expect(page.getByText(title)).toHaveCount(0)

    await bar.getByRole('button', { name: /Choose the period/ }).click()
    const sheet = page.getByRole('dialog')
    await sheet.getByRole('button', { name: 'All time' }).click()
    await expect(page.getByText(title)).toBeVisible()
    await expect(page).toHaveURL(/period=all/)
  } finally {
    await request.delete(`/api/vehicles/${ev.id}/carpools/${id}`, { headers })
  }
})

test('the fuel tab follows the period bar: rows and figures change with the period', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Fuel period reference', powertrain: 'ICE', currency: 'EUR' } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  const daysAgo = (n: number, minutes = 0) => new Date(Date.now() - n * 86_400_000 - minutes * 60_000).toISOString()
  try {
    // Two full tanks two years ago, two a moment ago: the period decides which are listed and counted
    const fills = [
      { date: daysAgo(740), odometer: 10000, liters: 40, amount: 60 },
      { date: daysAgo(700), odometer: 10600, liters: 36, amount: 54 },
      { date: daysAgo(0, 2), odometer: 20000, liters: 45, amount: 90 },
      { date: daysAgo(0, 1), odometer: 20500, liters: 30, amount: 60 },
    ]
    for (const f of fills) {
      const res = await request.post(`/api/vehicles/${vehicle.id}/fuel-logs`, { headers, data: { ...f, fuel_type: 'SP95_E10', is_full_tank: true } })
      expect(res.ok()).toBeTruthy()
    }
    // No period in the address: the fill-ups open on the current month, whatever the costs pages hold
    await useVehicle(page, vehicle.id, '/consumption?tab=FUEL')
    await expect(page).toHaveURL(/period=1M/)
    const cards = page.locator('.grid .card')
    await expect(page.getByTestId('period-bar')).toBeVisible()
    await expect(page.getByTestId('fuel-period-label')).toContainText(new Date().toLocaleDateString('en-US', { month: 'long', year: 'numeric' }))
    // Only the two recent fill-ups: 150 spent. The first still closes its interval with the full tank of two years ago
    // (45 L over 9400 km) and the second one 30 L over 500 km: 75 L over 9900 km, weighted by distance
    await expect(cards.nth(0)).toContainText('2')
    await expect(cards.nth(1)).toContainText('150')
    await expect(cards.nth(3)).toContainText('0.76')

    await page.getByTestId('period-bar').getByRole('button', { name: /Choose the period/ }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'All time' }).click()
    await expect(page).toHaveURL(/period=all/)
    await expect(cards.nth(0)).toContainText('4')
    await expect(cards.nth(1)).toContainText('264')
    await expectNoHorizontalScroll(page)

    // The choice is the fill-ups' own: the costs page keeps its period
    await page.goto('/costs')
    await expect(page).toHaveURL(/period=12M/)

    // A period with no fill-up says so instead of showing a blank list
    await page.goto(`/consumption?tab=FUEL&from=2019-01-01&to=2019-01-31`)
    await expect(page.getByTestId('fuel-period-empty')).toContainText('No fill-up in this period')
    await expect(cards).toHaveCount(0)
  } finally {
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
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

  await useVehicle(page, ev.id, '/consumption?tab=CHARGES')
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
      await useVehicle(page, ev.id, '/consumption?tab=CHARGES')
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

    await useVehicle(page, ev.id, '/consumption?tab=CHARGES')
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

test('the account button of the top bar is for a phone: the sidebar carries the account on a desktop', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  await expect(page.getByRole('link', { name: 'Account' })).toBeVisible()
  await expect(page.getByTestId('account-button')).toBeHidden()
})

test('on a wide screen a cost opens as a panel beside the journal, which stays visible and usable', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 800 })
  await useVehicle(page, ev.id, '/costs')
  const row = page.getByRole('button', { name: /^A\d+ / }).first()
  await row.click()
  const pane = page.getByRole('dialog')
  await expect(pane).toBeVisible()
  const box = (await pane.boundingBox())!
  expect(box.x + box.width).toBeGreaterThan(1270)
  expect(box.width).toBeLessThan(500)
  expect(box.height).toBeGreaterThan(700)
  // The journal is not covered: the row is left of the panel and a click on another row reaches it
  const rowBox = (await row.boundingBox())!
  expect(rowBox.x + rowBox.width).toBeLessThanOrEqual(box.x + 1)
  await expectNoHorizontalScroll(page)
  await page.keyboard.press('Escape')
  await expect(pane).toHaveCount(0)
  await expect(row).toBeVisible()
})
