import type { LocationQueryRaw } from 'vue-router'

type Section = 'expenses' | 'maintenance' | 'energy'

// The lists of tolls, maintenance and fixed expenses, and of charges, are read on the Costs page. The tool pages keep
// their own tabs (receipts, reminders, charges to complete, efficiency, estimate, fill-ups); an address that named a
// list tab, or none on a page that opened on a list, lands on the Costs page with the matching theme.
const LIST_TABS: Record<Section, Record<string, string | null>> = {
  expenses: { TOLLS: 'tolls', FIXED: null },
  maintenance: { MAINTENANCE: 'maintenance' },
  energy: {},
}

// The tab a page opened on before the lists moved: it is a list for these two pages
const DEFAULT_LIST_TAB: Partial<Record<Section, string>> = { expenses: 'TOLLS', maintenance: 'MAINTENANCE' }

/** Where an old address of a list goes on the Costs page, or null when the address is a tool page. */
export function listRedirect(section: Section, rawTab: unknown): { path: string; query: LocationQueryRaw } | null {
  const tab = String(Array.isArray(rawTab) ? rawTab[0] : (rawTab ?? '')).toUpperCase() || DEFAULT_LIST_TAB[section] || ''
  const lists = LIST_TABS[section]
  if (!Object.hasOwn(lists, tab)) return null
  const theme = lists[tab]
  return { path: '/costs', query: theme ? { theme } : {} }
}
