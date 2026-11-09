import { defineStore } from 'pinia'
import { ref } from 'vue'
import { DEFAULT_PERIOD, parsePeriod, periodFromLegacyRange, type Period } from '@/utils/period'

const COSTS_KEY = 'teslacost_period_costs'
const LEGACY_DASHBOARD_RANGE_KEY = 'teslacost_dashboard_range'

const storage = () => {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage
  } catch {
    return null
  }
}

function readCosts(): Period {
  const store = storage()
  try {
    const raw = store?.getItem(COSTS_KEY)
    const parsed = raw ? parsePeriod(JSON.parse(raw)) : null
    if (parsed) return parsed
  } catch {
    // a value that is not JSON reads as none
  }
  // The home page's selector kept its choice under its own key before the period was shared
  return periodFromLegacyRange(store?.getItem(LEGACY_DASHBOARD_RANGE_KEY) ?? null) ?? DEFAULT_PERIOD
}

// The period of the pages about what the vehicle costs (home, energy, expenses): one choice for all of them, kept in
// this browser. A URL that names a period wins over it (see the page's use of it) and does not overwrite it.
export const usePeriodStore = defineStore('period', () => {
  const costs = ref<Period>(readCosts())

  function setCosts(period: Period) {
    costs.value = period
    try {
      storage()?.setItem(COSTS_KEY, JSON.stringify(period))
    } catch {
      // storage full or blocked: the choice holds for this visit
    }
  }

  return { costs, setCosts }
})
