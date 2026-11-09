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
  await useVehicle(page, ev.id, '/costs?theme=maintenance&add=maintenance')
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
  await useVehicle(page, ev.id, '/costs?theme=maintenance&add=maintenance')
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
  await useVehicle(page, ev.id, '/costs?theme=maintenance&add=maintenance')
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
  await expect(page).toHaveURL(/\/costs(\?period=\w+)?$/)

  await page.goBack()
  await figures.getByRole('link', { name: /^Energy/ }).click()
  await expect(page).toHaveURL(/\/costs\?theme=energy/)
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

test('battery, resale value and contract are on the vehicle sheet, and the home keeps one line that points there', async ({ page, request }) => {
  await useVehicle(page, ev.id, '/vehicle')
  const main = page.getByRole('main')
  await expect(main.getByRole('button', { name: /Battery health/ })).toBeVisible()
  await expect(main.getByRole('button', { name: /Resale value/ })).toBeVisible()
  await main.getByRole('button', { name: /Battery health/ }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.goBack()
  await useVehicle(page, ev.id, '/')
  await expect(page.getByRole('main').getByRole('link', { name: /Vehicle sheet/ })).toHaveAttribute('href', '/vehicle')
  await expect(page.getByRole('main').getByRole('button', { name: /Resale value/ })).toHaveCount(0)
  // A combustion vehicle has no battery line
  const ice = (await vehicles(request)).find((v) => v.powertrain === 'ICE')
  if (ice) {
    await useVehicle(page, ice.id, '/vehicle')
    await expect(page.getByRole('main').getByRole('button', { name: /Resale value/ })).toBeVisible()
    await expect(page.getByRole('main').getByRole('button', { name: /Battery health/ })).toHaveCount(0)
  }
})

test('the consumption is a tile of the vehicle sheet that opens the energy page, no longer a block of the home', async ({ page }) => {
  await useVehicle(page, ev.id, '/vehicle')
  const tile = page.getByRole('main').getByRole('link', { name: /Consumption/ })
  await expect(tile).toHaveAttribute('href', '/energy')
  await tile.click()
  await expect(page).toHaveURL(/\/energy/)
  await page.goBack()
  await expect(page).toHaveURL(/\/vehicle$/)
  await useVehicle(page, ev.id, '/')
  await expect(page.getByRole('main').getByText(/kWh\/100/)).toHaveCount(0)
})

test('the tariff plans are a row of the vehicle sheet for a vehicle that charges, no longer on the account page', async ({ page, request }) => {
  await useVehicle(page, ev.id, '/vehicle')
  await page.getByRole('main').getByRole('button', { name: /Home charging/ }).click()
  const sheet = page.getByRole('dialog', { name: 'Electricity tariff plans' })
  await expect(sheet.getByRole('button', { name: 'New plan' })).toBeVisible()
  await page.goBack()
  await expect(sheet).toHaveCount(0)
  await page.goto('/account')
  await expect(page.getByRole('main').getByText('Electricity tariff plans')).toHaveCount(0)
  const ice = (await vehicles(request)).find((v) => v.powertrain === 'ICE')
  if (ice) {
    await useVehicle(page, ice.id, '/vehicle')
    await expect(page.getByRole('main').getByRole('button', { name: /Home charging/ })).toHaveCount(0)
  }
})

// The keyboard itself cannot be tested headless: the attribute that picks it can. A whole-kilometre field has no comma key.
test('the vehicle sheet lists the screens of the vehicle, each opens and Back returns to the sheet', async ({ page }) => {
  await useVehicle(page, ev.id, '/vehicle')
  const main = page.getByRole('main')
  await expect(main.getByRole('heading', { name: ev.name })).toBeVisible()
  await expect(main.getByRole('link', { name: 'Record' })).toHaveAttribute('href', '/odometer')
  await main.getByRole('link', { name: 'Tires' }).click()
  await expect(page).toHaveURL(/\/tires$/)
  await page.goBack()
  await expect(page).toHaveURL(/\/vehicle$/)
  // The "More" sheet no longer repeats what the sheet opens
  await page.getByRole('button', { name: 'More' }).click()
  const more = page.getByRole('dialog')
  await expect(more.getByRole('link', { name: 'Vehicle' })).toBeVisible()
  await expect(more.getByRole('link', { name: 'Tires' })).toHaveCount(0)
  await expect(more.getByRole('link', { name: 'Comparison' })).toHaveCount(0)
})

test('an odometer field asks for the numeric keyboard, a price for the decimal one', async ({ page }) => {
  await useVehicle(page, ev.id, '/odometer')
  await page.getByRole('main').getByRole('button', { name: 'Add the reading' }).click()
  const sheet = page.getByRole('dialog', { name: 'New odometer reading' })
  await expect(sheet.getByLabel(/^Mileage/)).toHaveAttribute('inputmode', 'numeric')
  await useVehicle(page, ev.id, '/tires')
  await page.getByRole('button', { name: 'Add tires' }).click()
  const price = page.locator('#tire-add-tire-price')
  await expect(price).toHaveAttribute('inputmode', 'decimal')
  await expect(price).toHaveAttribute('type', 'text')
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
  await useVehicle(page, ev.id, '/costs?theme=energy')
  const cards = page.getByTestId('cost-entry').getByRole('button')
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
  await page.getByTestId('period-bar').getByRole('button', { name: /Choose the period/ }).click()
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
    await useVehicle(page, ev.id, '/costs?theme=other')
    const row = page.getByTestId('cost-entry').filter({ hasText: description }).first()
    await row.scrollIntoViewIfNeeded()
    await page.waitForTimeout(600) // the list settles (figures, charts) before the row is measured
    const box = (await row.boundingBox())!
    const y = box.y + box.height / 2

    // A short drag slides back and neither deletes nor opens the row
    await drag(page, 340, y, -50, 0)
    await expect(page.getByText(description)).toBeVisible()
    await expect(page.getByRole('dialog')).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'Undo', exact: true })).toHaveCount(0)

    // A vertical move is a scroll: the row stays
    await drag(page, 340, y, -10, 60)
    await expect(page.getByText(description)).toBeVisible()

    await drag(page, 340, y, -200, 4)
    await expect(page.getByText(description)).toHaveCount(0)
    await expect(page.getByRole('dialog')).toHaveCount(0)
    expect(await rows()).toBe(1)
    await page.getByRole('button', { name: 'Undo', exact: true }).click()
    await expect(page.getByText(description)).toBeVisible()

    await drag(page, 340, (await page.getByTestId('cost-entry').filter({ hasText: description }).first().boundingBox())!.y + box.height / 2, -200, 4)
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
    const rowOf = () => page.locator('div.rounded-2xl', { hasText: notes }).last()
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

