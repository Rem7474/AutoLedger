import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export type DrivesTab = 'DRIVES' | 'TRIPS' | 'CARPOOLS'

/** The tab named by the `tab` query parameter (trips, carpools); anything else is the list of drives. */
export function parseDrivesTab(value: unknown): DrivesTab {
  const raw = Array.isArray(value) ? value[0] : value
  return raw === 'TRIPS' || raw === 'CARPOOLS' ? raw : 'DRIVES'
}

/** The query with the tab set (`?tab=TRIPS`, `?tab=CARPOOLS`) or, for the list of drives, without it, the other parameters kept. */
export function drivesTabQuery(query: LocationQuery, tab: DrivesTab): LocationQueryRaw {
  const { tab: _previous, ...rest } = query
  return tab === 'DRIVES' ? rest : { ...rest, tab }
}
