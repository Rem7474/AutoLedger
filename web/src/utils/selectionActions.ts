import type { Component } from 'vue'
import { Pencil, Trash2 } from 'lucide-vue-next'
import { t } from '@/i18n'

// One action of the contextual bar that opens when rows are selected. An action that does not apply to the current
// selection stays visible but `disabled`, with the `reason` shown in the sheet, instead of disappearing.
export interface SelectionAction {
  key: string
  label: string
  icon: Component
  run: () => void
  primary?: boolean
  danger?: boolean
  // Only in the "More" sheet / menu, never as a shortcut in the bar
  overflow?: boolean
  disabled?: boolean
  reason?: string
  busy?: boolean
}

// What the bar shows directly: on a phone the first few as icons, on a desktop all of them as buttons
export const barActions = (actions: SelectionAction[]) => actions.filter((a) => !a.overflow)
export const phoneShortcuts = (actions: SelectionAction[], max = 3) => barActions(actions).slice(0, max)

// Edit (one row at a time) and delete (several at once; blocked when a row cannot be deleted, e.g. synced from
// TeslaMate) of a list of rows, for the lists that do nothing more with a selection.
export function editDeleteActions<T>(
  items: T[],
  o: { edit?: (item: T) => void; remove?: (items: T[]) => void; deletable?: (item: T) => boolean; lockedReason?: string },
): SelectionAction[] {
  const actions: SelectionAction[] = []
  if (o.edit) {
    const one = items.length === 1
    actions.push({
      key: 'edit',
      label: t('common.edit'),
      icon: Pencil,
      run: () => o.edit?.(items[0]),
      disabled: !one,
      reason: one ? undefined : t('shell.bulkSelectionBar.selectOne'),
    })
  }
  if (o.remove) {
    const locked = !!o.deletable && items.some((i) => !o.deletable!(i))
    actions.push({
      key: 'delete',
      label: t('common.delete'),
      icon: Trash2,
      danger: true,
      run: () => o.remove?.(items),
      disabled: locked,
      reason: locked ? (o.lockedReason ?? t('shell.bulkSelectionBar.cannotDelete')) : undefined,
    })
  }
  return actions
}
