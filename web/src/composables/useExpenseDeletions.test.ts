import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { t } from '@/i18n'
import { useExpenseDeletions } from './useExpenseDeletions'

const api = vi.hoisted(() => ({ deleteDriveExpense: vi.fn(), deleteMaintenance: vi.fn(), deleteCharge: vi.fn(), deleteDocument: vi.fn() }))
vi.mock('@/services/api', () => ({ api }))
const store = vi.hoisted(() => ({ activeVehicle: { id: 'v1' } as { id: string } | null, currency: 'EUR' }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))
const confirm = vi.hoisted(() => ({ showConfirm: vi.fn(), showAlert: vi.fn() }))
vi.mock('@/composables/useConfirm', () => ({ useConfirm: () => confirm }))

const setup = () => {
  const documents = ref<any[]>([{ id: 'doc1', filename: 'a.pdf' }, { id: 'doc2', filename: 'b.pdf' }])
  const reload = vi.fn().mockResolvedValue(undefined)
  return { documents, reload, h: useExpenseDeletions(documents, reload) }
}

describe('useExpenseDeletions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    store.activeVehicle = { id: 'v1' }
    confirm.showConfirm.mockResolvedValue(true)
    for (const fn of Object.values(api)) fn.mockResolvedValue(undefined)
  })

  // One row goes through its own wording, a selection through one confirmation for all of them
  const kinds = [
    ['tolls', 'handleDeleteTolls', 'deleteDriveExpense'],
    ['maintenance entries', 'handleDeleteMaints', 'deleteMaintenance'],
    ['charges', 'handleDeleteCharges', 'deleteCharge'],
  ] as const

  for (const [label, handler, call] of kinds) {
    it(`deletes a single one of the ${label} after its own confirmation, then reloads`, async () => {
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1', amount: '12', kwh_added: 30, description: 'Oil' }])
      expect(confirm.showConfirm).toHaveBeenCalledTimes(1)
      expect(confirm.showConfirm.mock.calls[0][0].message).not.toBe(t('expenses.expensesView.deleteManyMessage', { count: 1 }))
      expect(api[call]).toHaveBeenCalledWith('v1', 'x1')
      expect(reload).toHaveBeenCalledTimes(1)
    })

    it(`asks once for a selection of ${label}, deletes them one by one and reloads`, async () => {
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }])
      expect(confirm.showConfirm).toHaveBeenCalledTimes(1)
      expect(confirm.showConfirm.mock.calls[0][0].message).toBe(t('expenses.expensesView.deleteManyMessage', { count: 3 }))
      expect(api[call].mock.calls).toEqual([['v1', 'x1'], ['v1', 'x2'], ['v1', 'x3']])
      expect(reload).toHaveBeenCalledTimes(1)
    })

    it(`deletes nothing from a selection of ${label} that is not confirmed`, async () => {
      confirm.showConfirm.mockResolvedValue(false)
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1' }, { id: 'x2' }])
      expect(api[call]).not.toHaveBeenCalled()
      expect(reload).not.toHaveBeenCalled()
    })

    it(`reports a failure of one single delete of the ${label} without reloading`, async () => {
      api[call].mockRejectedValueOnce(new Error('boom'))
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1', amount: '1', kwh_added: 1 }])
      expect(confirm.showAlert).toHaveBeenCalledWith(t('common.deleteError', { message: 'boom' }), t('shell.confirm.error'), 'danger')
      expect(reload).not.toHaveBeenCalled()
    })

    it(`reloads a selection of ${label} even when one delete fails, and stops at the failure`, async () => {
      api[call].mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error('boom'))
      const { h, reload } = setup()
      await h[handler]([{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }])
      expect(api[call]).toHaveBeenCalledTimes(2)
      expect(confirm.showAlert).toHaveBeenCalledWith(t('common.deleteError', { message: 'boom' }), t('shell.confirm.error'), 'danger')
      expect(reload).toHaveBeenCalledTimes(1)
    })
  }

  it('does nothing without an active vehicle or an empty selection', async () => {
    const { h, reload } = setup()
    await h.handleDeleteCharges([])
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    store.activeVehicle = null
    await h.handleDeleteTolls([{ id: 'x1' }, { id: 'x2' }])
    await h.handleDeleteMaints([{ id: 'x1' }])
    await h.handleDeleteCharge({ id: 'c1', kwh_added: 1 })
    await h.handleDeleteToll({ id: 't1', amount: '1' })
    await h.handleDeleteMaint({ id: 'm1', amount: '1' })
    await h.handleDeleteDocument({ id: 'doc1', filename: 'a.pdf' } as any)
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    expect(reload).not.toHaveBeenCalled()
  })

  it('formats a single amount in the row currency, falling back to the vehicle one', async () => {
    const { h } = setup()
    await h.handleDeleteToll({ id: 't1', amount: '5', currency: 'USD' })
    await h.handleDeleteToll({ id: 't2', amount: '5' })
    const [usd, eur] = confirm.showConfirm.mock.calls.map((c) => c[0].message as string)
    expect(usd).not.toBe(eur)
  })

  it('removes a deleted receipt from the list, confirms it and reloads', async () => {
    const { h, documents, reload } = setup()
    await h.handleDeleteDocument(documents.value[0])
    expect(api.deleteDocument).toHaveBeenCalledWith('v1', 'doc1')
    expect(documents.value.map((d) => d.id)).toEqual(['doc2'])
    expect(confirm.showAlert).toHaveBeenCalledWith(t('expenses.expensesView.receiptDeleted'), t('common.success'), 'success')
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('keeps the receipt and reports the error when its delete fails, and does not delete when declined', async () => {
    const { h, documents } = setup()
    confirm.showConfirm.mockResolvedValueOnce(false)
    await h.handleDeleteDocument(documents.value[0])
    expect(api.deleteDocument).not.toHaveBeenCalled()
    api.deleteDocument.mockRejectedValueOnce(new Error('nope'))
    await h.handleDeleteDocument(documents.value[0])
    expect(documents.value).toHaveLength(2)
    expect(confirm.showAlert).toHaveBeenCalledWith(t('common.errorWithMessage', { message: 'nope' }), t('shell.confirm.error'), 'danger')
  })
})
