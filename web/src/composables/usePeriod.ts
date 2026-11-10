import { computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { usePeriodStore } from '@/stores/period'
import { isRoundedToMonths, periodBounds, periodFromQuery, periodLabel, queryNamesPeriod, stepPeriod, canStep, withPeriodQuery, type Period } from '@/utils/period'

/**
 * The period of the pages about cost (home, energy, expenses), shared through the store and written in the URL; the
 * carpools keep their own (`scope` 'carpools', a month at first).
 * A period named by the URL on arrival (a shared link, a reload) is adopted and kept; afterwards the URL follows the
 * store. The query is replaced, never pushed, so a change of period leaves no history entry.
 */
export function usePeriod(scope: 'costs' | 'carpools' = 'costs') {
  const store = usePeriodStore()
  const route = useRoute()
  const router = useRouter()
  const path = route.path
  const current = () => (scope === 'costs' ? store.costs : store.carpools)
  const write = (p: Period) => (scope === 'costs' ? store.setCosts(p) : store.setCarpools(p))

  const fromUrl = periodFromQuery(route.query)
  if (fromUrl) write(fromUrl)

  const period = computed(() => current())
  const bounds = computed(() => periodBounds(current()))
  const label = computed(() => periodLabel(current()))
  const rounded = computed(() => isRoundedToMonths(current()))
  const canPrevious = computed(() => canStep(current(), -1))
  const canNext = computed(() => canStep(current(), 1))

  const set = (p: Period) => write(p)
  const step = (dir: -1 | 1) => write(stepPeriod(current(), dir))

  // The URL says what the store holds. Watching the query as well puts it back if something else rewrote it (the
  // system Back popping an entry a sheet held).
  watch(
    [() => current(), () => route.query],
    () => {
      if (route.path !== path || queryNamesPeriod(route.query, current())) return
      void router.replace({ query: withPeriodQuery(route.query, current()) as Record<string, string> })
    },
    { immediate: true },
  )

  return { period, bounds, label, rounded, canPrevious, canNext, set, step }
}