test('touching the tab of the page already shown brings it back to the top', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const scroller = page.locator('.app-scroll')
  await scroller.evaluate((el) => el.scrollTo({ top: 600 }))
  await expect.poll(() => scroller.evaluate((el) => el.scrollTop)).toBeGreaterThan(300)
  await page.getByRole('navigation', { name: 'Main navigation' }).last().getByRole('link', { name: 'Drives' }).click()
  await expect.poll(() => scroller.evaluate((el) => el.scrollTop)).toBe(0)
})

test('the "More" sheet groups the pages, with sign out last and set apart', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  await page.getByRole('button', { name: 'More' }).click()
  const sheet = page.getByRole('dialog', { name: 'Other pages' })
  await expect(sheet.getByRole('group')).not.toHaveCount(0)
  const signOut = sheet.getByRole('button', { name: 'Sign out' })
  // Last control of the sheet, below every page link
  const lastY = await sheet.getByRole('link').last().boundingBox()
  const outY = await signOut.boundingBox()
  expect(outY!.y).toBeGreaterThan(lastY!.y + lastY!.height)
  // Never the raw development marker
  await expect(sheet.getByText(/\bdev$/)).toHaveCount(0)
})

test('controls of the interface are not selectable text, values are', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  const bar = page.getByRole('navigation', { name: 'Main navigation' }).last()
  expect(await bar.evaluate((el) => getComputedStyle(el).userSelect)).toBe('none')
  expect(await page.locator('main').evaluate((el) => getComputedStyle(el).userSelect)).not.toBe('none')
})

test('the home header has no "+ Expense" button on a phone: the "+" of the bar does that', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  await expect(page.getByRole('heading', { name: 'Dashboard' }).first()).toBeVisible()
  await expect(page.getByRole('main').getByRole('link', { name: 'Expense', exact: true })).toBeHidden()
})

test('an unknown address shows a page that says so, with the way home', async ({ page }) => {
  await useVehicle(page, ev.id, '/no/such/page')
  await expect(page).toHaveURL(/\/no\/such\/page$/)
  await expect(page.getByRole('heading', { name: 'Page not found' }).last()).toBeVisible()
  await page.getByRole('main').getByRole('link', { name: 'Home' }).click()
  await expect(page).toHaveURL(/\/(\?period=[^&]*)?$/)
})

