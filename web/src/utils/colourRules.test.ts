import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A feature does not get an accent hue of its own: a cost's hue is its theme (`text-theme-*`, utils/themes.ts), anything
// else is slate. Green, amber and red are statuses, rose is the brand.
const FORBIDDEN = /\b(?:[a-z-]+:)*(?:text|bg|border|ring|from|to|via|stroke|fill|shadow|divide|outline|placeholder|accent|caret)-(?:violet|indigo|cyan|teal|blue|pink|purple|fuchsia|orange|lime|yellow)-\d{2,3}/g

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !/\.test\.ts$/.test(name) ? [path] : []
  })
}

// A state is coloured through the status tokens (`warning-*`, `success-*`, `info-*`, `critical-*`, `danger-*`) or through
// `StatusLabel` / `CountBadge`, never through a raw amber, emerald, red or sky class. The files below predate the rule:
// the list only shrinks (a file leaves it once migrated), a file outside it that uses one of these classes fails.
const RAW_STATE = /\b(?:[a-z-]+:)*(?:text|bg|border|ring|from|to|via|stroke|fill|shadow|divide|outline|placeholder|accent|caret)-(?:amber|emerald|red|sky)-\d{2,3}/
const RAW_STATE_ALLOWLIST = new Set([
  'components/AppDropzone.vue', 'components/CSVImportModal.vue', 'components/ConfirmModal.vue', 'components/KpiCard.vue',
  'components/KpiRow.vue', 'components/ListSummary.vue', 'components/account/TokenCallLog.vue',
  'components/carpool/CarpoolDetailModal.vue', 'components/carpool/CarpoolLegs.vue', 'components/carpool/CarpoolOverview.vue',
  'components/carpool/CarpoolPassengers.vue', 'components/carpool/CarpoolRecap.vue', 'components/carpool/CarpoolShareMath.vue',
  'components/carpool/CarpoolTripModal.vue', 'components/comparison/ComparisonEmptyState.vue',
  'components/comparison/ComparisonWhatIf.vue', 'components/dashboard/DataQualityCard.vue', 'components/dashboard/LeaseContractCard.vue',
  'components/dashboard/MonthDetailModal.vue', 'components/dashboard/ResaleValueCard.vue', 'components/drives/AddToTripModal.vue',
  'components/drives/DriveCarpoolChip.vue', 'components/drives/DriveCostCarpools.vue',
  'components/drives/DriveCostModal.vue', 'components/drives/DriveCostTolls.vue', 'components/drives/DrivePicker.vue',
  'components/drives/TripEditModal.vue', 'components/drives/TripGroupsPanel.vue', 'components/drives/TripSuggestions.vue',
  'components/expenses/ChargeCard.vue', 'components/expenses/ChargeModal.vue', 'components/expenses/ChargingInsights.vue',
  'components/expenses/ChargingPlacesModal.vue', 'components/expenses/DocumentPreviewModal.vue', 'components/expenses/DocumentsPanel.vue',
  'components/expenses/MaintenanceModal.vue', 'components/expenses/PublicChargeCalculatorModal.vue', 'components/expenses/ReminderModal.vue',
  'components/expenses/ReminderRow.vue', 'components/expenses/ReminderTemplatesBar.vue', 'components/expenses/RemindersPanel.vue',
  'components/expenses/TollModal.vue', 'components/expenses/UploadDocumentModal.vue', 'components/expenses/WebhookModal.vue',
  'components/manual/OdometerReadingForm.vue', 'components/manual/OdometerReadingsPanel.vue',
  'components/quickadd/QuickOdometerField.vue', 'components/quickadd/QuickPendingCosts.vue', 'components/quickadd/QuickPhotoField.vue',
  'components/tires/TireCopyHistoryModal.vue', 'components/tires/TireDuplicateSessionModal.vue', 'components/tires/TireHistoryModal.vue',
  'components/tires/TireQuickRotationBar.vue', 'components/tires/TireSessionModal.vue', 'components/tires/TireWheelTile.vue',
  'components/vehicles/OwnershipWizardModal.vue', 'components/vehicles/VehicleCard.vue', 'components/vehicles/VehicleDataSourcesModal.vue',
  'components/vehicles/VehicleMembersModal.vue', 'views/ComparisonView.vue', 'views/ManualTrackingView.vue', 'views/TiresView.vue',
])
const FOUNDATIONS = ['components/StatusLabel.vue', 'components/CountBadge.vue', 'components/SegmentedControl.vue']

describe('colour rules', () => {
  it('keeps raw amber, emerald, red and sky classes out of new files', () => {
    const root = join(__dirname, '..')
    const found = sources(root)
      .map((file) => relative(root, file))
      .filter((rel) => !RAW_STATE_ALLOWLIST.has(rel) && RAW_STATE.test(readFileSync(join(root, rel), 'utf8')))
    expect(found).toEqual([])
  })

  it('lists only files that still use a raw state class', () => {
    const root = join(__dirname, '..')
    const stale = [...RAW_STATE_ALLOWLIST].filter((rel) => !RAW_STATE.test(readFileSync(join(root, rel), 'utf8')))
    expect(stale).toEqual([])
  })

  it('holds the shared state components to the rule and to the status tokens', () => {
    const root = join(__dirname, '..')
    for (const rel of FOUNDATIONS) {
      expect(RAW_STATE_ALLOWLIST.has(rel)).toBe(false)
      expect(readFileSync(join(root, rel), 'utf8')).not.toMatch(RAW_STATE)
    }
    expect(readFileSync(join(root, 'assets/main.css'), 'utf8')).toMatch(/--color-critical-400:\s*var\(--color-red-500\)/)
  })

  it('uses no raw accent hue: the colour of a cost is a theme token', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const rel = relative(root, file)
      const hits = readFileSync(file, 'utf8').match(FORBIDDEN)
      if (hits) found.push(`${rel}: ${[...new Set(hits)].join(' ')}`)
    }
    expect(found).toEqual([])
  })

  // The primary action is rose in every dialog; a solid green button reads as a status, not as the main action.
  it('keeps solid green fills for the confirmation dialog of a success', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const rel = relative(root, file)
      if (rel === 'components/ConfirmModal.vue') continue
      if (/<button[^>]*\bbg-success-[56]00\b(?!\/)/.test(readFileSync(file, 'utf8'))) found.push(rel)
    }
    expect(found).toEqual([])
  })

  // Sky is the colour of energy: a link is recognised by its underline or its chevron, not by a blue.
  it('does not colour a link sky', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const hits = readFileSync(file, 'utf8')
        .split('\n')
        .filter((line) => /<(?:a|router-link|RouterLink)\b/.test(line) && /text-(?:sky|info)-[34]00/.test(line))
      if (hits.length) found.push(relative(root, file))
    }
    expect(found).toEqual([])
  })

  // One figure per screen is read first, in the hero colour (rose-400); a file that sets two of them has two such figures.
  it('uses the hero colour once per file, and the token names the brand rose', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const hits = readFileSync(file, 'utf8').match(/\btext-hero\b|\btone="hero"/g) ?? []
      if (hits.length > 1) found.push(`${relative(root, file)}: ${hits.length}`)
    }
    expect(found).toEqual([])
    expect(readFileSync(join(root, 'assets/main.css'), 'utf8')).toMatch(/--color-hero:\s*var\(--color-rose-400\)/)
  })
})
