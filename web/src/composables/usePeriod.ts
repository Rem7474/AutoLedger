import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePeriodStore } from '@/stores/period'
import { isRoundedToMonths, periodBounds, periodFromQuery, periodLabel, queryNamesPeriod, stepPeriod, canStep, withPeriodQuery, type Period } from '@/utils/period'

/**
 * The period of the pages about cost (home, energy, expenses), shared through the store and written in the URL.
 * A period named by the URL on arrival (a shared link, a reload) is adopted and kept; afterwards the URL follows the
 * store. The query is replaced, never pushed, so a change of period leaves no history entry.
 */
export function usePeriod() {
  const store = usePeriodStore()
  const route = useRoute()
  const router = useRouter()
  const path = route.path

  const fromUrl = periodFromQuery(route.query)
  if (fromUrl) store.setCosts(fromUrl)

  const period = computed(() => store.costs)
  const bounds = computed(() => periodBounds(store.costs))
  const label = computed(() => periodLabel(store.costs))
  const rounded = computed(() => isRoundedToMonths(store.costs))
  const canPrevious = computed(() => canStep(store.costs, -1))
  const canNext = computed(() => canStep(store.costs, 1))

  const set = (p: Period) => store.setCosts(p)
  const step = (dir: -1 | 1) => store.setCosts(stepPeriod(store.costs, dir))

  // The URL says what the store holds. Watching the query as well puts it back if something else rewrote it (the
  // system Back popping an entry a sheet held).
  watch(
    [() => store.costs, () => route.query],
    () => {
      if (route.path !== path || queryNamesPeriod(route.query, store.costs)) return
      void router.replace({ query: withPeriodQuery(route.query, store.costs) as Record<string, string> })
    },
    { immediate: true },
  )

  return { period, bounds, label, rounded, canPrevious, canNext, set, step }
}