test('the Trips tab is in the address: it survives a reload and leaves the address with the Drives tab', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await page.getByRole('tab', { name: 'Trips' }).click()
  await expect(page).toHaveURL(/\/drives\?tab=TRIPS$/)
  await page.reload()
  await expect(page.getByRole('tab', { name: 'Trips' })).toHaveAttribute('aria-selected', 'true')
  await page.getByRole('tab', { name: 'Drives' }).click()
  await expect(page).toHaveURL(/\/drives$/)
  await page.goto('/drives?tab=TRIPS')
  await expect(page.getByRole('tab', { name: 'Trips' })).toHaveAttribute('aria-selected', 'true')
})

test('a list that cannot load offline says so in plain words, and does not claim the vehicle is tracked by hand', async ({ page }) => {
  await page.route(/\/api\/vehicles\/[^/]+\/drives(\?|$)/, (route) => route.abort('internetdisconnected'))
  await useVehicle(page, ev.id, '/drives')
  await expect(page.getByRole('alert').getByText(/offline/i)).toBeVisible()
  await expect(page.getByText(/Failed to fetch/i)).toHaveCount(0)
  await expect(page.getByText(/manual or semi-automatic/i)).toHaveCount(0)
})

test.describe('signed out', () => {
  test.use({ storageState: { cookies: [], origins: [] } })

  test('the sign-in and sign-up fields tell the phone what to fill and what the keyboard key does', async ({ page }) => {
    await page.goto('/login')
    const attrs = (id: string) => page.locator(id).evaluate((e) => [e.getAttribute('autocomplete'), e.getAttribute('enterkeyhint')])
    expect(await attrs('#login-email')).toEqual(['username', 'next'])
    expect(await attrs('#login-password')).toEqual(['current-password', 'go'])
    await page.goto('/register')
    expect(await attrs('#register-email')).toEqual(['username', 'next'])
    expect(await attrs('#register-password')).toEqual(['new-password', 'next'])
    expect(await attrs('#register-confirm-password')).toEqual(['new-password', 'go'])
  })
})

test('the sign-out button of a session and the edit link of the energy estimate have a 44px hit area', async ({ page }) => {
  await useVehicle(page, ev.id, '/account')
  const session = page.getByRole('button', { name: /^Sign out / }).first()
  await expect(session).toBeVisible()
  const a = (await session.boundingBox())!
  expect(Math.min(a.width, a.height)).toBeGreaterThanOrEqual(44)

  await useVehicle(page, ev.id, '/energy')
  const edit = page.getByRole('button', { name: 'Edit', exact: true })
  await expect(edit).toBeVisible()
  const b = (await edit.boundingBox())!
  expect(Math.min(b.width, b.height)).toBeGreaterThanOrEqual(44)
})

test('the selection bar counter is not cut on a phone, and the sheet ends with Delete set apart', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const row = page.locator('div.relative.card', { has: page.getByRole('button', { name: /^Open the cost breakdown/ }) }).first()
  const box = (await row.boundingBox())!
  const cdp = await page.context().newCDPSession(page)
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: box.x + 120, y: box.y + box.height / 2 }] })
  const bar = page.locator('[data-selection-bar]')
  await expect(bar).toBeVisible()
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })

  const counter = bar.getByText('1 selected', { exact: true })
  await expect(counter).toBeVisible()
  expect(await counter.evaluate((e) => (e.parentElement as HTMLElement).scrollWidth <= (e.parentElement as HTMLElement).clientWidth)).toBe(true)

  await bar.getByRole('button', { name: 'More actions' }).tap()
  const items = page.getByRole('dialog').getByRole('menuitem')
  await expect(items.last()).toHaveText('Delete')
  await expect(page.getByRole('dialog').getByRole('separator')).not.toHaveCount(0)
})

test('the chart hint says "Tap" on a touch screen', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  await page.getByRole('button', { name: /^Mileage and cost/ }).first().tap()
  await expect(page.getByText(/^Tap a bar of the chart/)).toBeVisible()
  await expect(page.getByText(/^Click a bar of the chart/)).toBeHidden()
})

// The system Back (page.goBack) closes the layer on top before it leaves the page.
const path = (page: import('@playwright/test').Page) => new URL(page.url()).pathname
const openMore = (page: import('@playwright/test').Page) => page.getByRole('navigation').getByRole('button', { name: 'More' }).tap()

test('Back closes the More sheet and stays on the page', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await openMore(page)
  const sheet = page.getByRole('dialog')
  await expect(sheet).toBeVisible()
  await page.goBack()
  await expect(sheet).toHaveCount(0)
  expect(path(page)).toBe('/drives')
})

