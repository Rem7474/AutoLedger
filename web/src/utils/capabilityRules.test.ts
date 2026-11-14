import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// The interface asks what a vehicle can do (`vehicle.capabilities`, computed by the server), never what powertrain it
// has or whether TeslaMate is connected. The helpers that compare a powertrain or test the connection stay in
// utils/vehicles.ts for the forms that describe a vehicle which does not exist yet; a view, component, composable or
// store does not call them.
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !/\.test\.ts$/.test(name) ? [path] : []
  })
}

// A test of the connection or of "only fuel / only battery", and the powertrain helpers: allowed where the reason says.
const POWERTRAIN_TESTS = /\b(hasTeslaMate|fuelOnly|electricOnly|isFuelOnly|isElectricOnly|canCharge|canRefuel|canLinkTeslaMate)\b/
const STORE_FLAGS = /\b[a-zA-Z]*[sS]tore\.(canCharge|canRefuel|hasTeslaMate|fuelOnly|electricOnly)\b/

const ALLOWED: Record<string, string> = {
  // the vehicle form and the onboarding describe a vehicle that does not exist yet: they read the powertrain being chosen
  'components/vehicles/VehicleFormModal.vue': 'draft powertrain of the vehicle form',
  'utils/onboarding.ts': 'draft powertrain of the onboarding',
  'utils/dataSources.ts': 'CSV templates offered by the onboarding before the vehicle exists',
  'utils/capabilities.ts': 'fallback for an answer without capabilities',
  'utils/vehicles.ts': 'the powertrain helpers themselves',
  // not yet moved to the capabilities of the vehicle they show
  'components/dashboard/DataQualityCard.vue': 'reads the powertrain of the TCO answer',
  'components/dashboard/TcoDetailsList.vue': 'reads the powertrain of the TCO answer',
  'components/manual/OdometerReadingsPanel.vue': 'reads the powertrain of the vehicle prop',
  'components/vehicles/VehicleCard.vue': 'reads the powertrain of the listed vehicle',
  'components/vehicles/VehicleDataSourcesModal.vue': 'reads the powertrain of the vehicle prop',
  'views/ComparisonView.vue': 'reads the powertrain of the form vehicle',
  'utils/comparisonSide.ts': 'reads the powertrain of a scenario vehicle',
}

describe('capability rules', () => {
  const root = join(__dirname, '..')
  const scanned = ['components', 'views', 'composables', 'stores', 'utils']
    .flatMap((dir) => sources(join(root, dir)))
    .map((file) => relative(root, file))
  const read = (rel: string) => readFileSync(join(root, rel), 'utf8')

  it('reads no powertrain or connection test outside the closed list', () => {
    const found = scanned.filter((rel) => !(rel in ALLOWED) && POWERTRAIN_TESTS.test(read(rel)))
    expect(found).toEqual([])
  })

  it('lists only files that still need an exception', () => {
    const stale = Object.keys(ALLOWED).filter((rel) => !POWERTRAIN_TESTS.test(read(rel)))
    expect(stale).toEqual([])
  })

  it('has no store flag standing for a powertrain or a connection', () => {
    const found = scanned.filter((rel) => STORE_FLAGS.test(read(rel)))
    expect(found).toEqual([])
  })
})
