import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A form that creates or edits a record opens as a panel (`<ModalShell pane>`: on the right of a wide screen, a sheet on a
// phone). `ModalShell` without `pane` is a dialog: a choice, a wizard, a preview, a tool, a sheet of options, or a form
// opened from another dialog. The list below is closed: a file outside it that draws a dialog fails, and an entry whose
// file no longer does fails too.
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return name.endsWith('.vue') ? [path] : []
  })
}

const DIALOGS: Record<string, string> = {
  // sheets of options and navigation
  'components/AccountSheet.vue': 'account sheet',
  'components/BulkSelectionBar.vue': 'all the actions of a selection',
  'components/GlossaryTip.vue': 'the definition of a term',
  'components/RowMenu.vue': 'actions of a row',
  'components/SheetRow.vue': 'a row of a sheet that opens a sheet',
  'components/drives/DriveBulkActions.vue': 'driver choice for a selection',
  'components/drives/DrivesPeriodBar.vue': 'period choice',
  'components/drives/DrivesToolbar.vue': 'filters',
  'components/period/PeriodSheet.vue': 'period choice',
  // tools, previews, exports and wizards
  'components/CSVImportModal.vue': 'import',
  'components/ExportDataModal.vue': 'export',
  'components/expenses/DocumentPreviewModal.vue': 'document preview',
  'components/expenses/PublicChargeCalculatorModal.vue': 'public calculator, opened from the charge form',
  'components/dashboard/MonthDetailModal.vue': 'month detail',
  'components/fleet/FleetComparison.vue': 'detail of a vehicle of the fleet',
  'components/vehicles/OwnershipWizardModal.vue': 'ownership wizard',
  'components/account/ApiTokensSection.vue': 'a created token is shown once',
  // details and managers
  'components/carpool/CarpoolDetailModal.vue': 'detail of a carpool',
  'components/drives/DriveCostModal.vue': 'cost breakdown of a drive',
  'components/tires/TireHistoryModal.vue': 'history of a tire',
  'components/manual/OdometerReadingsPanel.vue': 'the reading form is inline on a wide screen, this sheet only exists on a phone',
  'components/expenses/ChargingPlacesModal.vue': 'manager of the places and of their locations',
  'components/vehicles/VehicleMembersModal.vue': 'manager of the shares and persons of a vehicle',
  'components/vehicles/VehicleDataSourcesModal.vue': 'connections of a vehicle',
  // forms that are still a dialog
  'components/carpool/CarpoolTripModal.vue': 'the legs table needs the width',
  'components/drives/AddToTripModal.vue': 'picker of a trip for the selected drives',
  'components/tariffs/TariffPlanModal.vue': 'opened from the tariff editor, itself a dialog',
  'views/VehicleView.vue': 'tariff editor',
  'components/tires/TireBatchDisposeModal.vue': 'opened from another dialog (nested)',
  'components/tires/TireCopyHistoryModal.vue': 'opened from the tire history',
  'components/tires/TireDuplicateSessionModal.vue': 'opened from the tire history',
}

function drawsDialog(source: string): boolean {
  return [...source.matchAll(/<ModalShell\b([^>]*?)>/gs)].some((tag) => !/(^|\s)pane(\s|$)/.test(tag[1]))
}

describe('modal rules', () => {
  it('draws a dialog only in the files that keep one', () => {
    const root = join(__dirname, '..')
    const dialogs = sources(root)
      .map((file) => relative(root, file))
      .filter((rel) => drawsDialog(readFileSync(join(root, rel), 'utf8')))
    expect(dialogs.sort()).toEqual(Object.keys(DIALOGS).sort())
  })

  it('tells a panel from a dialog by its pane attribute', () => {
    expect(drawsDialog('<ModalShell pane v-model:open="open" size="sm">')).toBe(false)
    expect(drawsDialog('<ModalShell\n  v-model:open="open"\n  pane\n>')).toBe(false)
    expect(drawsDialog('<ModalShell v-model:open="open" :title="plane">')).toBe(true)
  })
})
