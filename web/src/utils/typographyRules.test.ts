import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// Monospace is for what the reader copies or compares character by character: a token, a version, a VIN, a tyre size, a CSV
// column, a file extension. A figure is never monospace (`tabular-nums` already aligns digits).
const MONOSPACE_FILES = [
  'components/AppDropzone.vue',
  'components/CSVImportModal.vue',
  'components/Navigation.vue',
  'components/account/ApiTokensSection.vue',
  'components/account/TokenCallLog.vue',
  'components/tires/TireAddModal.vue',
  'components/tires/TireDisposedCard.vue',
  'components/tires/TireHistoryModal.vue',
  'components/tires/TireOdometerTimeline.vue',
  'components/tires/TireStorageCard.vue',
  'components/tires/TireWheelCard.vue',
  'components/vehicles/VehicleCard.vue',
]

function vueFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return vueFiles(path)
    return name.endsWith('.vue') ? [path] : []
  })
}

describe('typography rules', () => {
  it('uses font-mono only in the files that show a copyable identifier', () => {
    const root = join(__dirname, '..')
    const found = vueFiles(root)
      .map((file) => relative(root, file))
      .filter((rel) => /\bfont-mono\b/.test(readFileSync(join(root, rel), 'utf8')) && !MONOSPACE_FILES.includes(rel))
    expect(found).toEqual([])
  })

  it('keeps every allowed file in use', () => {
    const root = join(__dirname, '..')
    const unused = MONOSPACE_FILES.filter((rel) => !/\bfont-mono\b/.test(readFileSync(join(root, rel), 'utf8')))
    expect(unused).toEqual([])
  })
})
