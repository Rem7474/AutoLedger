import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// ActionMenu ("⋯") belongs to one card or one page, never to a line of a list: a row opens its detail, its actions are
// in the detail and in the selection bar. A new user of the component has to be added here on purpose.
const ALLOWED = ['components/vehicles/VehicleCard.vue', 'views/ManualTrackingView.vue']

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !/\.test\.ts$/.test(name) ? [path] : []
  })
}

describe('ActionMenu', () => {
  it('is imported only by the files allowed to carry a card or page menu', () => {
    const root = join(__dirname, '..')
    const users = sources(root)
      .map((file) => relative(root, file))
      .filter((rel) => /components\/ActionMenu\.vue['"]/.test(readFileSync(join(root, rel), 'utf8')))
    expect(users.sort()).toEqual(ALLOWED)
  })
})
