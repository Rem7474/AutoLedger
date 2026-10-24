import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useCarpoolEditor } from './useCarpoolEditor'

const api = vi.hoisted(() => ({ getCarpools: vi.fn() }))
vi.mock('@/services/api', () => ({ api }))
vi.mock('@/i18n', () => ({ t: (k: string) => k }))
const store = vi.hoisted(() => ({ activeVehicle: { id: 'v1' } as { id: string } | null }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))

describe('useCarpoolEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    store.activeVehicle = { id: 'v1' }
  })

  it('opens the editor on the given drives with a new open token and loads the names already used', async () => {
    api.getCarpools.mockResolvedValue({ trips: [{ date: '2026-01-01', passengers: [{ passenger_name: 'Alice' }] }] })
    const e = useCarpoolEditor()
    await e.openCarpoolEditor({ driveIds: ['d1'] })
    expect(e.showCarpoolEditor.value).toBe(true)
    expect(e.carpoolCreateOptions.value).toEqual({ driveIds: ['d1'] })
    expect(e.carpoolOpenToken.value).toBe(1)
    expect(api.getCarpools).toHaveBeenCalledWith('v1')
    expect(e.recentCarpoolNames.value).toEqual(['Alice'])
    await e.openCarpoolEditor({ tripGroupId: 'g1' })
    expect(e.carpoolOpenToken.value).toBe(2)
  })

  it('still opens when the names cannot be loaded', async () => {
    api.getCarpools.mockRejectedValue(new Error('down'))
    const e = useCarpoolEditor()
    await e.openCarpoolEditor({ driveIds: ['d1'] })
    expect(e.showCarpoolEditor.value).toBe(true)
    expect(e.recentCarpoolNames.value).toEqual([])
  })

  it('does not look the names up without a vehicle', async () => {
    store.activeVehicle = null
    const e = useCarpoolEditor()
    await e.openCarpoolEditor({ driveIds: ['d1'] })
    expect(api.getCarpools).not.toHaveBeenCalled()
  })
})
