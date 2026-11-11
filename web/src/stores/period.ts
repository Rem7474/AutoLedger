import { defineStore } from 'pinia'
import { ref } from 'vue'
import { DEFAULT_PERIOD, parsePeriod, periodFromLegacyRange, presetPeriod, type Period } from '@/utils/period'

const COSTS_KEY = 'teslacost_period_costs'
const CARPOOLS_KEY = 'teslacost_period_carpools'
const FUEL_KEY = 'teslacost_period_fuel'
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

function readOwn(key: string): Period {
  try {
    const raw = storage()?.getItem(key)
    const parsed = raw ? parsePeriod(JSON.parse(raw)) : null
    if (parsed) return parsed
  } catch {
    // a value that is not JSON reads as none
  }
  return presetPeriod('thisMonth')
}

const readCarpools = () => readOwn(CARPOOLS_KEY)
const readFuel = () => readOwn(FUEL_KEY)

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

  // The carpools have their own choice: a month at first, where the cost pages open on a year
  const carpools = ref<Period>(readCarpools())

  function setCarpools(period: Period) {
    carpools.value = period
    try {
      storage()?.setItem(CARPOOLS_KEY, JSON.stringify(period))
    } catch {
      // storage full or blocked: the choice holds for this visit
    }
  }

  // The fill-ups have their own choice too, a month at first: the list is read for what was filled up lately
  const fuel = ref<Period>(readFuel())

  function setFuel(period: Period) {
    fuel.value = period
    try {
      storage()?.setItem(FUEL_KEY, JSON.stringify(period))
    } catch {
      // storage full or blocked: the choice holds for this visit
    }
  }

  return { costs, setCosts, carpools, setCarpools, fuel, setFuel }
})
