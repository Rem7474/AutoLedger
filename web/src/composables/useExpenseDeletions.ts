import type { Ref } from 'vue'
import { t } from '@/i18n'
import { api, type ExpenseDocumentHeader } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { deleteWithUndo } from '@/composables/useUndoableDelete'

export interface ExpenseLists {
  tolls: Ref<any[]>
  maintenance: Ref<any[]>
  charges: Ref<any[]>
}

// Delete handlers of the expenses page; `reload` refreshes the list on screen afterwards. A single row is deleted
// without a question and can be taken back for a few seconds; a selection and a receipt still ask first.
export function useExpenseDeletions(documents: Ref<ExpenseDocumentHeader[]>, reload: () => Promise<void> | void, lists: ExpenseLists) {
  const vehicleStore = useVehicleStore()
  const { showConfirm, showAlert } = useConfirm()

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

  function handleDeleteToll(row: any) {
    deleteRow(lists.tolls, row, t('expenses.expensesView.costDeleted'), (v, id) => api.deleteDriveExpense(v, id))
  }

  function handleDeleteMaint(row: any) {
    deleteRow(lists.maintenance, row, t('expenses.expensesView.costDeleted'), (v, id) => api.deleteMaintenance(v, id))
  }

  function handleDeleteCharge(row: any) {
    deleteRow(lists.charges, row, t('expenses.expensesView.chargeDeleted'), (v, id) => api.deleteCharge(v, id))
  }

  async function handleDeleteDocument(doc: ExpenseDocumentHeader) {
    if (!vehicleStore.activeVehicle) return
    const ok = await showConfirm({
      title: t('expenses.expensesView.deleteReceiptTitle'),
      message: t('expenses.expensesView.deleteReceiptMessage', { filename: doc.filename }),
      confirmText: t('common.delete'),
      type: 'danger',
    })
    if (!ok) return
    try {
      await api.deleteDocument(vehicleStore.activeVehicle.id, doc.id)
      documents.value = documents.value.filter((d) => d.id !== doc.id)
      void showAlert(t('expenses.expensesView.receiptDeleted'), t('common.success'), 'success')
      await reload()
    } catch (err: any) {
      void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
    }
  }

  // One confirmation for a selection, then the rows are deleted one by one; the list reloads even when one fails
  async function deleteMany(items: any[], title: string, message: string, remove: (vehicleId: string, id: string) => Promise<unknown>) {
    const vehicle = vehicleStore.activeVehicle
    if (!vehicle || items.length === 0) return
    const ok = await showConfirm({ title, message, confirmText: t('common.delete'), type: 'danger' })
    if (!ok) return
    try {
      for (const item of items) await remove(vehicle.id, item.id)
    } catch (err: any) {
      void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger')
    }
    await reload()
  }
  const deleteManyMessage = (count: number) => t('expenses.expensesView.deleteManyMessage', { count })
  const handleDeleteTolls = (items: any[]) =>
    items.length === 1 ? handleDeleteToll(items[0]) : deleteMany(items, t('expenses.expensesView.deleteExpenseTitle'), deleteManyMessage(items.length), (v, id) => api.deleteDriveExpense(v, id))
  const handleDeleteMaints = (items: any[]) =>
    items.length === 1 ? handleDeleteMaint(items[0]) : deleteMany(items, t('expenses.expensesView.deleteExpenseTitle'), deleteManyMessage(items.length), (v, id) => api.deleteMaintenance(v, id))
  const handleDeleteCharges = (items: any[]) =>
    items.length === 1 ? handleDeleteCharge(items[0]) : deleteMany(items, t('expenses.expensesView.deleteChargeTitle'), deleteManyMessage(items.length), (v, id) => api.deleteCharge(v, id))

  return { handleDeleteTolls, handleDeleteMaints, handleDeleteCharges, handleDeleteToll, handleDeleteMaint, handleDeleteCharge, handleDeleteDocument }
}
