import type { LocationQuery, LocationQueryRaw } from 'vue-router'

export type CostsTab = 'JOURNAL' | 'ANALYSIS'

/** The tab named by the `tab` query parameter; anything else is the journal. */
export function parseCostsTab(value: unknown): CostsTab {
  return (Array.isArray(value) ? value[0] : value) === 'ANALYSIS' ? 'ANALYSIS' : 'JOURNAL'
}

/** The query with the tab set (`?tab=ANALYSIS`) or, for the journal, without it, the other parameters kept. */
export function costsTabQuery(query: LocationQuery, tab: CostsTab): LocationQueryRaw {
  const { tab: _previous, ...rest } = query
  return tab === 'ANALYSIS' ? { ...rest, tab } : rest
}