test('a link of the More sheet opens its page, and Back from there returns to the page before, not to a copy of it', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const before = await page.evaluate(() => history.length)
  await openMore(page)
  await page.getByRole('dialog').getByRole('link', { name: 'Carpooling' }).tap()
  await expect(page).toHaveURL(/\/carpools$/)
  await expect(page.getByRole('dialog')).toHaveCount(0)
  expect(await page.evaluate(() => history.length)).toBe(before + 1)
  await page.goBack()
  await expect(page).toHaveURL(/\/drives$/)
})

test('Back closes a modal of a list, then leaves the page', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const start = path(page)
  await page.getByRole('button', { name: /^Open the cost breakdown/ }).first().tap()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.goBack()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  expect(path(page)).toBe(start)
  await page.goBack()
  expect(path(page)).not.toBe(start)
})

test('a modal closed by its own button leaves no entry behind', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const before = await page.evaluate(() => history.length)
  await page.getByRole('button', { name: /^Open the cost breakdown/ }).first().tap()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dialog).toHaveCount(0)
  await page.waitForTimeout(300)
  await page.goBack()
  expect(path(page)).not.toBe('/drives')
  expect(before).toBeGreaterThan(0)
})

test('Back closes the layers one at a time: the selection sheet, then the selection, then the page', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const row = page.locator('div.relative.card', { has: page.getByRole('button', { name: /^Open the cost breakdown/ }) }).first()
  const box = (await row.boundingBox())!
  const cdp = await page.context().newCDPSession(page)
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: box.x + 120, y: box.y + box.height / 2 }] })
  const bar = page.locator('[data-selection-bar]')
  await expect(bar).toBeVisible()
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] })
  await bar.getByRole('button', { name: 'More actions' }).tap()
  await expect(page.getByRole('dialog')).toBeVisible()

  await page.goBack()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(bar).toBeVisible()
  await page.goBack()
  await expect(bar).toHaveCount(0)
  expect(path(page)).toBe('/drives')
  await page.goBack()
  expect(path(page)).not.toBe('/drives')
})

// The shell scrolls inside .app-scroll: Back puts a page back where it was left, a new page starts at the top.
const shellTop = (page: import('@playwright/test').Page) => page.evaluate(() => document.querySelector('.app-scroll')!.scrollTop)
const scrollShell = async (page: import('@playwright/test').Page, top: number) => {
  await page.evaluate((y) => document.querySelector('.app-scroll')!.scrollTo(0, y), top)
  await expect.poll(() => shellTop(page)).toBeGreaterThan(top - 3)
}
const openCarpooling = async (page: import('@playwright/test').Page) => {
  await openMore(page)
  await page.getByRole('dialog').getByRole('link', { name: 'Carpooling' }).tap()
  await expect(page).toHaveURL(/\/carpools$/)
}

test('Back puts a page where it was left, and a page opened afresh starts at the top', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await scrollShell(page, 800)
  const left = await shellTop(page)
  await openCarpooling(page)
  await expect.poll(() => shellTop(page)).toBe(0)
  await page.goBack()
  await expect(page).toHaveURL(/\/drives$/)
  await expect.poll(() => shellTop(page), { timeout: 5000 }).toBeGreaterThan(left - 3)
  expect(await shellTop(page)).toBeLessThan(left + 3)
})

test('going forward again restores the page too, and two visits of a page keep their own position', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await scrollShell(page, 500)
  const first = await shellTop(page)
  await openCarpooling(page)
  await page.goBack()
  await expect.poll(() => shellTop(page)).toBeGreaterThan(first - 3)
  await page.goForward()
  await expect(page).toHaveURL(/\/carpools$/)
  await expect.poll(() => shellTop(page)).toBe(0)
  await page.goBack()
  await expect.poll(() => shellTop(page), { timeout: 5000 }).toBeGreaterThan(first - 3)
})

test('a page that loads slowly is put back once it is tall enough', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await scrollShell(page, 800)
  const left = await shellTop(page)
  await openCarpooling(page)
  await page.route(/\/api\/vehicles\/[^/]+\/drives(\?|$)/, async (route) => {
    await new Promise((r) => setTimeout(r, 700))
    await route.continue()
  })
  await page.goBack()
  await expect(page).toHaveURL(/\/drives$/)
  await expect.poll(() => shellTop(page), { timeout: 5000 }).toBeGreaterThan(left - 3)
})

