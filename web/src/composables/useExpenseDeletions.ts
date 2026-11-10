import type { Ref } from 'vue'
import { t } from '@/i18n'
import { api, type ExpenseDocumentHeader } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { deleteRowsWithUndo, deleteWithUndo } from '@/composables/useUndoableDelete'

export interface ExpenseLists {
  charges: Ref<any[]>
}

// Delete handlers of the expenses page; `reload` refreshes the list on screen afterwards. A row, a selection and a
// receipt are deleted without a question and can be taken back for a few seconds.
export function useExpenseDeletions(documents: Ref<ExpenseDocumentHeader[]>, reload: () => Promise<void> | void, lists: ExpenseLists) {
  const vehicleStore = useVehicleStore()
  const { showAlert } = useConfirm()

  function deleteRow(list: Ref<any[]>, row: any, message: string, remove: (vehicleId: string, id: string) => Promise<unknown>) {
    const vehicle = vehicleStore.activeVehicle
    if (!vehicle) return
    const index = list.value.findIndex((r) => r.id === row.id)
    deleteWithUndo({
      message,
      hide: () => {
        list.value = list.value.filter((r) => r.id !== row.id)
      },
      restore: () => {
        if (list.value.some((r) => r.id === row.id)) return
        const next = [...list.value]
        next.splice(Math.min(Math.max(index, 0), next.length), 0, row)
        list.value = next
      },
      commit: () => remove(vehicle.id, row.id),
      committed: () => void reload(),
      onError: (err: any) => void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger'),
    })
  }

  function handleDeleteCharge(row: any) {
    deleteRow(lists.charges, row, t('expenses.expensesView.chargeDeleted'), (v, id) => api.deleteCharge(v, id))
  }

  // The receipt goes at once and comes back with Undo, the server is asked when that time is over
  function handleDeleteDocument(doc: ExpenseDocumentHeader) {
    const vehicle = vehicleStore.activeVehicle
    if (!vehicle) return
    deleteRowsWithUndo({
      list: documents,
      items: [doc],
      key: (d) => d.id,
      message: t('expenses.expensesView.receiptDeleted'),
      remove: (d) => api.deleteDocument(vehicle.id, d.id),
      committed: () => void reload(),
      onError: (err: any) => void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger'),
    })
  }

  // A selection goes at once under one Undo, the server is asked when that time is over
  function handleDeleteCharges(items: any[]) {
    const vehicle = vehicleStore.activeVehicle
    if (!vehicle || items.length === 0) return
    if (items.length === 1) return handleDeleteCharge(items[0])
    deleteRowsWithUndo({
      list: lists.charges,
      items,
      key: (c) => c.id,
      message: t('expenses.expensesView.chargesDeleted', { count: items.length }),
      remove: (c) => api.deleteCharge(vehicle.id, c.id),
      committed: () => void reload(),
      onError: (err: any) => void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger'),
    })
  }

  return { handleDeleteCharges, handleDeleteCharge, handleDeleteDocument }
}
