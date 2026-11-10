import type { MaintenanceReminder } from '@/services/api'
import type { TireSetSummary } from '@/utils/tires'
import { nextDueReminder } from '@/utils/dashboard'

export type UpcomingTone = 'late' | 'soon' | 'plain'

export interface UpcomingItem {
  key: string
  kind: 'reminder' | 'tire' | 'undated' | 'carpool' | 'lease'
  tone: UpcomingTone
  /** The page the row opens. */
  to: string
  reminder?: MaintenanceReminder
  tire?: NonNullable<TireSetSummary['mostWorn']>
  /** `undated`: how many reminders have neither a due date nor a mileage. */
  count?: number
  /** `carpool`: what the passengers still owe, in the vehicle's currency. */
  amount?: number
  /** `lease`: when the contract ends and how many months are left. */
  leaseEnd?: Date
  months?: number
}

/** What the block lists besides the reminders and the tires: the carpools not settled and the end of a lease. */
export interface UpcomingExtras {
  /** What the passengers should have paid for the carpools minus what they paid; only a positive amount is listed. */
  carpoolOwed?: number | null
  lease?: { endDate: Date | null; remainingMonths: number; isEnded: boolean; isNotStarted: boolean } | null
}

/** The contract is listed once it ends within this many months. */
export const LEASE_HORIZON_MONTHS = 6
/** A contract ending within this many months is a warning. */
export const LEASE_SOON_MONTHS = 3

/** Reminders that are in order but have nothing to count down: no date, no mileage. They never come due on their own. */
export function undatedReminders(reminders: MaintenanceReminder[]): MaintenanceReminder[] {
  return reminders.filter((r) => r.status === 'OK' && !nextDueReminder([r]))
}

const TONE_ORDER: Record<UpcomingTone, number> = { late: 0, soon: 1, plain: 2 }

/**
 * What the "Coming up" block lists: the reminders that are overdue, then those due soon, the next planned one (the others live on the reminders page), and the
 * most worn tire once it is worth a warning. Late first; inside a tone the order of the reminders is kept.
 */
export function upcomingItems(reminders: MaintenanceReminder[], tires: TireSetSummary | null, extras: UpcomingExtras = {}): UpcomingItem[] {
  const items: UpcomingItem[] = []
  for (const r of reminders) {
    if (r.status === 'OVERDUE') items.push({ key: `reminder-${r.id}`, kind: 'reminder', tone: 'late', to: '/maintenance?tab=REMINDERS', reminder: r })
    else if (r.status === 'DUE_SOON') items.push({ key: `reminder-${r.id}`, kind: 'reminder', tone: 'soon', to: '/maintenance?tab=REMINDERS', reminder: r })
  }
  const next = nextDueReminder(reminders.filter((r) => r.status === 'OK'))
  if (next) {
    items.push({ key: `reminder-${next.reminder.id}`, kind: 'reminder', tone: 'plain', to: '/maintenance?tab=REMINDERS', reminder: next.reminder })
  }
  const lease = extras.lease
  if (lease?.endDate && !lease.isEnded && !lease.isNotStarted && lease.remainingMonths <= LEASE_HORIZON_MONTHS) {
    items.push({ key: 'lease', kind: 'lease', tone: lease.remainingMonths <= LEASE_SOON_MONTHS ? 'soon' : 'plain', to: '/vehicle', leaseEnd: lease.endDate, months: lease.remainingMonths })
  }
  if ((extras.carpoolOwed ?? 0) >= 0.005) {
    items.push({ key: 'carpool', kind: 'carpool', tone: 'plain', to: '/carpools', amount: extras.carpoolOwed! })
  }
  const undated = undatedReminders(reminders)
  if (undated.length) {
    items.push({ key: 'undated', kind: 'undated', tone: 'plain', to: '/maintenance?tab=REMINDERS', count: undated.length })
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

/** What the passengers of the carpools still owe: their fair share of the trips minus what they paid; null when settled or unknown. */
export function carpoolOwed(summary: { total_passengers_share?: number; total_revenue?: number } | null | undefined): number | null {
  const owed = Number(summary?.total_passengers_share || 0) - Number(summary?.total_revenue || 0)
  return owed > 0 ? owed : null
}
