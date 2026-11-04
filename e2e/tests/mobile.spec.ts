import { expect, test, type Page } from '@playwright/test'
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

test('a date field inside a form is the native one on a phone, 16px so the page does not zoom on focus', async ({ page }) => {
  await useVehicle(page, ev.id, '/expenses')
  await page.getByRole('button', { name: /^Add/ }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  // No floating calendar: the browser's own picker opens from a plain date field
  await expect(dialog.locator('input.dp--input')).toHaveCount(0)
  const fields = dialog.locator('input[type=date], input[type=datetime-local]')
  expect(await fields.count()).toBeGreaterThan(0)
  const sizes = await fields.evaluateAll((els) => els.map((e) => getComputedStyle(e).fontSize))
  expect(sizes).toEqual(sizes.map(() => '16px'))

  // A value typed in the field stays as typed
  const first = fields.first()
  const withTime = (await first.getAttribute('type')) === 'datetime-local'
  await first.fill(withTime ? '2026-02-03T08:30' : '2026-02-03')
  await expect(first).toHaveValue(withTime ? '2026-02-03T08:30' : '2026-02-03')
})

test('a carpool is built in three steps on a phone: legs, passengers, then the recap', async ({ page, request }) => {
  const headers = await authHeaders(request)
  // The title is unique to the run, so a carpool left by an interrupted run does not get in the way
  const title = `E2E phone carpool ${Date.now()}`
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
  await dialog.locator('#carpool-title').fill(title)
  await expect(dialog.getByText('€30.00').first()).toBeVisible()
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(390)
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog).toHaveCount(0)
  await expect(page.getByText(title).first()).toBeVisible()

  // On a phone the row has no menu: a long press selects the carpool and the bar carries the delete action
  const box = (await page.locator('div.rounded-2xl', { hasText: title }).first().boundingBox())!
  const cdp = await page.context().newCDPSession(page)
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: box.x + 120, y: box.y + box.height / 2 }] })
  await expect(page.locator('[data-selection-bar]')).toBeVisible()
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
  await page.locator('[data-selection-bar]').getByRole('button', { name: 'Delete' }).click()
  // A single carpool is deleted without a question and the snackbar offers to take it back; wait for the server to be asked
  await expect(page.getByText(title)).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Undo' })).toBeVisible()
  const left = async () => (((await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()).trips ?? []) as { title: string }[]).filter((c) => c.title === title).length
  await expect.poll(left, { timeout: 12_000 }).toBe(0)
})

test('a cost deleted on a phone goes at once, comes back with Undo, and is only deleted once the time to undo is over', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const description = `Undo probe ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/maintenance`, {
    headers,
    data: { description, amount: '12.34', date: new Date().toISOString().slice(0, 10), odometer: 1, category: 'OTHER', type: 'OTHER' },
  })
  expect(created.status()).toBe(201)
  const rows = async () => ((await (await request.get(`/api/vehicles/${ev.id}/maintenance`, { headers })).json()) as { description: string }[]).filter((m) => m.description === description).length

  try {
    await useVehicle(page, ev.id, '/maintenance')
    const cdp = await page.context().newCDPSession(page)
    const deleteFromSelection = async () => {
      const box = (await page.locator('div.card', { hasText: description }).first().boundingBox())!
      await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: box.x + 120, y: box.y + box.height / 2 }] })
      await expect(page.locator('[data-selection-bar]')).toBeVisible()
      await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
      await page.locator('[data-selection-bar]').getByRole('button', { name: 'Delete' }).click()
    }

    await deleteFromSelection()
    await expect(page.getByText(description)).toHaveCount(0)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await rows()).toBe(1)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(page.getByText(description)).toBeVisible()
    expect(await rows()).toBe(1)

    await deleteFromSelection()
    await expect.poll(rows, { timeout: 12_000 }).toBe(0)
    await expect(page.getByText(description)).toHaveCount(0)
  } finally {
    const left = (await (await request.get(`/api/vehicles/${ev.id}/maintenance`, { headers })).json()) as { id: string; description: string }[]
    for (const m of left.filter((x) => x.description === description)) await request.delete(`/api/vehicles/${ev.id}/maintenance/${m.id}`, { headers })
  }
})

// A finger on the screen: Playwright has no swipe, so the touch is dispatched through the browser's input domain.
async function drag(page: Page, x: number, y: number, dx: number, dy: number, steps = 10) {
  const cdp = await page.context().newCDPSession(page)
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] })
  for (let i = 1; i <= steps; i++) {
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: [{ x: x + (dx * i) / steps, y: y + (dy * i) / steps }] })
  }
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
}
const swipeDown = (page: Page, x: number, y: number, distance: number, steps = 10) => drag(page, x, y, 0, distance, steps)

