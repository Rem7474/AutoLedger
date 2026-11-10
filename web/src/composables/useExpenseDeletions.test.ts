import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { t } from '@/i18n'
import { useExpenseDeletions } from './useExpenseDeletions'
import { UNDO_MS } from './useUndoableDelete'
import { useToast } from './useToast'

const { toasts } = useToast()

const api = vi.hoisted(() => ({ deleteCharge: vi.fn(), deleteDocument: vi.fn() }))
vi.mock('@/services/api', () => ({ api }))
const store = vi.hoisted(() => ({ activeVehicle: { id: 'v1' } as { id: string } | null, currency: 'EUR' }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => store }))
const confirm = vi.hoisted(() => ({ showConfirm: vi.fn(), showAlert: vi.fn() }))
vi.mock('@/composables/useConfirm', () => ({ useConfirm: () => confirm }))

const setup = () => {
  const documents = ref<any[]>([{ id: 'doc1', filename: 'a.pdf' }, { id: 'doc2', filename: 'b.pdf' }])
  const lists = {
    charges: ref<any[]>([{ id: 'x1' }, { id: 'c1' }]),
  }
  const reload = vi.fn().mockResolvedValue(undefined)
  return { documents, lists, reload, h: useExpenseDeletions(documents, reload, lists) }
}

describe('useExpenseDeletions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.useFakeTimers()
    store.activeVehicle = { id: 'v1' }
    confirm.showConfirm.mockResolvedValue(true)
    for (const fn of Object.values(api)) fn.mockResolvedValue(undefined)
  })

  // One row goes through its own wording, a selection through one confirmation for all of them
  const kinds = [
    ['charges', 'handleDeleteCharges', 'deleteCharge'],
  ] as const

  const listOf = { charges: 'charges' } as const

  for (const [label, handler, call] of kinds) {
    it(`takes a single one of the ${label} off the list at once and deletes it once the time to undo is over`, async () => {
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      await h[handler]([{ id: 'x1', amount: '12', kwh_added: 30, description: 'Oil' }])
      expect(confirm.showConfirm).not.toHaveBeenCalled()
      expect(list.value.map((r) => r.id)).not.toContain('x1')
      expect(api[call]).not.toHaveBeenCalled()
      await vi.advanceTimersByTimeAsync(UNDO_MS)
      expect(api[call]).toHaveBeenCalledWith('v1', 'x1')
      expect(reload).toHaveBeenCalledTimes(1)
    })

    it(`puts a single one of the ${label} back where it was when the deletion is taken back`, async () => {
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      const before = list.value.map((r) => r.id)
      await h[handler]([{ id: 'x1' }])
      toasts.value.at(-1)!.action!.run()
      expect(list.value.map((r) => r.id)).toEqual(before)
      await vi.advanceTimersByTimeAsync(UNDO_MS * 2)
      expect(api[call]).not.toHaveBeenCalled()
      expect(reload).not.toHaveBeenCalled()
    })

    it(`takes a selection of ${label} off the list at once under one snackbar, deletes them one by one once the time is over and reloads`, async () => {
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      list.value = [{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }, { id: 'keep' }]
      await h[handler]([{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }])
      expect(confirm.showConfirm).not.toHaveBeenCalled()
      expect(list.value.map((r) => r.id)).toEqual(['keep'])
      expect(toasts.value).toHaveLength(1)
      expect(api[call]).not.toHaveBeenCalled()
      await vi.advanceTimersByTimeAsync(UNDO_MS)
      expect(api[call].mock.calls).toEqual([['v1', 'x1'], ['v1', 'x2'], ['v1', 'x3']])
      expect(reload).toHaveBeenCalledTimes(1)
    })

    it(`puts a selection of ${label} back when undone`, async () => {
      const { h, lists } = setup()
      const list = lists[listOf[label]]
      list.value = [{ id: 'x1' }, { id: 'x2' }, { id: 'keep' }]
      await h[handler]([{ id: 'x1' }, { id: 'x2' }])
      toasts.value.at(-1)!.action!.run()
      expect(list.value.map((r) => r.id)).toEqual(['x1', 'x2', 'keep'])
      await vi.advanceTimersByTimeAsync(UNDO_MS * 2)
      expect(api[call]).not.toHaveBeenCalled()
    })

    it(`brings a single one of the ${label} back and reports it when the server refuses the delete`, async () => {
      api[call].mockRejectedValueOnce(new Error('boom'))
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      await h[handler]([{ id: 'x1' }])
      await vi.advanceTimersByTimeAsync(UNDO_MS)
      expect(list.value.map((r) => r.id)).toContain('x1')
      expect(confirm.showAlert).toHaveBeenCalledWith(t('common.deleteError', { message: 'boom' }), t('shell.confirm.error'), 'danger')
      expect(reload).not.toHaveBeenCalled()
    })

    it(`stops at the failure of a selection of ${label} and brings back the rows the server did not delete`, async () => {
      api[call].mockResolvedValueOnce(undefined).mockRejectedValueOnce(new Error('boom'))
      const { h, reload, lists } = setup()
      const list = lists[listOf[label]]
      list.value = [{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }]
      await h[handler]([{ id: 'x1' }, { id: 'x2' }, { id: 'x3' }])
      await vi.advanceTimersByTimeAsync(UNDO_MS)
      expect(api[call]).toHaveBeenCalledTimes(2)
      expect(list.value.map((r) => r.id)).toEqual(['x2', 'x3'])
      expect(confirm.showAlert).toHaveBeenCalledWith(t('common.deleteError', { message: 'boom' }), t('shell.confirm.error'), 'danger')
      expect(reload).not.toHaveBeenCalled()
    })
  }

  it('does nothing without an active vehicle or an empty selection', async () => {
    const { h, reload } = setup()
    await h.handleDeleteCharges([])
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    store.activeVehicle = null
    await h.handleDeleteCharge({ id: 'c1', kwh_added: 1 })
    await h.handleDeleteDocument({ id: 'doc1', filename: 'a.pdf' } as any)
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    expect(reload).not.toHaveBeenCalled()
  })

  it('hides a deleted receipt at once, deletes it once the time to undo is over and reloads', async () => {
    const { h, documents, reload } = setup()
    h.handleDeleteDocument(documents.value[0])
    expect(documents.value.map((d) => d.id)).toEqual(['doc2'])
    expect(api.deleteDocument).not.toHaveBeenCalled()
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(api.deleteDocument).toHaveBeenCalledWith('v1', 'doc1')
    expect(confirm.showConfirm).not.toHaveBeenCalled()
    expect(reload).toHaveBeenCalledTimes(1)
  })

  it('brings the receipt back when undone or when its delete fails', async () => {
    const { h, documents } = setup()
    const first = documents.value[0]
    h.handleDeleteDocument(first)
    toasts.value.at(-1)!.action!.run()
    expect(documents.value.map((d) => d.id)).toEqual(['doc1', 'doc2'])
    api.deleteDocument.mockRejectedValueOnce(new Error('nope'))
    h.handleDeleteDocument(first)
    await vi.advanceTimersByTimeAsync(UNDO_MS)
    expect(documents.value.map((d) => d.id)).toEqual(['doc1', 'doc2'])
    expect(confirm.showAlert).toHaveBeenCalledWith(t('common.errorWithMessage', { message: 'nope' }), t('shell.confirm.error'), 'danger')
  })
})
