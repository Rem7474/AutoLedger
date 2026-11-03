import type { Ref } from 'vue'
import { t } from '@/i18n'
import { api, type ExpenseDocumentHeader } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { formatAmount } from '@/currency'
import { formatNumber } from '@/utils/numbers'

// Confirm-then-delete handlers of the expenses page; `reload` refreshes the list on screen afterwards.
export function useExpenseDeletions(documents: Ref<ExpenseDocumentHeader[]>, reload: () => Promise<void> | void) {
  const vehicleStore = useVehicleStore()
  const { showConfirm, showAlert } = useConfirm()

  async function handleDeleteToll(e: any) {
    if (!vehicleStore.activeVehicle) return
    const ok = await showConfirm({
      title: t('expenses.expensesView.deleteExpenseTitle'),
      message: t('expenses.expensesView.deleteTollMessage', { amount: formatAmount(Number(e.amount), e.currency || vehicleStore.currency) }),
      confirmText: t('common.delete'),
      type: 'danger',
    })
    if (!ok) return
    try {
      await api.deleteDriveExpense(vehicleStore.activeVehicle.id, e.id)
      await reload()
    } catch (err: any) {
      void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger')
    }
  }

  async function handleDeleteMaint(m: any) {
    if (!vehicleStore.activeVehicle) return
    const ok = await showConfirm({
      title: t('expenses.expensesView.deleteExpenseTitle'),
      message: t('expenses.expensesView.deleteExpenseMessage', { description: m.description, amount: formatAmount(Number(m.amount), m.currency || vehicleStore.currency) }),
      confirmText: t('common.delete'),
      type: 'danger',
    })
    if (!ok) return
    try {
      await api.deleteMaintenance(vehicleStore.activeVehicle.id, m.id)
      await reload()
    } catch (err: any) {
      void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger')
    }
  }

  async function handleDeleteCharge(c: any) {
    if (!vehicleStore.activeVehicle) return
    const ok = await showConfirm({
      title: t('expenses.expensesView.deleteChargeTitle'),
      message: t('expenses.expensesView.deleteChargeMessage', { kwh: formatNumber(c.kwh_added, 2) }),
      confirmText: t('common.delete'),
      type: 'danger',
    })
    if (!ok) return
    try {
      await api.deleteCharge(vehicleStore.activeVehicle.id, c.id)
      await reload()
    } catch (err: any) {
      void showAlert(t('common.deleteError', { message: err.message }), t('shell.confirm.error'), 'danger')
    }
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
