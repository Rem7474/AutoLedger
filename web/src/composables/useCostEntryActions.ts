import { ref } from 'vue'
import { api, type CostEntryLine, type ExpenseDocumentHeader } from '@/services/api'
import { useVehicleStore } from '@/stores/vehicle'
import { useConfirm } from '@/composables/useConfirm'
import { deleteWithUndo } from '@/composables/useUndoableDelete'
import { chargeDeletable } from '@/utils/chargeProvenance'
import { entryTarget } from '@/utils/costEntries'
import { t } from '@/i18n'

/**
 * What the Costs page does with a line: open the record it comes from in its own form (read from the server by id, a
 * page of the journal holds only the figures), and delete it with a way back. `hide`/`restore` take the line off the
 * page and put it back; `reload` reads the journal again once the server has changed.
 */
export function useCostEntryActions(hooks: { hide: (id: string) => void; restore: (id: string) => void; reload: () => unknown }) {
  const vehicleStore = useVehicleStore()
  const { showAlert } = useConfirm()
  const showToll = ref(false)
  const showMaintenance = ref(false)
  const showCharge = ref(false)
  const editing = ref<any | null>(null)
  const documents = ref<ExpenseDocumentHeader[]>([])
  // The earlier maintenance entries, read when a new one is added: they suggest descriptions and close a revision
  const maintenanceExpenses = ref<any[]>([])

  const fail = (err: any) => void showAlert(t('common.errorWithMessage', { message: err?.message ?? '' }), t('shell.confirm.error'), 'danger')

  async function loadDocuments(vehicleId: string) {
    try {
      documents.value = await api.getDocuments(vehicleId)
    } catch (err) {
      console.error('Failed to load documents', err)
    }
  }

  /** Opens the maintenance form empty, for a new entry. */
  async function addMaintenance() {
    const vehicle = vehicleStore.activeVehicle
    if (!vehicle) return
    editing.value = null
    showMaintenance.value = true
    void loadDocuments(vehicle.id)
    try {
      maintenanceExpenses.value = await api.getMaintenance(vehicle.id)
    } catch (err) {
      console.error('Failed to load the maintenance entries', err)
    }
  }

  async function open(entry: CostEntryLine) {
    const vehicle = vehicleStore.activeVehicle
    const target = entryTarget(entry)
    if (!vehicle || target.type !== 'edit') return
    try {
      const [record] = await Promise.all([
        target.modal === 'charge' ? api.getCharge(vehicle.id, target.id) : target.modal === 'toll' ? api.getDriveExpense(vehicle.id, target.id) : api.getMaintenanceExpense(vehicle.id, target.id),
        loadDocuments(vehicle.id),
      ])
      editing.value = record
      if (target.modal === 'charge') showCharge.value = true
      else if (target.modal === 'toll') showToll.value = true
      else showMaintenance.value = true
    } catch (err) {
      fail(err)
    }
  }

  /** A charge reported by TeslaMate would come back with the next synchronisation: it is not deleted. */
  async function remove(entry: CostEntryLine, id: string) {
    const vehicle = vehicleStore.activeVehicle
    const target = entryTarget(entry)
    if (!vehicle || target.type !== 'edit') return
    if (target.modal === 'charge') {
      try {
        if (!chargeDeletable(await api.getCharge(vehicle.id, target.id))) {
          void showAlert(t('expenses.chargesPanel.syncedCannotDelete'), t('shell.confirm.error'), 'warning')
          return
        }
      } catch (err) {
        fail(err)
        return
      }
    }
    const erase = target.modal === 'charge' ? api.deleteCharge : target.modal === 'toll' ? api.deleteDriveExpense : api.deleteMaintenance
    deleteWithUndo({
      message: t(target.modal === 'charge' ? 'expenses.expensesView.chargeDeleted' : 'expenses.expensesView.costDeleted'),
      hide: () => hooks.hide(id),
      restore: () => hooks.restore(id),
      commit: () => erase(vehicle.id, target.id),
      committed: () => void hooks.reload(),
      onError: fail,
    })
  }

  function onDocumentAdded(doc: ExpenseDocumentHeader) {
    documents.value.unshift(doc)
  }

  return { showToll, showMaintenance, showCharge, editing, documents, maintenanceExpenses, open, addMaintenance, remove, onDocumentAdded }
}