test('pulling the page down from the top reloads the list, and a short pull does not', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await page.waitForLoadState('networkidle')
  const drivesRequests: string[] = []
  page.on('request', (req) => {
    if (/\/api\/vehicles\/[^/]+\/drives\?/.test(req.url())) drivesRequests.push(req.url())
  })

  // A short pull follows the finger and comes back without asking for anything
  await swipeDown(page, 195, 200, 60)
  await page.waitForTimeout(700)
  expect(drivesRequests).toHaveLength(0)

  // A long pull from the top reloads the page's list and shows the badge while it does
  const reload = page.waitForRequest((req) => /\/api\/vehicles\/[^/]+\/drives\?/.test(req.url()))
  await swipeDown(page, 195, 200, 260)
  await reload
  await expect(page.getByRole('status').filter({ hasText: 'Refreshing' })).toHaveCount(1)
  await expect(page.getByRole('status').filter({ hasText: 'Refreshing' })).toHaveCount(0)
  expect(new URL(page.url()).pathname).toBe('/drives')
})

// Each pull starts where there is no field: a pull that starts on one is left to the field.
for (const [path, endpoint, startY] of [
  ['/tires', /\/api\/vehicles\/[^/]+\/tires$/, 200],
  ['/comparison', /\/api\/comparison-scenarios$/, 200],
  ['/account', /\/api\/auth\/sessions$/, 160],
] as const) {
  test(`pulling ${path} down reloads its data without swapping the page for a skeleton`, async ({ page }) => {
    await useVehicle(page, ev.id, path)
    await page.waitForLoadState('networkidle')
    const skeletons = page.locator('[aria-busy="true"]')
    await expect(skeletons).toHaveCount(0)
    const reload = page.waitForRequest((req) => endpoint.test(new URL(req.url()).pathname))
    await swipeDown(page, 195, startY, 260)
    await reload
    await expect(skeletons).toHaveCount(0)
    await expect(page.getByRole('status').filter({ hasText: 'Refreshing' })).toHaveCount(0)
    expect(new URL(page.url()).pathname).toBe(path)
  })
}

test('a sheet is closed by pulling it down by its handle, and a short pull lets it settle back', async ({ page }) => {
  await useVehicle(page, ev.id, '/expenses')
  await page.getByRole('button', { name: /^Add/ }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await page.waitForTimeout(400) // the slide-up animation
  const box = (await dialog.boundingBox())!

  await swipeDown(page, box.x + box.width / 2, box.y + 10, 20)
  await expect(dialog).toBeVisible()
  await expect.poll(async () => (await dialog.boundingBox())!.y).toBeCloseTo(box.y, 0)

  await swipeDown(page, box.x + box.width / 2, box.y + 10, 300)
  await expect(dialog).toBeHidden()
})

test('a sheet is also closed by pulling its content down, but not from a field', async ({ page }) => {
  await useVehicle(page, ev.id, '/expenses')
  await page.getByRole('button', { name: /^Add/ }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await page.waitForTimeout(400)
  const box = (await dialog.boundingBox())!

  // A field keeps its own gestures (selecting text, moving the caret)
  const field = (await dialog.locator('input:visible').first().boundingBox())!
  await swipeDown(page, field.x + field.width / 2, field.y + field.height / 2, 300)
  await page.waitForTimeout(500)
  await expect(dialog).toBeVisible()
  expect((await dialog.boundingBox())!.y).toBeCloseTo(box.y, 0)

  // The content, scrolled to the top, follows the finger like the handle does
  await swipeDown(page, box.x + 40, box.y + box.height - 120, 300)
  await expect(dialog).toBeHidden()
})

test('the Back gesture leaves the selection instead of the page', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const row = page.locator('div.relative.card', { has: page.getByRole('button', { name: /^Open the cost breakdown/ }) }).first()
  const box = (await row.boundingBox())!
  const cdp = await page.context().newCDPSession(page)
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: box.x + 120, y: box.y + box.height / 2 }] })
  const bar = page.locator('[data-selection-bar]')
  await expect(bar).toBeVisible()
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })

  await page.goBack()
  await expect(bar).toHaveCount(0)
  expect(new URL(page.url()).pathname).toBe('/drives')

  // The selection closed by its own button leaves no extra entry behind: Back then leaves the page
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: box.x + 120, y: box.y + box.height / 2 }] })
  await expect(bar).toBeVisible()
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
  await bar.getByRole('button').first().tap()
  await expect(bar).toHaveCount(0)
  await page.goBack()
  await page.waitForLoadState('networkidle')
  expect(new URL(page.url()).pathname).not.toBe('/drives')
})

