import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { CostEntryLine } from '@/services/api'
import { useCostEntryActions } from './useCostEntryActions'

const api = vi.hoisted(() => ({
  getCharge: vi.fn(),
  getFuelLogs: vi.fn(),
  deleteFuelLog: vi.fn(),
  getDriveExpense: vi.fn(),
  getMaintenanceExpense: vi.fn(),
  getMaintenance: vi.fn(),
  getDocuments: vi.fn(),
  deleteCharge: vi.fn(),
  deleteDriveExpense: vi.fn(),
  deleteMaintenance: vi.fn(),
}))
vi.mock('@/services/api', () => ({ api }))
vi.mock('@/i18n', () => ({ t: (k: string) => k }))
const store = vi.hoisted(() => ({ activeVehicle: { id: 'v1' } as { id: string } | null }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))
const showAlert = vi.hoisted(() => vi.fn())
vi.mock('@/composables/useConfirm', () => ({ useConfirm: () => ({ showAlert }) }))
const deleteWithUndo = vi.hoisted(() => vi.fn())
vi.mock('@/composables/useUndoableDelete', () => ({ deleteWithUndo, deleteRowsWithUndo: vi.fn() }))

const entry = (source: string, id = 'r1'): CostEntryLine => ({
  date: '2026-03-10', category: 'toll', kind: 'toll', amount: { amount: '5.00', currency: 'EUR' }, note: null, place: null,
  place_category: null, kwh: null, litres: null, recurring: false, has_receipt: null, source, source_id: id,
})
const hooks = () => ({ hide: vi.fn(), restore: vi.fn(), reload: vi.fn() })

