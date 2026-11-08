import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export type DrivesTab = 'DRIVES' | 'TRIPS'

/** The tab named by the `tab` query parameter; anything else is the list of drives. */
export function parseDrivesTab(value: unknown): DrivesTab {
  return (Array.isArray(value) ? value[0] : value) === 'TRIPS' ? 'TRIPS' : 'DRIVES'
}

/** The query with the tab set (`?tab=TRIPS`) or, for the list of drives, without it, the other parameters kept. */
export function drivesTabQuery(query: LocationQuery, tab: DrivesTab): LocationQueryRaw {
  const { tab: _previous, ...rest } = query
  return tab === 'TRIPS' ? { ...rest, tab } : rest
}
