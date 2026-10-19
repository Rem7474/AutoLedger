import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// An amount is shown in the currency of the vehicle that owns it; no source file may fall back to the euro.
// A literal is allowed only where it is a choice offered to the user or example data, or the toll dataset (French
// autoroutes, priced in euros whatever the vehicle's currency).
const ALLOWED = new Set([
  'utils/vehicles.ts', // preselected currency of the new-vehicle form
  'utils/expenses.ts', // the currencies offered by the expense form
  'utils/dataSources.ts', // example rows of the CSV import help
  'views/OnboardingView.vue', // preselected currency of the first vehicle, from the interface language
  'components/drives/DriveCostTolls.vue', // detected toll prices come from the euro dataset
])

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return name === 'locales' ? [] : sources(path)
    return /\.(ts|vue)$/.test(name) && !/\.test\.ts$/.test(name) ? [path] : []
  })
}

describe('currency literals', () => {
  it('keeps the euro out of the sources, outside the allowed places', () => {
    const root = join(__dirname)
    const found = sources(root)
      .filter((file) => readFileSync(file, 'utf8').split('\n').some((line) => !/^\s*(\/\/|\*|\/\*)/.test(line) && /['"`]EUR['"`]/.test(line)))
      .map((file) => relative(root, file))
      .filter((file) => !ALLOWED.has(file))
    expect(found).toEqual([])
  })
})
