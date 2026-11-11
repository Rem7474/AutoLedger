import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A disclosure (`<details>`) hides content, so it is kept for what a reader rarely needs: the advanced settings of a form
// or of an inline set of inputs. Information and lists are shown, or move to the detail of their row. Tabs are `TabBar`
// and carry no counter.
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.vue$/.test(name) ? [path] : []
  })
}

const DISCLOSURES_KEPT = [
  // optional fields of the maintenance form
  'components/expenses/MaintenanceModal.vue',
  // the import form: format help, previous imports, column mapping
  'components/CSVImportModal.vue',
  // the extra assumptions of the what-if form
  'components/comparison/ComparisonWhatIf.vue',
  // the assumption inputs of the resale estimate
  'components/dashboard/ResaleValueCard.vue',
  // the inline form that records a battery reading
  'components/dashboard/BatteryHealthCard.vue',
]

// The passenger picker of the carpool form selects one of the rows edited below it, it is not a page's tab set.
const TABLISTS_KEPT = ['components/TabBar.vue', 'components/carpool/CarpoolPassengers.vue']

describe('disclosure rules', () => {
  const root = join(__dirname, '..')
  const files = sources(root).map((file) => relative(root, file))
  const read = (rel: string) => readFileSync(join(root, rel), 'utf8')

  it('uses <details> only in the files that keep one', () => {
    const using = files.filter((rel) => /<details[\s>]/.test(read(rel)))
    expect(using.sort()).toEqual([...DISCLOSURES_KEPT].sort())
  })

  it('writes no tablist outside TabBar and the passenger picker', () => {
    const tablists = files.filter((rel) => /role="tablist"/.test(read(rel)))
    expect(tablists.sort()).toEqual([...TABLISTS_KEPT].sort())
  })

  it('gives a tab no counter', () => {
    expect(read('components/TabBar.vue')).not.toMatch(/badge/i)
  })
})
