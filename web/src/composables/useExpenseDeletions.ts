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

  return { handleDeleteToll, handleDeleteMaint, handleDeleteCharge, handleDeleteDocument }
}
