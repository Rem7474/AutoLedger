import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useTireHistory } from './useTireHistory'

const api = vi.hoisted(() => ({
  getTireHistory: vi.fn(),
  deleteTire: vi.fn(),
  deleteTireSession: vi.fn(),
  deleteTireLog: vi.fn(),
}))
vi.mock('@/services/api', () => ({ api }))
vi.mock('@/i18n', () => ({ t: (k: string) => k, intlLocale: () => 'en' }))
vi.mock('@/stores/vehicle', () => ({
  useVehicleStore: () => ({ activeVehicle: { id: 'v1', current_odometer: 1234.6 } }),
}))
const confirm = vi.hoisted(() => ({ showConfirm: vi.fn(), showAlert: vi.fn() }))
vi.mock('@/composables/useConfirm', () => ({ useConfirm: () => confirm }))

function setup() {
  const tires = ref<any[]>([{ tire: { id: 't1' } }])
  const loadTires = vi.fn().mockResolvedValue(undefined)
  const selectedTireIds = ref<string[]>(['t1', 't2'])
  return { tires, loadTires, selectedTireIds, h: useTireHistory({ tires, loadTires, selectedTireIds }) }
}

describe('useTireHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    confirm.showConfirm.mockResolvedValue(true)
    api.getTireHistory.mockResolvedValue({ tire: { id: 't1' }, stats: { s: 1 }, sessions: [{ id: 's1' }], logs: null })
  })

  it('opens the history with the sessions and readings of the tire', async () => {
    const { h } = setup()
    await h.openHistoryModal({ tire: { id: 't1' } })
    expect(h.showHistoryModal.value).toBe(true)
    expect(h.tireSessions.value).toEqual([{ id: 's1' }])
    expect(h.tireLogs.value).toEqual([])
    expect(h.selectedTireStats.value).toEqual({ s: 1 })
  })

  it('alerts and keeps the modal closed when the history cannot be loaded', async () => {
    api.getTireHistory.mockRejectedValue(new Error('boom'))
    const { h } = setup()
    await h.openHistoryModal({ tire: { id: 't1' } })
    expect(h.showHistoryModal.value).toBe(false)
    expect(confirm.showAlert).toHaveBeenCalled()
  })

  it('opens the history of a timeline tire only when it exists', async () => {
    const { h } = setup()
    h.openTimelineTire('nope')
    expect(api.getTireHistory).not.toHaveBeenCalled()
    h.openTimelineTire('t1')
    expect(api.getTireHistory).toHaveBeenCalledWith('v1', 't1')
  })

  it('reloads the list and the open history after a change', async () => {
    const { h, loadTires } = setup()
    await h.refreshAfterHistoryChange()
    expect(loadTires).toHaveBeenCalledTimes(1)
    expect(api.getTireHistory).not.toHaveBeenCalled()
    await h.openHistoryModal({ tire: { id: 't1' } })
    await h.refreshAfterHistoryChange()
    expect(api.getTireHistory).toHaveBeenCalledTimes(2)
  })

  it('deletes a tire after confirmation and unselects it', async () => {
    const { h, loadTires, selectedTireIds } = setup()
    await h.handleDeleteTire({ id: 't1' })
    expect(api.deleteTire).toHaveBeenCalledWith('v1', 't1')
    expect(selectedTireIds.value).toEqual(['t2'])
    expect(loadTires).toHaveBeenCalled()
  })

  it('does nothing when the deletion is declined, and alerts when it fails', async () => {
    const { h } = setup()
    confirm.showConfirm.mockResolvedValue(false)
    await h.handleDeleteTire({ id: 't1' })
    expect(api.deleteTire).not.toHaveBeenCalled()
    confirm.showConfirm.mockResolvedValue(true)
    api.deleteTire.mockRejectedValue(new Error('x'))
    await h.handleDeleteTire({ id: 't1' })
    expect(confirm.showAlert).toHaveBeenCalled()
  })

  it('deletes sessions and readings of the selected tire and reloads', async () => {
    const { h, loadTires } = setup()
    await h.handleDeleteSession({ id: 's1' })
    expect(api.deleteTireSession).not.toHaveBeenCalled() // no tire selected yet
    await h.openHistoryModal({ tire: { id: 't1' } })
    await h.handleDeleteSession({ id: 's1' })
    expect(api.deleteTireSession).toHaveBeenCalledWith('v1', 't1', 's1')
    await h.handleDeleteLog({ id: 'l1', depth_mm: 5, date: '2026-01-02T00:00:00Z' })
    expect(api.deleteTireLog).toHaveBeenCalledWith('v1', 't1', 'l1')
    expect(loadTires).toHaveBeenCalledTimes(2)
    api.deleteTireLog.mockRejectedValue(new Error('x'))
    await h.handleDeleteLog({ id: 'l1', depth_mm: 5, date: '2026-01-02T00:00:00Z' })
    expect(confirm.showAlert).toHaveBeenCalled()
  })

  it('prefills the reading form from the tire, or from the reading being edited', () => {
    const { h } = setup()
    h.openLogModal({ tire: { id: 't1' }, current_depth_mm: 4.2 })
    expect(h.editingLogId.value).toBeNull()
    expect(h.logInitialForm.value).toMatchObject({ depth_mm: 4.2, odometer: 1235, notes: '' })
    h.editLog({ id: 'l1', depth_mm: 3, odometer: 900.4, notes: null, date: '2026-03-04T10:00:00Z' })
    expect(h.editingLogId.value).toBe('l1')
    expect(h.logInitialForm.value).toMatchObject({ depth_mm: 3, odometer: 900, notes: '' })
    expect(h.showLogModal.value).toBe(true)
  })

  it('only pastes a session that was copied', () => {
    const { h } = setup()
    h.pasteSessionToCurrentTire()
    expect(h.showSessionModal.value).toBe(false)
    h.openAddSessionModal()
    expect(h.editingSessionId.value).toBeNull()
    expect(h.showSessionModal.value).toBe(true)
    h.openEditSessionModal({ id: 's9', position: 'FL', mounted_date: '2026-01-01T00:00:00Z' })
    expect(h.editingSessionId.value).toBe('s9')
    h.openDuplicateSessionModal({ id: 's9' })
    expect(h.showDuplicateSessionModal.value).toBe(true)
  })

  it('unselects a disposed tire and closes the history', async () => {
    const { h, loadTires, selectedTireIds } = setup()
    h.showHistoryModal.value = true
    await h.onTireDisposed('t2')
    expect(h.showHistoryModal.value).toBe(false)
    expect(selectedTireIds.value).toEqual(['t1'])
    expect(loadTires).toHaveBeenCalled()
  })
})
