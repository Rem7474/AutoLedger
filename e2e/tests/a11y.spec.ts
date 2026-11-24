import AxeBuilder from '@axe-core/playwright'
import { expect, test } from '@playwright/test'
import { useVehicle, vehicles, type Vehicle } from './helpers'

let ev: Vehicle
let ice: Vehicle

test.beforeAll(async ({ request }) => {
  const all = await vehicles(request)
  ev = all.find((v) => v.powertrain === 'EV')!
  ice = all.find((v) => v.powertrain === 'ICE')!
  expect(ev && ice).toBeTruthy()
})

const routes = ['/', '/?scope=all', '/drives', '/drives?tab=CARPOOLS', '/odometer', '/consumption', '/comparison', '/tires', '/expenses', '/maintenance', '/vehicles', '/account']

for (const width of [1280, 375]) {
  test(`no WCAG 2.1 AA violation at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 })
    for (const route of routes) {
      await useVehicle(page, ev.id, route)
      const { violations } = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()
      expect(violations.map((v) => `${route} ${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)).toEqual([])
    }
  })
}

test('a drive card opens from its button and Space selects it instead of opening it', async ({ page }) => {
  await useVehicle(page, ev.id, '/drives')
  const card = page.getByRole('button', { name: /^Open the cost breakdown of the drive of/ }).first()
  await card.focus()
  await page.keyboard.press('Enter')
  await expect(page.getByRole('dialog')).toBeVisible()
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)

  await card.focus()
  await page.keyboard.press('Space')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(card).toHaveAttribute('aria-pressed', 'true')
  // While a selection is going on a click toggles the row instead of opening it
  await card.click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(card).not.toHaveAttribute('aria-pressed')
  await card.click()
  await expect(page.getByRole('dialog')).toBeVisible()
})

test('the sidebar is a named landmark with four destinations and marks the current page', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ev.id, '/tires')
  const nav = page.getByRole('navigation', { name: 'Main navigation' }).first()
  await expect(nav).toBeVisible()
  // The tires page belongs to the vehicle: one entry is lit
  await expect(nav.locator('a[aria-current="page"]')).toHaveCount(1)
  await expect(nav.locator('a[aria-current="page"]')).toHaveAttribute('href', '/vehicle')
  await expect(nav.getByRole('link')).toHaveText(['Dashboard', 'Drives', 'Expenses', 'Vehicle', 'Garage'])
  await expect(page.getByRole('complementary').getByRole('link', { name: /Account/ })).toBeVisible()
})

test('every page that left the sidebar is reached from the vehicle sheet, the garage or the account link', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  const nav = page.getByRole('navigation', { name: 'Main navigation' }).first()
  const tiles: [RegExp, string][] = [
    [/Maintenance/, '/maintenance'],
    [/Consumption/, '/consumption'],
    [/Tires/, '/tires'],
    [/Odometer readings/, '/odometer'],
    [/Comparison/, '/comparison'],
  ]
  for (const [name, path] of tiles) {
    await useVehicle(page, ev.id, '/vehicle')
    await page.getByRole('main').getByRole('link', { name }).first().click()
    await expect(page).toHaveURL(new RegExp(`${path}(\\?|$)`))
    // The vehicle destination stays lit on those pages
    await expect(nav.locator('a[aria-current="page"]')).toHaveAttribute('href', '/vehicle')
  }
  await nav.getByRole('link', { name: 'Garage' }).click()
  await expect(page).toHaveURL(/\/vehicles$/)
  await expect(nav.locator('a[aria-current="page"]')).toHaveAttribute('href', '/vehicles')
  await page.getByRole('complementary').getByRole('link', { name: /Account/ }).click()
  await expect(page).toHaveURL(/\/account$/)
  await expect(page.getByTestId('about-version')).toBeVisible()
})

test('the sidebar carries no slogan, version or language selector', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ev.id, '/')
  const aside = page.getByRole('complementary')
  await expect(aside).toBeVisible()
  await expect(aside.getByRole('combobox')).toHaveCount(0)
  await expect(aside).not.toContainText(/Version|Signed in|Fleet Manager/)
})

test('a combustion vehicle sidebar has the same four destinations and a drives page', async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 })
  await useVehicle(page, ice.id, '/odometer')
  const nav = page.getByRole('navigation', { name: 'Main navigation' }).first()
  await expect(nav.getByRole('link')).toHaveText(['Dashboard', 'Drives', 'Expenses', 'Vehicle', 'Garage'])
  await expect(nav.locator('a[aria-current="page"]')).toHaveAttribute('href', '/vehicle')
})

test('a focused field shows a visible ring', async ({ page }) => {
  await useVehicle(page, ev.id, '/tires')
  await page.getByRole('button', { name: 'Add tires' }).first().click()
  const brand = page.locator('#tire-add-tire-brand')
  await brand.focus()
  const ring = await brand.evaluate((el) => getComputedStyle(el).boxShadow)
  expect(ring).not.toBe('none')
})

test('the pages of a combustion vehicle have no WCAG 2.1 AA violation', async ({ page }) => {
  for (const route of ['/', '/odometer', '/consumption?tab=FUEL', '/maintenance']) {
    await useVehicle(page, ice.id, route)
    const { violations } = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()
    expect(violations.map((v) => `${route} ${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)).toEqual([])
  }
})

// The sheets a phone opens from a row hold content the page itself does not show: axe reads them open too.
test.describe('sheets opened on a phone', () => {
  test.use({ viewport: { width: 375, height: 800 }, hasTouch: true, isMobile: true })

  const violationsOf = async (page: import('@playwright/test').Page, name: string) => {
    const { violations } = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']).analyze()
    return violations.map((v) => `${name} ${v.id}: ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`)
  }

  test('the resale value sheet has no violation', async ({ page }) => {
    await useVehicle(page, ev.id, '/vehicle')
    await page.getByRole('main').getByRole('button', { name: /Resale value/ }).first().tap()
    await expect(page.getByRole('dialog')).toBeVisible()
    expect(await violationsOf(page, 'resale value')).toEqual([])
  })

  test('the costs page has no violation', async ({ page }) => {
    await useVehicle(page, ev.id, '/costs')
    await expect(page.getByTestId('cost-entry').first()).toBeVisible()
    expect(await violationsOf(page, 'costs')).toEqual([])
  })

  test('the carpool detail sheet has no violation', async ({ page }) => {
    await useVehicle(page, ev.id, '/drives?tab=CARPOOLS&period=all')
    await page.getByRole('main').getByRole('button', { name: /^Open / }).first().tap()
    await expect(page.getByRole('dialog')).toBeVisible()
    expect(await violationsOf(page, 'carpool detail')).toEqual([])
  })
})
