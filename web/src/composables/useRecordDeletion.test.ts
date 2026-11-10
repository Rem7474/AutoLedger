import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { t } from '@/i18n'
import { useRecordDeletion } from './useRecordDeletion'
import { UNDO_MS } from './useUndoableDelete'
import { useToast } from './useToast'

const { toasts } = useToast()
const api = vi.hoisted(() => ({ getCharge: vi.fn(), deleteCharge: vi.fn(), deleteFuelLog: vi.fn() }))
vi.mock('@/services/api', () => ({ api }))
const store = vi.hoisted(() => ({ activeVehicle: { id: 'v1' } as { id: string } | null }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))
const confirm = vi.hoisted(() => ({ showAlert: vi.fn() }))
vi.mock('@/composables/useConfirm', () => ({ useConfirm: () => confirm }))

describe('useRecordDeletion', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    store.activeVehicle = { id: 'v1' }
    api.deleteCharge.mockResolvedValue(undefined)
    api.deleteFuelLog.mockResolvedValue(undefined)
  })

  it('deletes fill-ups of a list under one snackbar, one after the other, and stops at the first refusal', async () => {
    api.deleteFuelLog.mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error('boom'))
    const list = ref<any[]>([{ id: 'f1' }, { id: 'f2' }, { id: 'f3' }])
    const reload = vi.fn()
    await useRecordDeletion().removeRecords({ kind: 'fuel_log', items: [...list.value], place: { list }, loaded: true, reload })
    expect(list.value).toEqual([])
    expect(toasts.value.at(-1)!.message).toBe(t('manual.fuelLogsPanel.fillUpsDeleted', { count: 3 }))
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(api.deleteFuelLog.mock.calls).toEqual([['v1', 'f1'], ['v1', 'f2']])
    expect(list.value.map((r) => r.id)).toEqual(['f2', 'f3'])
    expect(confirm.showAlert).toHaveBeenCalledWith('boom', t('shell.confirm.error'), 'danger')
    expect(reload).not.toHaveBeenCalled()
  })

  it('names a single fill-up and falls back on a generic message when the server gives none', async () => {
    api.deleteFuelLog.mockRejectedValueOnce(new Error(''))
    const list = ref<any[]>([{ id: 'f1' }])
    await useRecordDeletion().removeRecords({ kind: 'fuel_log', items: [{ id: 'f1' }], place: { list }, loaded: true, reload: vi.fn() })
    expect(toasts.value.at(-1)!.message).toBe(t('manual.fuelLogsPanel.fillUpDeleted'))
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(list.value.map((r) => r.id)).toEqual(['f1'])
    expect(confirm.showAlert).toHaveBeenCalledWith(t('manual.fuelLogsPanel.deleteFailed'), t('shell.confirm.error'), 'danger')
  })

  it('hides and restores a journal line by its own key, reading a charge before deleting it', async () => {
    api.getCharge.mockResolvedValue({ origin: 'MANUAL' })
    const place = { hide: vi.fn(), restore: vi.fn() }
    const reload = vi.fn()
    await useRecordDeletion().removeRecords({ kind: 'charge', items: [{ id: 'c1' }], place, reload })
    expect(api.getCharge).toHaveBeenCalledWith('v1', 'c1')
    expect(place.hide).toHaveBeenCalledTimes(1)
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(api.deleteCharge).toHaveBeenCalledWith('v1', 'c1')
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('refuses a synchronised charge from a journal line: nothing is hidden or sent', async () => {
    api.getCharge.mockResolvedValue({ origin: 'TESLAMATE' })
    const place = { hide: vi.fn(), restore: vi.fn() }
    await useRecordDeletion().removeRecords({ kind: 'charge', items: [{ id: 'c1' }], place, reload: vi.fn() })
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(place.hide).not.toHaveBeenCalled()
    expect(api.deleteCharge).not.toHaveBeenCalled()
    expect(confirm.showAlert).toHaveBeenCalledWith(t('expenses.chargesPanel.syncedCannotDelete'), t('shell.confirm.error'), 'warning')
  })

  it('does not delete a charge it cannot read, and reads no fill-up before deleting it', async () => {
    api.getCharge.mockRejectedValueOnce(new Error('gone'))
    const place = { hide: vi.fn(), restore: vi.fn() }
    await useRecordDeletion().removeRecords({ kind: 'charge', items: [{ id: 'c1' }], place, reload: vi.fn() })
    expect(place.hide).not.toHaveBeenCalled()
    await useRecordDeletion().removeRecords({ kind: 'fuel_log', items: [{ id: 'f1' }], place, reload: vi.fn() })
    expect(api.getCharge).toHaveBeenCalledTimes(1)
    expect(place.hide).toHaveBeenCalledTimes(1)
  })

  it('does nothing without a vehicle or without rows', async () => {
    const place = { hide: vi.fn(), restore: vi.fn() }
    await useRecordDeletion().removeRecords({ kind: 'fuel_log', items: [], place, reload: vi.fn() })
    store.activeVehicle = null
    await useRecordDeletion().removeRecords({ kind: 'fuel_log', items: [{ id: 'f1' }], place, reload: vi.fn() })
    expect(place.hide).not.toHaveBeenCalled()
  })
})