test('a reload brings the page back to where it was', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  await scrollShell(page, 800)
  const left = await shellTop(page)
  await openCarpooling(page)
  await page.goBack()
  await expect.poll(() => shellTop(page)).toBeGreaterThan(left - 3)
  await page.reload()
  await expect.poll(() => shellTop(page), { timeout: 5000 }).toBeGreaterThan(left - 3)
})

// A vehicle of its own with two fitted tires (front left "Alpha", rear left "Beta"), removed at the end.
async function withFittedTires(request: import('@playwright/test').APIRequestContext, fn: (vehicleId: string) => Promise<void>, storage = false) {
  const headers = await authHeaders(request)
  const created = await request.post('/api/vehicles', { headers, data: { name: 'Tire rotation reference', powertrain: 'EV', currency: 'EUR', current_odometer: 1000 } })
  expect(created.ok()).toBeTruthy()
  const vehicle = await created.json()
  try {
    const tire = (model: string, current_position: string) =>
      request.post(`/api/vehicles/${vehicle.id}/tires`, {
        headers,
        data: { brand: model, model: 'Grip', dimension: '205/55R16', season: 'SUMMER', purchase_date: '2026-01-01', purchase_price: 9000, current_position, mounted_odometer: current_position === 'STORAGE' ? undefined : 0, initial_depth_mm: 8, min_legal_depth_mm: 1.6, estimated_lifespan_km: 40000 },
      })
    expect((await tire('Alpha', 'FL')).ok()).toBeTruthy()
    expect((await tire('Beta', 'RL')).ok()).toBeTruthy()
    if (storage) expect((await tire('Gamma', 'STORAGE')).ok()).toBeTruthy()
    await fn(vehicle.id)
  } finally {
    await request.delete(`/api/vehicles/${vehicle.id}`, { headers })
  }
}

test('a quick rotation shows at once, comes back with Undo, and is only sent once the time to undo is over', async ({ page, request }) => {
  await withFittedTires(request, async (vehicleId) => {
    const sent: string[] = []
    await page.route(/\/tires\/quick-rotate$/, (route) => {
      sent.push(route.request().postData() ?? '')
      return route.continue()
    })
    await useVehicle(page, vehicleId, '/tires')
    const tiles = page.getByRole('main').getByRole('button', { name: /^Open / })
    await expect(tiles.first()).toHaveAccessibleName(/Alpha/)

    await page.getByRole('button', { name: 'Front ⇄ Rear' }).tap()
    await expect(tiles.first()).toHaveAccessibleName(/Beta/)
    await expect(page.getByText('Tires rotated')).toBeVisible()
    // One rotation at a time while the time to undo runs
    await expect(page.getByRole('button', { name: 'Front ⇄ Rear' })).toBeDisabled()
    await page.getByRole('button', { name: 'Undo' }).tap()
    await expect(tiles.first()).toHaveAccessibleName(/Alpha/)
    await page.waitForTimeout(500)
    expect(sent).toHaveLength(0)

    await page.getByRole('button', { name: 'Front ⇄ Rear' }).tap()
    await expect(tiles.first()).toHaveAccessibleName(/Beta/)
    await expect.poll(() => sent.length, { timeout: 12_000 }).toBe(1)
    expect(sent[0]).toContain('FRONT_BACK')
    await expect(tiles.first()).toHaveAccessibleName(/Beta/)
  })
})

test('"Change set" is disabled with its reason when no tire is in storage, and opens the swap when one is', async ({ page, request }) => {
  await withFittedTires(request, async (vehicleId) => {
    await useVehicle(page, vehicleId, '/tires')
    const button = page.getByRole('button', { name: 'Change set' })
    await expect(button).toBeDisabled()
    await expect(page.getByText('No tire set in storage')).toBeVisible()
    await expect(button).toHaveAccessibleDescription('No tire set in storage')
  })
  await withFittedTires(
    request,
    async (vehicleId) => {
      await useVehicle(page, vehicleId, '/tires')
      await expect(page.getByRole('button', { name: 'Change set' })).toBeEnabled()
      await expect(page.getByText('No tire set in storage')).toHaveCount(0)
      await page.getByRole('button', { name: 'Change set' }).tap()
      await expect(page.getByRole('dialog')).toBeVisible()
    },
    true,
  )
})

