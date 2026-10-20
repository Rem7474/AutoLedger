<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronUp } from 'lucide-vue-next'
import { euros } from '@/utils/carpool'
import { recapFigures } from '@/utils/carpoolEditor'
import { formatAmount } from '@/currency'

// The totals of the carpool as the form stands: what it costs, how that splits between the passengers and the driver, what
// was received against the fair shares, and what stays with the driver. On a phone it is a bar that opens on the details.
const props = defineProps<{
  live: any
  revenue: number
  currency: string
  compact?: boolean
}>()
const fmt = (v: number) => formatAmount(euros(v), props.currency)
const fig = computed(() => recapFigures(props.live, props.revenue))
const pct = (v: number) => `${Math.round(v)} %`
const open = ref(false)
// Nothing left to the driver: what was received covers the cost
const covered = computed(() => fig.value.total > 0 && fig.value.left <= 0)
</script>

<template>
  <section :class="compact ? '' : 'rounded-2xl border border-slate-800 bg-slate-950/60 p-4'" :aria-label="$t('carpool.editor.recap')">
    <button
      v-if="compact"
      type="button"
      class="flex w-full items-center justify-between gap-3 rounded-xl border px-3 py-2 text-left"
      :class="covered ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-slate-700 bg-slate-800/70'"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span class="min-w-0">
        <span class="block text-xs text-slate-400">{{ covered ? $t('carpool.carpoolTripModal.surplus') : $t('carpool.carpoolTripModal.leftToDriver') }}</span>
        <span class="block text-base font-bold tabular-nums" :class="covered ? 'text-emerald-300' : 'text-white'">{{ fmt(Math.abs(fig.left)) }}</span>
      </span>
      <span class="flex items-center gap-2 text-xs text-emerald-300">
        <span class="tabular-nums">{{ $t('carpool.editor.receivedOf', { received: pct(fig.receivedPct) }) }}</span>
        <ChevronUp class="h-4 w-4 text-slate-300 transition-transform" :class="open ? 'rotate-180' : ''" aria-hidden="true" />
      </span>
    </button>

    <div v-if="!compact || open" :class="compact ? 'mt-3 max-h-[40dvh] space-y-3 overflow-y-auto' : 'space-y-4'">
      <div v-if="!compact" class="rounded-xl border p-3" :class="covered ? 'border-emerald-500/30 bg-emerald-500/10' : 'border-slate-700 bg-slate-800/70'">
        <div class="text-xs text-slate-400">{{ covered ? $t('carpool.carpoolTripModal.surplus') : $t('carpool.carpoolTripModal.leftToDriver') }}</div>
        <div class="text-2xl font-bold tabular-nums" :class="covered ? 'text-emerald-300' : 'text-white'">{{ fmt(Math.abs(fig.left)) }}</div>
      </div>

      <dl class="grid grid-cols-2 gap-2 text-xs">
        <div class="rounded-xl border border-slate-700/70 bg-slate-800/70 p-2.5">
          <dt class="text-slate-400">{{ $t('carpool.carpoolTripModal.actualCost') }}</dt>
          <dd class="text-sm font-bold tabular-nums text-white">{{ fmt(fig.total) }}</dd>
        </div>
        <div class="rounded-xl border border-emerald-500/20 bg-emerald-500/10 p-2.5">
          <dt class="text-slate-400">{{ $t('carpool.carpoolTripModal.received') }}</dt>
          <dd class="text-sm font-bold tabular-nums text-emerald-300">{{ fmt(fig.received) }} <span class="font-semibold">· {{ pct(fig.receivedPct) }}</span></dd>
        </div>
        <div class="rounded-xl border border-sky-500/20 bg-sky-500/10 p-2.5">
          <dt class="text-slate-400">{{ $t('carpool.carpoolTripModal.passengersShares') }}</dt>
          <dd class="text-sm font-bold tabular-nums text-sky-300">{{ fmt(fig.passengersShare) }}</dd>
        </div>
        <div class="rounded-xl border border-slate-600/60 bg-slate-700/40 p-2.5">
          <dt class="text-slate-400">{{ $t('carpool.carpoolTripModal.driverSShare') }}</dt>
          <dd class="text-sm font-bold tabular-nums text-slate-100">{{ fmt(fig.driverShare) }}</dd>
        </div>
      </dl>

      <div class="space-y-1.5">
        <div class="text-xs text-slate-400">{{ $t('carpool.editor.receivedTitle') }}</div>
        <div class="relative h-2 rounded-full bg-slate-800" role="img" :aria-label="$t('carpool.editor.receivedLabel', { received: pct(fig.receivedPct), fair: pct(fig.passengersPct) })">
          <span class="absolute inset-y-0 left-0 rounded-full bg-emerald-400" :style="{ width: `${Math.min(fig.receivedPct, 100)}%` }"></span>
          <span class="absolute -inset-y-1 w-0.5 rounded bg-white" :style="{ left: `${Math.min(fig.passengersPct, 100)}%` }" :title="$t('carpool.editor.fairTick')"></span>
        </div>
        <p class="text-xs text-slate-400">{{ $t('carpool.editor.fairTickHint', { fair: pct(fig.passengersPct) }) }}</p>
      </div>

    </div>
  </section>
</template>