test('on the home page every headline figure leads somewhere you can see', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  // The cost per distance opens the totals in a sheet rather than unfolding a block below the fold
  await page.getByRole('button', { name: 'Cost details' }).first().click()
  await expect(page.getByRole('dialog', { name: 'Cost details' })).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)

  const figures = page.getByRole('main')
  await figures.getByRole('link', { name: /^Cash paid/ }).click()
  await expect(page).toHaveURL(/\/expenses/)

  await page.goBack()
  await figures.getByRole('link', { name: /^Energy/ }).click()
  await expect(page).toHaveURL(/\/energy/)
})

test('a tire opens from its wheel tile, and the fitted set is one card instead of four', async ({ page }) => {
  await useVehicle(page, ev.id, '/tires')
  const tiles = page.getByRole('main').getByRole('button', { name: /^Open / })
  await expect(tiles.first()).toBeVisible()
  // The four wheels fit in the first screen of the page
  const last = (await tiles.last().boundingBox())!
  expect(last.y + last.height).toBeLessThan(844)
  await tiles.first().click()
  await expect(page.getByRole('dialog')).toBeVisible()
})

test('an odometer reading is added from a sheet on a phone, the history comes first', async ({ page, request }) => {
  await useVehicle(page, ev.id, '/odometer')
  const main = page.getByRole('main')
  await expect(main.locator('form')).toHaveCount(0)
  await main.getByRole('button', { name: 'Add the reading' }).click()
  const sheet = page.getByRole('dialog', { name: 'New odometer reading' })
  await expect(sheet).toBeVisible()
  // A reading is unique per vehicle, day and mileage: a fresh value keeps the test repeatable on a reused database
  const reading = 30000 + (Date.now() % 9000)
  await sheet.getByLabel(/^Mileage/).fill(String(reading))
  await sheet.getByRole('button', { name: 'Add the reading' }).click()
  await expect(sheet).toBeHidden()
  await expect(main.getByText(reading.toLocaleString('en-GB')).first()).toBeVisible()

  // Leave the shared demo vehicle as it was found
  const headers = await authHeaders(request)
  const list = await (await request.get(`/api/vehicles/${ev.id}/odometer-checkpoints`, { headers })).json()
  const created = list.find((c: { odometer: number }) => c.odometer === reading)
  expect((await request.delete(`/api/vehicles/${ev.id}/odometer-checkpoints/${created.id}`, { headers })).ok()).toBeTruthy()
})

test('saving a charge deep in the list keeps the page where it was', async ({ page }) => {
  await useVehicle(page, ev.id, '/energy?tab=CHARGES')
  const cards = page.getByRole('main').getByRole('button', { name: /^Open the charge of / })
  await expect(cards.first()).toBeVisible()
  const last = cards.last()
  await last.scrollIntoViewIfNeeded()
  const scroller = page.locator('.app-scroll')
  const before = await scroller.evaluate((e) => e.scrollTop)
  expect(before).toBeGreaterThan(400)

  await last.click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await dialog.getByRole('button', { name: 'Save', exact: true }).click()
  await expect(dialog).toHaveCount(0)

  // The list is refreshed behind the sheet without the skeleton: the page stays at the same place
  await page.waitForTimeout(800)
  const after = await scroller.evaluate((e) => e.scrollTop)
  expect(Math.abs(after - before)).toBeLessThan(60)
})

test('a sheet never scrolls sideways, whichever control it holds', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await page.getByRole('button', { name: /Filters/ }).first().click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  // The sheet sits in a backdrop: a hidden control wider than the screen would make the whole sheet slide left
  const sideways = () => dialog.evaluate((e) => e.parentElement!.scrollWidth - e.parentElement!.clientWidth)
  expect(await sideways()).toBe(0)
  await dialog.getByRole('button', { name: 'Month', exact: true }).click()
  await dialog.getByRole('button', { name: 'Range', exact: true }).click()
  expect(await sideways()).toBe(0)
  expect(await dialog.evaluate((e) => e.parentElement!.scrollLeft)).toBe(0)
})