describe('useCostEntryActions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    store.activeVehicle = { id: 'v1' }
    api.getDocuments.mockResolvedValue([{ id: 'doc1' }])
  })

  it('reads the record a line comes from and opens the form of its kind, with the receipts', async () => {
    api.getDriveExpense.mockResolvedValue({ id: 'r1' })
    api.getMaintenanceExpense.mockResolvedValue({ id: 'm1' })
    api.getCharge.mockResolvedValue({ id: 'c1' })
    const a = useCostEntryActions(hooks())
    await a.open(entry('drive_expenses'))
    expect(api.getDriveExpense).toHaveBeenCalledWith('v1', 'r1')
    expect([a.showToll.value, a.showMaintenance.value, a.showCharge.value]).toEqual([true, false, false])
    expect(a.editing.value).toEqual({ id: 'r1' })
    expect(a.documents.value).toEqual([{ id: 'doc1' }])
    await a.open(entry('maintenance_expenses', 'm1'))
    expect(a.showMaintenance.value).toBe(true)
    await a.open(entry('charge_logs', 'c1'))
    expect(a.showCharge.value).toBe(true)
    expect(a.editing.value).toEqual({ id: 'c1' })
  })

  it('opens nothing for a line without an editable record, or without a vehicle', async () => {
    const a = useCostEntryActions(hooks())
    await a.open(entry('tires'))
    store.activeVehicle = null
    await a.open(entry('drive_expenses'))
    expect(api.getDriveExpense).not.toHaveBeenCalled()
    expect(a.showToll.value).toBe(false)
  })

  it('says so when the record cannot be read, and opens no form', async () => {
    api.getDriveExpense.mockRejectedValue(new Error('gone'))
    const a = useCostEntryActions(hooks())
    await a.open(entry('drive_expenses'))
    expect(a.showToll.value).toBe(false)
    expect(showAlert).toHaveBeenCalledTimes(1)
  })

  it('keeps the list of receipts empty when it cannot be read, and still opens the form', async () => {
    api.getDocuments.mockRejectedValue(new Error('down'))
    api.getDriveExpense.mockResolvedValue({ id: 'r1' })
    vi.spyOn(console, 'error').mockImplementation(() => {})
    const a = useCostEntryActions(hooks())
    await a.open(entry('drive_expenses'))
    expect(a.showToll.value).toBe(true)
    expect(a.documents.value).toEqual([])
  })

  it('opens the maintenance form empty, with the receipts and the earlier entries that suggest and close a revision', async () => {
    api.getMaintenance.mockResolvedValue([{ id: 'm1' }])
    const a = useCostEntryActions(hooks())
    a.editing.value = { id: 'old' }
    await a.addMaintenance()
    expect(a.showMaintenance.value).toBe(true)
    expect(a.editing.value).toBeNull()
    expect(a.maintenanceExpenses.value).toEqual([{ id: 'm1' }])
    expect(api.getDocuments).toHaveBeenCalledWith('v1')
  })

  it('still opens the empty maintenance form when the earlier entries cannot be read, and opens nothing without a vehicle', async () => {
    api.getMaintenance.mockRejectedValue(new Error('down'))
    const a = useCostEntryActions(hooks())
    await a.addMaintenance()
    expect(a.showMaintenance.value).toBe(true)
    expect(a.maintenanceExpenses.value).toEqual([])
    const b = useCostEntryActions(hooks())
    store.activeVehicle = null
    await b.addMaintenance()
    expect(b.showMaintenance.value).toBe(false)
  })

  it('adds a receipt uploaded from a form to the list', () => {
    const a = useCostEntryActions(hooks())
    a.onDocumentAdded({ id: 'd2' } as any)
    expect(a.documents.value).toEqual([{ id: 'd2' }])
  })

  it('deletes with a way back: the line is hidden, put back on undo, sent to the server on commit, and the list read again', async () => {
    const h = hooks()
    const a = useCostEntryActions(h)
    await a.remove(entry('drive_expenses', 'e1'), 'row1')
    const d = deleteWithUndo.mock.calls[0][0]
    expect(d.message).toBe('expenses.expensesView.costDeleted')
    d.hide()
    d.restore()
    expect([h.hide, h.restore].map((f) => f.mock.calls[0][0])).toEqual(['row1', 'row1'])
    await d.commit()
    expect(api.deleteDriveExpense).toHaveBeenCalledWith('v1', 'e1')
    d.committed()
    expect(h.reload).toHaveBeenCalled()
    d.onError(new Error('refused'))
    expect(showAlert).toHaveBeenCalled()
  })

  it('names the charge and the maintenance when it deletes them', async () => {
    api.getCharge.mockResolvedValue({ origin: 'MANUAL' })
    const a = useCostEntryActions(hooks())
    await a.remove(entry('charge_logs', 'c1'), 'row1')
    const charge = deleteWithUndo.mock.calls[0][0]
    expect(charge.message).toBe('expenses.expensesView.chargeDeleted')
    await charge.commit()
    expect(api.deleteCharge).toHaveBeenCalledWith('v1', 'c1')
    await a.remove(entry('maintenance_expenses', 'm1'), 'row2')
    await deleteWithUndo.mock.calls[1][0].commit()
    expect(api.deleteMaintenance).toHaveBeenCalledWith('v1', 'm1')
  })

  it('does not delete a charge reported by TeslaMate: the next synchronisation would bring it back', async () => {
    api.getCharge.mockResolvedValue({ origin: 'TESLAMATE' })
    const a = useCostEntryActions(hooks())
    await a.remove(entry('charge_logs', 'c1'), 'row1')
    expect(deleteWithUndo).not.toHaveBeenCalled()
    expect(showAlert).toHaveBeenCalledWith('expenses.chargesPanel.syncedCannotDelete', 'shell.confirm.error', 'warning')
  })

  it('does not delete a charge it cannot read, nor a line without a record or without a vehicle', async () => {
    api.getCharge.mockRejectedValue(new Error('gone'))
    const a = useCostEntryActions(hooks())
    await a.remove(entry('charge_logs', 'c1'), 'row1')
    await a.remove(entry('tires', 't1'), 'row2')
    store.activeVehicle = null
    await a.remove(entry('drive_expenses', 'e1'), 'row3')
    expect(deleteWithUndo).not.toHaveBeenCalled()
    expect(showAlert).toHaveBeenCalledTimes(1)
  })

  it('opens a fill-up in its sheet, found by id among the fill-ups of the vehicle', async () => {
    api.getFuelLogs.mockResolvedValue({ logs: [{ id: 'f0' }, { id: 'f1', version: 2 }] })
    const a = useCostEntryActions(hooks())
    await a.open(entry('fuel_logs', 'f1'))
    expect(a.showFuel.value).toBe(true)
    expect(a.editing.value).toEqual({ id: 'f1', version: 2 })
    expect([a.showToll.value, a.showMaintenance.value, a.showCharge.value]).toEqual([false, false, false])
  })

  it('says so when the fill-up is gone, and opens nothing', async () => {
    api.getFuelLogs.mockResolvedValue({ logs: [{ id: 'f0' }] })
    const a = useCostEntryActions(hooks())
    await a.open(entry('fuel_logs', 'f1'))
    expect(a.showFuel.value).toBe(false)
    expect(showAlert).toHaveBeenCalledTimes(1)
  })

  it('deletes a fill-up with a way back, from a swipe or from the sheet, and reads the journal again', async () => {
    api.getFuelLogs.mockResolvedValue({ logs: [{ id: 'f1', version: 2 }] })
    const h = hooks()
    const a = useCostEntryActions(h)
    await a.remove(entry('fuel_logs', 'f1'), 'row1')
    const swipe = deleteWithUndo.mock.calls[0][0]
    expect(swipe.message).toBe('manual.fuelLogsPanel.fillUpDeleted')
    await swipe.commit()
    expect(api.deleteFuelLog).toHaveBeenCalledWith('v1', 'f1')
    swipe.committed()
    expect(h.reload).toHaveBeenCalled()

    await a.open(entry('fuel_logs', 'f1'))
    await a.removeOpenedFuel({ id: 'f1' })
    expect(a.showFuel.value).toBe(false)
    const sheet = deleteWithUndo.mock.calls[1][0]
    sheet.hide()
    sheet.restore()
    expect(h.hide.mock.calls[0][0]).toBe(h.restore.mock.calls[0][0])
  })
})
