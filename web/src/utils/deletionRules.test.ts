import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A deletion does not ask first: the row goes at once and a snackbar offers to take it back (`deleteWithUndo`,
// `deleteRowsWithUndo`). A question stays only where the deletion cannot be taken back or reaches beyond the row.
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !name.endsWith('.test.ts') ? [path] : []
  })
}

const QUESTIONS_KEPT = [
  // revokes the token the Home Assistant automation holds
  'components/expenses/WebhookModal.vue',
  // a cost shared with a trip changes the totals of its other drives
  'components/drives/DriveCostTolls.vue',
  // ends the vehicle's ownership record
  'components/vehicles/OwnershipWizardModal.vue',
]

describe('deletion rules', () => {
  it('asks before deleting only in the files that keep a question', () => {
    const root = join(__dirname, '..')
    const asking = sources(root)
      .map((file) => relative(root, file))
      .filter((rel) => /confirmText:\s*t\('common\.delete'\)/.test(readFileSync(join(root, rel), 'utf8')))
    expect(asking.sort()).toEqual([...QUESTIONS_KEPT].sort())
  })
})