test('a definition opens on touch, and Back closes it', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  const tip = page.getByRole('main').getByRole('button', { name: /^Economic cost: / }).first()
  await tip.scrollIntoViewIfNeeded()
  const box = (await tip.boundingBox())!
  expect(box.width).toBeGreaterThanOrEqual(43)
  expect(box.height).toBeGreaterThanOrEqual(43)
  const before = page.url()
  await tip.tap()
  const sheet = page.getByRole('dialog', { name: 'Economic cost' })
  await expect(sheet).toBeVisible()
  await expect(sheet).toContainText('Costs spread over the period they cover')
  // Above the bottom bar, not stacked under it by the card that holds the tip
  const definition = sheet.getByText('Costs spread over')
  await expect
    .poll(() =>
      definition.evaluate((el) => {
        const r = el.getBoundingClientRect()
        const top = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2)
        return !!top && !el.contains(top)
      }),
    )
    .toBe(false)
  await page.goBack()
  await expect(sheet).toHaveCount(0)
  expect(page.url()).toBe(before)
})

test('an account without a vehicle sees one "Add a vehicle first" state on every page about a vehicle', async ({ page }) => {
  await page.route(/\/api\/vehicles$/, (route) => (route.request().method() === 'GET' ? route.fulfill({ json: [] }) : route.continue()))
  for (const route of ['/', '/drives', '/carpools', '/odometer', '/tires', '/expenses', '/maintenance', '/energy', '/fleet', '/comparison']) {
    await page.goto(route)
    await expect(page.getByRole('heading', { name: 'Add a vehicle first' }), route).toBeVisible()
    await expect(page.getByRole('main').getByRole('link', { name: 'Add a vehicle' }), route).toHaveAttribute('href', '/vehicles')
    // None of the page's own actions
    await expect(page.getByRole('main').getByRole('button'), route).toHaveCount(0)
  }
  // The vehicles page is where the first one is added
  await page.goto('/vehicles')
  await expect(page.getByRole('heading', { name: 'Add a vehicle first' })).toHaveCount(0)
})

test('the authorization page has no tab bar or vehicle picker, and its buttons stay at the bottom of the screen', async ({ page, request }) => {
  const { authorizeUrl, pkce, registerClient } = await import('./oauthFlow')
  const clientId = await registerClient(request, 'Layout assistant')
  await page.goto(authorizeUrl(clientId, pkce().challenge, 'layout'))
  await expect(page.getByRole('heading', { name: 'Authorize an application' })).toBeVisible()
  await expect(page.getByRole('navigation')).toHaveCount(0)
  await expect(page.getByRole('combobox')).toHaveCount(0)
  const approve = page.getByRole('button', { name: 'Approve' })
  await expect(approve).toBeVisible()
  const box = (await approve.boundingBox())!
  expect(box.y + box.height).toBeLessThanOrEqual(844)
  expect(box.y).toBeGreaterThan(844 - 140)
  // Still in reach after scrolling the whole page
  await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight))
  const after = (await approve.boundingBox())!
  expect(after.y + after.height).toBeLessThanOrEqual(844)
})

test('in the month detail the two modes share the track, and the fixed/variable line has no stray bullet', async ({ page }) => {
  await useVehicle(page, ev.id, '/')
  await page.getByRole('main').getByRole('button', { name: /spent so far/ }).first().tap()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  const modes = dialog.locator('button[aria-pressed]')
  await expect(modes).toHaveCount(2)
  const [a, b] = [(await modes.nth(0).boundingBox())!, (await modes.nth(1).boundingBox())!]
  expect(Math.abs(a.width - b.width)).toBeLessThan(2)
  // Together they fill the track (border, padding and gap apart)
  const track = (await modes.nth(0).locator('xpath=..').boundingBox())!
  expect(a.width + b.width).toBeGreaterThan(track.width - 14)
  await expect(dialog.getByText('•', { exact: true })).toBeHidden()
})

