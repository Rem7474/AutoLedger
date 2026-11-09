<script setup lang="ts">
import { computed } from 'vue'
import { CalendarClock, ChevronRight, Disc } from 'lucide-vue-next'
import type { MaintenanceReminder } from '@/services/api'
import type { TireSetSummary } from '@/utils/tires'
import { needsFirstReminder, upcomingItems, type UpcomingItem } from '@/utils/upcoming'
import { reminderDaysLabel } from '@/utils/expenses'
import { formatPercent } from '@/utils/numbers'
import { wearAndRemaining } from '@/utils/tires'
import { distanceUnit, formatDistanceValue } from '@/units'
import { t } from '@/i18n'

// What is coming up for the vehicle: maintenance that is late or due, the next one planned, the tires once worn.
// A row says its figure in words and opens the page where it is dealt with; amber and red are the judgement, a planned row has none.
const props = defineProps<{ reminders: MaintenanceReminder[]; tires: TireSetSummary | null; canEdit: boolean }>()
const items = computed(() => upcomingItems(props.reminders, props.tires))
const invite = computed(() => needsFirstReminder(props.reminders))

const TONE = {
  late: { tint: 'bg-danger-500/15 text-danger-400', text: 'text-danger-400' },
  soon: { tint: 'bg-warning-500/15 text-warning-400', text: 'text-warning-400' },
  plain: { tint: 'bg-slate-800 text-slate-300', text: 'text-slate-300' },
} as const

function detail(item: UpcomingItem): string {
  if (item.tire) {
    const { worn, left } = wearAndRemaining(item.tire.pct)
    return t('tires.tireSetSummary.wornAndLeft', { worn: formatPercent(worn, 0), left: formatPercent(left, 0) })
  }
  const r = item.reminder!
  const parts: string[] = []
  if (r.remaining_km != null) {
    parts.push(
      r.remaining_km <= 0
        ? t('expenses.remindersPanel.kmOver', { unit: distanceUnit(), km: formatDistanceValue(Math.abs(Math.round(r.remaining_km))) })
        : t('expenses.remindersPanel.kmLeft', { unit: distanceUnit(), km: formatDistanceValue(r.remaining_km) }),
    )
  }
  if (r.remaining_days != null) parts.push(reminderDaysLabel({ remaining_days: r.remaining_days, scheduled_date: r.scheduled_date }))
  return parts.join(' · ')
}
const title = (item: UpcomingItem) => (item.tire ? t('dashboard.today.tireTitle', { name: item.tire.name }) : item.reminder!.title)
</script>

<template>
  <section aria-labelledby="today-upcoming" data-testid="upcoming">
    <h2 id="today-upcoming" class="mb-2 px-1 text-xs font-semibold uppercase tracking-wide text-slate-400">{{ $t('dashboard.today.upcoming') }}</h2>
    <ul v-if="items.length" class="card divide-y divide-slate-800/70 overflow-hidden">
      <li v-for="item in items" :key="item.key">
        <router-link :to="item.to" class="flex min-h-14 items-center gap-3 px-4 py-3 transition-colors active:bg-slate-800/60">
          <span class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl" :class="TONE[item.tone].tint">
            <Disc v-if="item.kind === 'tire'" class="h-5 w-5" aria-hidden="true" />
            <CalendarClock v-else class="h-5 w-5" aria-hidden="true" />
          </span>
          <span class="min-w-0 flex-1">
            <span class="block truncate text-sm font-semibold text-white">{{ title(item) }}</span>
            <span class="block truncate text-xs" :class="TONE[item.tone].text">{{ detail(item) }}</span>
          </span>
          <ChevronRight class="h-5 w-5 shrink-0 text-slate-500" aria-hidden="true" />
        </router-link>
      </li>
    </ul>
    <div v-else class="card flex items-center justify-between gap-3 p-4" data-testid="upcoming-empty">
      <p class="min-w-0 text-sm text-slate-300">{{ invite ? $t('dashboard.today.nothingPlanned') : $t('dashboard.today.nothingSoon') }}</p>
      <router-link v-if="invite && canEdit" to="/maintenance?tab=REMINDERS" class="btn btn-secondary tap-text shrink-0">{{ $t('dashboard.today.planOne') }}</router-link>
    </div>
  </section>
</template>