test('a cost dragged to the left is deleted with an Undo, and a short drag leaves it alone', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const description = `Swipe probe ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/maintenance`, {
    headers,
    data: { description, amount: '9.87', date: new Date().toISOString().slice(0, 10), odometer: 1, category: 'OTHER', type: 'OTHER' },
  })
  expect(created.status()).toBe(201)
  const rows = async () => ((await (await request.get(`/api/vehicles/${ev.id}/maintenance`, { headers })).json()) as { description: string }[]).filter((m) => m.description === description).length

  try {
    await useVehicle(page, ev.id, '/maintenance')
    const row = page.locator('div.card', { hasText: description }).first()
    await row.scrollIntoViewIfNeeded()
    const box = (await row.boundingBox())!
    const y = box.y + box.height / 2

    // A short drag slides back and neither deletes nor opens the row
    await drag(page, 340, y, -50, 0)
    await expect(page.getByText(description)).toBeVisible()
    await expect(page.getByRole('dialog')).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Undo' })).toHaveCount(0)

    // A vertical move is a scroll: the row stays
    await drag(page, 340, y, -10, 60)
    await expect(page.getByText(description)).toBeVisible()

    await drag(page, 340, y, -200, 4)
    await expect(page.getByText(description)).toHaveCount(0)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await rows()).toBe(1)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(page.getByText(description)).toBeVisible()

    await drag(page, 340, (await page.locator('div.card', { hasText: description }).first().boundingBox())!.y + box.height / 2, -200, 4)
    await expect.poll(rows, { timeout: 12_000 }).toBe(0)
  } finally {
    const left = (await (await request.get(`/api/vehicles/${ev.id}/maintenance`, { headers })).json()) as { id: string; description: string }[]
    for (const m of left.filter((x) => x.description === description)) await request.delete(`/api/vehicles/${ev.id}/maintenance/${m.id}`, { headers })
  }
})

test('an odometer reading dragged to the left is deleted with an Undo', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const notes = `Swipe reading ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/odometer-checkpoints`, {
    headers,
    data: { date: new Date().toISOString().slice(0, 10), odometer: 987654, notes },
  })
  expect(created.ok()).toBeTruthy()
  const count = async () => ((await (await request.get(`/api/vehicles/${ev.id}/odometer-checkpoints`, { headers })).json()) as { notes?: string }[]).filter((c) => c.notes === notes).length

  try {
    await useVehicle(page, ev.id, '/odometer')
    const rowOf = () => page.locator('div.rounded-xl', { hasText: notes }).last()
    await rowOf().scrollIntoViewIfNeeded()
    const box = (await rowOf().boundingBox())!
    const y = box.y + box.height / 2

    await drag(page, 340, y, -200, 4)
    await expect(page.getByText(notes)).toHaveCount(0)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await count()).toBe(1)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(page.getByText(notes)).toBeVisible()
    expect(await count()).toBe(1)

    await drag(page, 340, (await rowOf().boundingBox())!.y + box.height / 2, -200, 4)
    await expect.poll(count, { timeout: 12_000 }).toBe(0)
  } finally {
    const left = (await (await request.get(`/api/vehicles/${ev.id}/odometer-checkpoints`, { headers })).json()) as { id: string; notes?: string }[]
    for (const c of left.filter((x) => x.notes === notes)) await request.delete(`/api/vehicles/${ev.id}/odometer-checkpoints/${c.id}`, { headers })
  }
})

test('a drive entered by hand dragged to the left is deleted with an Undo', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const address = `Swipe drive ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/drives`, {
    headers,
    data: { start_time: new Date().toISOString(), distance_km: 12.3, start_address: address, end_address: 'Swipe end', tags: [] },
  })
  expect(created.ok()).toBeTruthy()
  const listed = async () => {
    const body = (await (await request.get(`/api/vehicles/${ev.id}/drives?limit=50`, { headers })).json()) as { drives?: { id: string; start_address?: string }[] } | { id: string; start_address?: string }[]
    return (Array.isArray(body) ? body : (body.drives ?? [])).filter((d) => d.start_address === address)
  }
  const count = async () => (await listed()).length

  try {
    expect(await count()).toBe(1)
    await useVehicle(page, ev.id, '/drives')
    const rowOf = () => page.locator('div.card', { hasText: address }).last()
    await rowOf().scrollIntoViewIfNeeded()
    const box = (await rowOf().boundingBox())!
    const y = box.y + box.height / 2

    await drag(page, 340, y, -200, 4)
    await expect(page.getByText(address)).toHaveCount(0)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await count()).toBe(1)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(page.getByText(address)).toBeVisible()

    await drag(page, 340, (await rowOf().boundingBox())!.y + box.height / 2, -200, 4)
    await expect.poll(count, { timeout: 12_000 }).toBe(0)
  } finally {
    for (const d of await listed()) await request.delete(`/api/vehicles/${ev.id}/drives/${d.id}`, { headers })
  }
})

