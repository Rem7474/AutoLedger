import { intlLocale } from '@/i18n'
import { toDateInputString } from '@/utils/carpool'

// Pure helpers of the carpool editor: the stops a passenger rides between, the recap figures, and
// what the drive picker offers (chains of drives, short drives, days).

// ---------- Stops ----------

export interface StopRange {
  board: number
  alight: number
}

const clamp = (v: number, min: number, max: number) => Math.min(Math.max(v, min), max)

/** The stop nearest to a position along the route (0 = first stop, 1 = last stop). */
export function stopFromRatio(ratio: number, legCount: number): number {
  return clamp(Math.round(ratio * legCount), 0, legCount)
}

/** Moves the boarding stop; a passenger always rides at least one leg. */
export function withBoard(range: StopRange, stop: number): StopRange {
  return { board: clamp(stop, 0, range.alight - 1), alight: range.alight }
}

/** Moves the alighting stop; a passenger always rides at least one leg. */
export function withAlight(range: StopRange, stop: number, legCount: number): StopRange {
  return { board: range.board, alight: clamp(stop, range.board + 1, legCount) }
}

/** A tap on a stop: outside the ride it extends it, inside it moves the nearer end (the alighting one on a tie). */
export function tapStop(range: StopRange, stop: number, legCount: number): StopRange {
  if (stop <= range.board) return withBoard(range, stop)
  if (stop >= range.alight) return withAlight(range, stop, legCount)
  return stop - range.board < range.alight - stop ? withBoard(range, stop) : withAlight(range, stop, legCount)
}

// ---------- Recap ----------

/** The figures of the recap: the split of the cost, what was received against the fair shares, what is left. */
export function recapFigures(live: { total: number; passengersShare: number; driverShare: number }, revenue: number) {
  const pct = (v: number) => (live.total > 0 ? (v / live.total) * 100 : 0)
  return {
    total: live.total,
    passengersShare: live.passengersShare,
    driverShare: live.driverShare,
    passengersPct: pct(live.passengersShare),
    driverPct: pct(live.driverShare),
    received: revenue,
    receivedPct: pct(revenue),
    left: live.total - revenue,
  }
}

// ---------- Drive picker ----------

/** Drives shorter than this are hidden behind a link: a parking move is rarely part of a carpool. */
export const SHORT_DRIVE_KM = 1
/** Longest pause between two drives still read as one journey. */
const CHAIN_MAX_GAP_MS = 6 * 3600 * 1000

const place = (address?: string | null) => (address || '').split(',')[0].trim().toLowerCase()

/**
 * The most recent run of at least two drives where each starts where the previous one ended (same place, after a short
 * pause), oldest first; empty when there is none.
 */
export function findDriveChain(drives: any[]): any[] {
  const sorted = [...drives].sort((a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime())
  let chain: any[] = []
  let best: any[] = []
  for (const d of sorted) {
    const prev = chain[chain.length - 1]
    const linked =
      prev &&
      place(prev.end_address) !== '' &&
      place(prev.end_address) === place(d.start_address) &&
      new Date(d.start_time).getTime() - new Date(prev.end_time || prev.start_time).getTime() <= CHAIN_MAX_GAP_MS
    chain = linked ? [...chain, d] : [d]
    if (chain.length >= 2) best = chain
  }
  return best
}

/** Splits the drives of the picker: the short ones (not selected, not in a carpool) are hidden until asked for. */
export function splitShortDrives(drives: any[], selectedIds: string[]) {
  const hidden = drives.filter((d) => Number(d.distance_km) < SHORT_DRIVE_KM && !selectedIds.includes(d.id) && !d.carpools?.length)
  return { shown: drives.filter((d) => !hidden.includes(d)), hidden }
}

/** Drives grouped by calendar day, days and drives in chronological order. */
export function groupDrivesByDay(drives: any[]): Array<{ day: string; drives: any[] }> {
  const sorted = [...drives].sort((a, b) => new Date(a.start_time).getTime() - new Date(b.start_time).getTime())
  const groups: Array<{ day: string; drives: any[] }> = []
  for (const d of sorted) {
    const day = toDateInputString(d.start_time)
    const last = groups[groups.length - 1]
    if (last?.day === day) last.drives.push(d)
    else groups.push({ day, drives: [d] })
  }
  return groups
}

/** The heading of a day of the picker ("Saturday 18 October"), from its yyyy-mm-dd date. */
export function formatDriveDay(day: string): string {
  const [y, m, d] = day.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString(intlLocale(), { weekday: 'long', day: 'numeric', month: 'long' })
}

/** The time of day of a drive ("18:42"). */
export function formatDriveClock(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString(intlLocale(), { hour: '2-digit', minute: '2-digit' })
}

/** The leg number of each selected drive, in the order they are driven (the oldest is leg 1). */
export function driveSteps(drives: any[]): Map<string, number> {
  const ordered = [...drives].sort((a, b) => String(a.start_time).localeCompare(String(b.start_time)))
  return new Map(ordered.map((d, i) => [d.id, i + 1]))
}

/** Whether two consecutive drives meet: the second starts where the first ended. */
export function drivesChain(prev: any, next: any): boolean {
  return place(prev.end_address) !== '' && place(prev.end_address) === place(next.start_address)
}
