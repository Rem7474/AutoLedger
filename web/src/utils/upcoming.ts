import type { MaintenanceReminder } from '@/services/api'
import type { TireSetSummary } from '@/utils/tires'
import { nextDueReminder } from '@/utils/dashboard'

export type UpcomingTone = 'late' | 'soon' | 'plain'

export interface UpcomingItem {
  key: string
  kind: 'reminder' | 'tire'
  tone: UpcomingTone
  /** The page the row opens. */
  to: string
  reminder?: MaintenanceReminder
  tire?: NonNullable<TireSetSummary['mostWorn']>
}

const TONE_ORDER: Record<UpcomingTone, number> = { late: 0, soon: 1, plain: 2 }

/**
 * What the "Coming up" block lists: the reminders that are overdue, then those due soon, the next planned one (the others live on the reminders page), and the
 * most worn tire once it is worth a warning. Late first; inside a tone the order of the reminders is kept.
 */
export function upcomingItems(reminders: MaintenanceReminder[], tires: TireSetSummary | null): UpcomingItem[] {
  const items: UpcomingItem[] = []
  for (const r of reminders) {
    if (r.status === 'OVERDUE') items.push({ key: `reminder-${r.id}`, kind: 'reminder', tone: 'late', to: '/maintenance?tab=REMINDERS', reminder: r })
    else if (r.status === 'DUE_SOON') items.push({ key: `reminder-${r.id}`, kind: 'reminder', tone: 'soon', to: '/maintenance?tab=REMINDERS', reminder: r })
  }
  const next = nextDueReminder(reminders.filter((r) => r.status === 'OK'))
  if (next) {
    items.push({ key: `reminder-${next.reminder.id}`, kind: 'reminder', tone: 'plain', to: '/maintenance?tab=REMINDERS', reminder: next.reminder })
  }
  const worn = tires?.mostWorn
  if (worn && worn.tone !== 'ok') {
    items.push({ key: 'tire', kind: 'tire', tone: worn.tone === 'danger' ? 'late' : 'soon', to: '/tires', tire: worn })
  }
  return items
    .map((item, index) => ({ item, index }))
    .sort((a, b) => TONE_ORDER[a.item.tone] - TONE_ORDER[b.item.tone] || a.index - b.index)
    .map(({ item }) => item)
}

/** True when the reminders have no schedule at all: the block then invites to plan one instead of saying nothing is due. */
export function needsFirstReminder(reminders: MaintenanceReminder[]): boolean {
  return !reminders.some((r) => r.status === 'OVERDUE' || r.status === 'DUE_SOON' || nextDueReminder([r]))
}