test('a fill-up dragged to the left is deleted with an Undo', async ({ page, request }) => {
  const ice = (await vehicles(request)).find((v) => v.powertrain === 'ICE')!
  const headers = await authHeaders(request)
  const odometer = 900000 + (Date.now() % 90000)
  const created = await request.post(`/api/vehicles/${ice.id}/fuel-logs`, {
    headers,
    data: { date: new Date().toISOString(), odometer, liters: 40, amount: 71.23, fuel_type: 'SP95_E10', is_full_tank: true },
  })
  expect(created.ok()).toBeTruthy()
  const logs = async () => (((await (await request.get(`/api/vehicles/${ice.id}/fuel-logs`, { headers })).json()) as { logs: { id: string; odometer: number }[] }).logs ?? []).filter((l) => Math.round(l.odometer) === odometer)
  const count = async () => (await logs()).length

  try {
    await useVehicle(page, ice.id, '/energy?tab=FUEL')
    const rowOf = () => page.locator('div.card', { hasText: odometer.toLocaleString('en-GB') }).last()
    await rowOf().scrollIntoViewIfNeeded()
    const box = (await rowOf().boundingBox())!
    const y = box.y + box.height / 2

    await drag(page, 340, y, -200, 4)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await count()).toBe(1)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(rowOf()).toBeVisible()

    await drag(page, 340, (await rowOf().boundingBox())!.y + box.height / 2, -200, 4)
    await expect.poll(count, { timeout: 12_000 }).toBe(0)
  } finally {
    for (const l of await logs()) await request.delete(`/api/vehicles/${ice.id}/fuel-logs/${l.id}`, { headers })
  }
})

test('the drives come under a header per day and the next ones load while scrolling, without pages', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const headers = page.locator('main section > h2')
  await expect(headers.first()).toBeVisible()
  // The days are named, not numbered pages
  await expect(page.getByText(/^Page \d+ of \d+$/)).toHaveCount(0)
  const firstDays = await headers.count()
  expect(firstDays).toBeGreaterThan(1)

  const next = page.waitForRequest((req) => /\/api\/vehicles\/[^/]+\/drives\?/.test(req.url()) && new URL(req.url()).searchParams.get('page') === '2')
  await page.locator('.app-scroll').evaluate((el) => el.scrollTo({ top: el.scrollHeight }))
  await next
  await expect.poll(() => headers.count()).toBeGreaterThan(firstDays)

  // A day is named once even when it runs over two pages
  const names = await headers.allTextContents()
  expect(new Set(names).size).toBe(names.length)
})

test('a carpool dragged to the left is deleted with an Undo', async ({ page, request }) => {
  const headers = await authHeaders(request)
  const title = `Swipe carpool ${Date.now()}`
  const created = await request.post(`/api/vehicles/${ev.id}/carpools`, {
    headers,
    data: {
      title,
      date: new Date().toISOString(),
      costs_frozen: true,
      legs: [{ start_address: 'A', end_address: 'B', distance_km: 10, electricity_cost: 2, tolls_cost: 0, overrides: {} }],
      passengers: [{ name: 'Probe', seats: 1, amount_paid: 1 }],
    },
  })
  expect(created.ok(), await created.text()).toBeTruthy()
  const listed = async () => (((await (await request.get(`/api/vehicles/${ev.id}/carpools`, { headers })).json()).trips ?? []) as { id: string; title: string }[]).filter((c) => c.title === title)
  const count = async () => (await listed()).length

  try {
    await useVehicle(page, ev.id, '/carpools')
    const rowOf = () => page.locator('div.rounded-2xl', { hasText: title }).last()
    await rowOf().scrollIntoViewIfNeeded()
    const box = (await rowOf().boundingBox())!
    const y = box.y + box.height / 2

    await drag(page, 340, y, -200, 4)
    await expect(page.getByText(title)).toHaveCount(0)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await count()).toBe(1)
    await page.getByRole('button', { name: 'Undo' }).click()
    await expect(page.getByText(title)).toBeVisible()

    await drag(page, 340, (await rowOf().boundingBox())!.y + box.height / 2, -200, 4)
    await expect.poll(count, { timeout: 12_000 }).toBe(0)
  } finally {
    for (const c of await listed()) await request.delete(`/api/vehicles/${ev.id}/carpools/${c.id}`, { headers })
  }
})
