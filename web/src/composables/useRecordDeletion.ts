import type { Ref } from 'vue'
import { t } from '@/i18n'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { deleteRowsWithUndo, deleteWithUndo } from '@/composables/useUndoableDelete'
import { chargeDeletable } from '@/utils/chargeProvenance'

export type RecordKind = 'charge' | 'fuel_log'

/** Where the rows are shown: a list held in a ref, or a page that hides and restores a row by its own key. */
export type RecordPlace = { list: Ref<any[]> } | { hide: () => void; restore: () => void }

export interface RecordDeletion {
  kind: RecordKind
  /** The rows to delete. A selection goes under one snackbar. */
  items: { id: string }[]
  place: RecordPlace
  /** The rows are the full records, so a charge's origin is read from them. Without it a charge is read from the server first. */
  loaded?: boolean
  /** The server has deleted every row: the list on screen is read again. */
  reload: () => unknown
}

/**
 * The one way a charge or a fill-up is deleted, whichever list it is deleted from: a charge reported by TeslaMate is
 * refused (it would come back with the next synchronisation), otherwise the rows go at once, one snackbar offers to
 * take them back, and the server is asked when that time is over, one row after the other. The first refusal stops
 * the rest and the rows not deleted come back.
 */
export function useRecordDeletion() {
  const vehicleStore = useVehicleStore()
  const { showAlert } = useConfirm()

  function fail(kind: RecordKind, err: any) {
    const text = kind === 'charge' ? t('common.deleteError', { message: err?.message ?? '' }) : err?.message || t('manual.fuelLogsPanel.deleteFailed')
    void showAlert(text, t('shell.confirm.error'), 'danger')
  }

  function erase(kind: RecordKind, vehicleId: string, id: string) {
    return kind === 'charge' ? api.deleteCharge(vehicleId, id) : api.deleteFuelLog(vehicleId, id)
  }

  function snackbar(kind: RecordKind, count: number) {
    if (kind === 'charge') return count === 1 ? t('expenses.expensesView.chargeDeleted') : t('expenses.expensesView.chargesDeleted', { count })
    return count === 1 ? t('manual.fuelLogsPanel.fillUpDeleted') : t('manual.fuelLogsPanel.fillUpsDeleted', { count })
  }

  /** True when nothing in the selection is a charge the synchronisation would bring back. */
  async function allowed(d: RecordDeletion, vehicleId: string): Promise<boolean> {
    if (d.kind !== 'charge') return true
    try {
      const records = d.loaded ? d.items : await Promise.all(d.items.map((item) => api.getCharge(vehicleId, item.id)))
      if (records.every((r) => chargeDeletable(r))) return true
      void showAlert(t('expenses.chargesPanel.syncedCannotDelete'), t('shell.confirm.error'), 'warning')
    } catch (err: any) {
      void showAlert(t('common.errorWithMessage', { message: err?.message ?? '' }), t('shell.confirm.error'), 'danger')
    }
    return false
  }

  async function removeRecords(d: RecordDeletion): Promise<void> {
    const vehicle = vehicleStore.activeVehicle
    if (!vehicle || d.items.length === 0) return
    if (!(await allowed(d, vehicle.id))) return
    const onError = (err: unknown) => fail(d.kind, err)
    const committed = () => void d.reload()
    if ('list' in d.place) {
      deleteRowsWithUndo({
        list: d.place.list,
        items: d.items,
        key: (r) => r.id,
        message: snackbar(d.kind, d.items.length),
        remove: (r) => erase(d.kind, vehicle.id, r.id),
        committed,
        onError,
      })
      return
    }
    const [row] = d.items
    deleteWithUndo({
      message: snackbar(d.kind, 1),
      hide: d.place.hide,
      restore: d.place.restore,
      commit: () => erase(d.kind, vehicle.id, row.id),
      committed,
      onError,
    })
  }

  return { removeRecords }
}
