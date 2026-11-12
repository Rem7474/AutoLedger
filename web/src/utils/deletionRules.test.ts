import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A deletion does not ask first: the row goes at once and a snackbar offers to take it back (`deleteWithUndo`,
// `deleteRowsWithUndo`). A question stays only where the deletion cannot be taken back or reaches beyond the row.
// The list below is closed: a file outside it that calls `showConfirm` or `showChoice` fails, and an entry whose file no
// longer calls them fails too, so the list stays exact.
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !name.endsWith('.test.ts') ? [path] : []
  })
}

const QUESTIONS_KEPT: Record<string, string> = {
  // deletions that stay a question
  'views/VehiclesView.vue': 'deletes a vehicle and everything recorded for it',
  'components/vehicles/VehicleMembersModal.vue': 'ends a share or removes a person',
  'components/account/ApiTokensSection.vue': 'revokes an API token',
  'components/vehicles/VehicleDataSourcesModal.vue': 'disconnects the data source of a vehicle',
  'components/CSVImportModal.vue': 'undoes an import batch',
  'views/AccountView.vue': 'ends sessions of the account',
  'components/expenses/WebhookModal.vue': 'revokes the token the Home Assistant automation holds',
  'components/drives/DriveCostTolls.vue': 'a cost shared with a trip changes the totals of its other drives',
  'components/vehicles/OwnershipWizardModal.vue': 'ends the vehicle ownership record',
  'composables/useTripGroups.ts': 'a trip with costs: delete them with it or keep them unlinked',
  // questions that are not deletions
  'components/carpool/CarpoolTripModal.vue': 'the difference before the amounts follow the drives again',
  'views/CarpoolView.vue': 'the difference before the amounts follow the drives again',
  'composables/useDriveBulkActions.ts': 'applies the detected tolls to the selection',
}

describe('deletion rules', () => {
  it('asks a question only in the files that keep one', () => {
    const root = join(__dirname, '..')
    const asking = sources(root)
      .map((file) => relative(root, file))
      .filter((rel) => rel !== 'composables/useConfirm.ts')
      .filter((rel) => /\bshow(?:Confirm|Choice)\s*\(/.test(readFileSync(join(root, rel), 'utf8')))
    expect(asking.sort()).toEqual(Object.keys(QUESTIONS_KEPT).sort())
  })

  // A deletion with no question goes through Undo: the server call lives in the `remove` callback of
  // `deleteRowsWithUndo`, never awaited at the click.
  it('deletes the readings, rates, templates and import profiles with Undo', () => {
    const root = join(__dirname, '..')
    for (const rel of ['components/dashboard/BatteryHealthCard.vue', 'components/MileageRatesEditor.vue', 'components/expenses/ReminderTemplatesBar.vue', 'components/CSVImportModal.vue']) {
      const source = readFileSync(join(root, rel), 'utf8')
      expect(source, rel).toMatch(/deleteRowsWithUndo\(/)
      expect(source, rel).toMatch(/<RowDeleteButton\b/)
      expect(source, rel).not.toMatch(/await\s+api\.delete(?:BatteryReading|MileageRate|ReminderTemplate|ImportProfile)\b/)
    }
  })
})
