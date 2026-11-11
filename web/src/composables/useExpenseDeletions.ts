import type { Ref } from 'vue'
import { t } from '@/i18n'
import { api, type ExpenseDocumentHeader } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { deleteRowsWithUndo } from '@/composables/useUndoableDelete'
import { useRecordDeletion } from '@/composables/useRecordDeletion'

export interface ExpenseLists {
  charges: Ref<any[]>
}

// Delete handlers of the expenses page; `reload` refreshes the list on screen afterwards. A row, a selection and a
// receipt are deleted without a question and can be taken back for a few seconds.
export function useExpenseDeletions(documents: Ref<ExpenseDocumentHeader[]>, reload: () => Promise<void> | void, lists: ExpenseLists) {
  const vehicleStore = useVehicleStore()
  const { showAlert } = useConfirm()
  const { removeRecords } = useRecordDeletion()

  function handleDeleteCharge(row: any) {
    return removeRecords({ kind: 'charge', items: [row], place: { list: lists.charges }, loaded: true, reload })
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
    return removeRecords({ kind: 'charge', items, place: { list: lists.charges }, loaded: true, reload })
  }

  return { handleDeleteCharges, handleDeleteCharge, handleDeleteDocument }
}