test.describe('the period bar of the home page', () => {
  const bar = (page: Page) => page.getByTestId('period-bar')

  test('an arrow goes to the neighbouring period, the URL follows without a history entry, and a reload keeps it', async ({ page }) => {
    await useVehicle(page, ev.id, '/')
    await expect(bar(page)).toBeVisible()
    await expect(page).toHaveURL(/period=12M/)
    const before = await page.evaluate(() => history.length)
    await bar(page).getByRole('button', { name: 'Previous period' }).tap()
    await expect(page).toHaveURL(/period=12M&end=\d{4}-\d{2}/)
    expect(await page.evaluate(() => history.length)).toBe(before)
    const label = await bar(page).getByRole('button', { name: /Choose the period/ }).getAttribute('aria-label')
    await page.reload()
    await expect(bar(page).getByRole('button', { name: /Choose the period/ })).toHaveAttribute('aria-label', label!)
    await expect(bar(page).getByRole('button', { name: 'Next period' })).toBeEnabled()
  })

  test('the system Back closes the sheet and leaves the URL as it was', async ({ page }) => {
    await useVehicle(page, ev.id, '/')
    const url = page.url()
    await bar(page).getByRole('button', { name: /Choose the period/ }).tap()
    const dialog = page.getByRole('dialog')
    await expect(dialog).toBeVisible()
    await page.goBack()
    await expect(dialog).toBeHidden()
    expect(page.url()).toBe(url)
  })

  test('a preset closes the sheet and changes the period; "Custom" shows labelled From and To fields with 44px targets', async ({ page }) => {
    await useVehicle(page, ev.id, '/')
    await bar(page).getByRole('button', { name: /Choose the period/ }).tap()
    const dialog = page.getByRole('dialog')
    await dialog.getByRole('button', { name: 'Last 3 months' }).tap()
    await expect(dialog).toBeHidden()
    await expect(page).toHaveURL(/period=3M/)

    await bar(page).getByRole('button', { name: /Choose the period/ }).tap()
    await expect(dialog.locator('input[type=date]')).toHaveCount(0)
    await dialog.getByRole('button', { name: 'Custom' }).tap()
    await expect(dialog.getByText('From', { exact: true })).toBeVisible()
    await expect(dialog.getByText('To', { exact: true })).toBeVisible()
    const fields = dialog.locator('input[type=date]')
    await expect(fields).toHaveCount(2)
    for (const box of await Promise.all([fields.nth(0).boundingBox(), fields.nth(1).boundingBox()])) expect(box!.height).toBeGreaterThanOrEqual(44)
    await fields.nth(0).fill('2026-01-10')
    await fields.nth(1).fill('2026-02-20')
    await dialog.getByRole('button', { name: 'Apply' }).tap()
    await expect(page).toHaveURL(/from=2026-01-10&to=2026-02-20/)
  })

  test('the bar gives way while the page is read downwards and comes back upwards, above the tab bar', async ({ page }) => {
    await useVehicle(page, ev.id, '/')
    const nav = (await page.locator('nav[aria-label]').last().boundingBox())!
    const shown = (await bar(page).boundingBox())!
    expect(shown.y + shown.height).toBeLessThanOrEqual(nav.y)
    await page.evaluate(() => document.querySelector('.app-scroll')!.scrollTo(0, 600))
    await expect.poll(async () => (await bar(page).boundingBox())!.y).toBeGreaterThanOrEqual(844)
    await page.evaluate(() => document.querySelector('.app-scroll')!.scrollTo(0, 400))
    await expect.poll(async () => (await bar(page).boundingBox())!.y).toBeLessThan(nav.y)
  })

  for (const width of [360, 430]) {
    test(`no horizontal overflow at ${width}px`, async ({ page }) => {
      await page.setViewportSize({ width, height: 844 })
      await useVehicle(page, ev.id, '/')
      await expect(bar(page)).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBe(width)
    })
  }
})

test.describe('the period bar of the drives list', () => {
  const bar = (page: Page) => page.getByTestId('period-bar')

  test('the period is a bar above the tab bar, the Filters sheet no longer holds it, and the sheet reaches the screen edge', async ({ page }) => {
    await useVehicle(page, ev.id, '/drives')
    await expect(bar(page)).toBeVisible()
    await expect(bar(page)).toContainText('All')
    await expect(bar(page).getByRole('button', { name: 'Previous period' })).toBeDisabled()

    await page.getByRole('button', { name: 'Filters' }).first().tap()
    const filters = page.getByRole('dialog', { name: 'Filters' })
    await expect(filters).toBeVisible()
    await expect(filters.getByRole('group', { name: 'Period' })).toHaveCount(0)
    // The layer covers the whole screen: no strip of the tab bar shows under the sheet
    const overlay = filters.locator('xpath=..')
    const box = (await overlay.boundingBox())!
    expect(Math.round(box.y + box.height)).toBe(page.viewportSize()!.height)
  })

  test('a unit chosen in the sheet is stepped with the arrows and the list reloads', async ({ page }) => {
    await useVehicle(page, ev.id, '/drives')
    await bar(page).getByRole('button', { name: /Choose the period/ }).tap()
    const sheet = page.getByRole('dialog', { name: 'Period' })
    await sheet.getByRole('button', { name: 'Month', exact: true }).tap()
    await expect(sheet.getByRole('button', { name: 'Month', exact: true })).toHaveAttribute('aria-pressed', 'true')
    await page.goBack()
    await expect(sheet).toBeHidden()
    await expect(bar(page).getByRole('button', { name: 'Next period' })).toBeDisabled()

    const reload = page.waitForRequest((req) => /\/api\/vehicles\/[^/]+\/drives\?.*from=/.test(req.url()))
    await bar(page).getByRole('button', { name: 'Previous period' }).tap()
    await reload
    await expect(bar(page).getByRole('button', { name: 'Next period' })).toBeEnabled()
  })
})

test('the costs page lists the cost lines of the period under their figures, and a theme filters both', async ({ page }) => {
  await useVehicle(page, ev.id, '/costs')
  const headline = page.getByTestId('costs-headline')
  await expect(headline).toContainText('Net cash out')
  await expect(page.getByTestId('cost-entry').first()).toBeVisible()
  const all = await headline.innerText()

  await page.getByTestId('costs-filter-tolls').click()
  await expect(headline).toContainText('Cash out · Tolls')
  await expect(page).toHaveURL(/theme=tolls/)
  await expect(page.getByTestId('cost-entry').first()).toBeVisible()
  expect(await headline.innerText()).not.toBe(all)

  // The filter is in the address: a reload keeps it
  await page.reload()
  await expect(page.getByTestId('costs-filter-tolls')).toHaveAttribute('aria-pressed', 'true')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('a line of the costs page opens the form of its record, read from the server', async ({ page }) => {
  await useVehicle(page, ev.id, '/costs?theme=tolls')
  const row = page.getByTestId('cost-entry').first()
  await expect(row).toBeVisible()
  await row.getByRole('button').click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.locator('input').first()).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
})

test('the energy theme of the costs page adds the figures by month and by place, under the one period bar', async ({ page }) => {
  await useVehicle(page, ev.id, '/costs')
  await expect(page.getByRole('heading', { name: 'Charging by place' })).toHaveCount(0)
  await page.getByTestId('costs-filter-energy').click()
  const analysis = page.getByRole('region', { name: 'Charging by place' })
  await expect(analysis).toBeVisible()
  // The period is the bar of the page, not a second control in the analysis
  await expect(analysis.getByRole('button', { name: /month|Last|Current/i })).toHaveCount(0)
  await analysis.getByRole('button', { name: 'Places', exact: true }).click()
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('the costs page draws the monthly charts for the whole cost, and only for the whole cost', async ({ page }) => {
  await useVehicle(page, ev.id, '/costs')
  await expect(page.getByRole('heading', { name: /Monthly expense trend/ })).toBeVisible()
  // A theme is a subset of the cost: the charts of the whole cost are not shown under it
  await page.getByTestId('costs-filter-tolls').click()
  await expect(page.getByRole('heading', { name: /Monthly expense trend/ })).toHaveCount(0)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})

test('the month detail is titled by its month alone and a swipe moves between months', async ({ page }) => {
  await useVehicle(page, ev.id, '/costs')
  await page.getByRole('button', { name: /Last month/ }).click()
  const dialog = page.getByRole('dialog')
  const title = dialog.getByRole('heading', { level: 3 })
  await expect(title).toBeVisible()
  const last = await title.innerText()
  expect(last).toMatch(/^[A-Za-z]+ \d{4}$/)

  const body = dialog.getByTestId('month-detail-body')
  const box = (await body.boundingBox())!
  const y = Math.min(box.y + 120, 600)
  // Swiping right brings the month before
  await drag(page, 100, y, 180, 4)
  await expect(title).not.toHaveText(last)
  const before = await title.innerText()
  // Swiping left brings the month after, back to the last one
  await drag(page, 300, y, -180, 4)
  await expect(title).toHaveText(last)
  expect(before).not.toBe(last)
  // A swipe down is the sheet's own gesture: the month does not change by it
  await swipeDown(page, 200, y, 40)
  await expect(title).toHaveText(last)
})

test('the old addresses of the lists land on the costs page, and the tool pages keep their tabs', async ({ page }) => {
  await useVehicle(page, ev.id, '/expenses?tab=TOLLS')
  await expect(page).toHaveURL(/\/costs\?theme=tolls/)
  await expect(page.getByTestId('costs-filter-tolls')).toHaveAttribute('aria-pressed', 'true')

  await page.goto('/maintenance')
  await expect(page).toHaveURL(/\/costs\?theme=maintenance/)

  await page.goto('/maintenance?tab=REMINDERS')
  await expect(page).toHaveURL(/\/maintenance\?tab=REMINDERS$/)
  await expect(page.getByRole('heading', { name: 'Reminders' })).toBeVisible()

  await page.goto('/energy')
  await expect(page).toHaveURL(/\/energy/)
  await page.getByTestId('charges-list-link').click()
  await expect(page).toHaveURL(/\/costs\?theme=energy/)
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
})
