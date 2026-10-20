<script setup lang="ts">
import { computed, ref } from 'vue'
import { ChevronUp, Copy, Check } from 'lucide-vue-next'
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
  copyLabel?: string
  copied?: boolean
}>()
const emit = defineEmits<{ copy: [] }>()
const fmt = (v: number) => formatAmount(euros(v), props.currency)
const fig = computed(() => recapFigures(props.live, props.revenue))
const pct = (v: number) => `${Math.round(v)} %`
const open = ref(false)
</script>

<template>
  <section :class="compact ? '' : 'rounded-2xl border border-slate-800 bg-slate-950/60 p-4'" :aria-label="$t('carpool.editor.recap')">
    <button
      v-if="compact"
      type="button"
      class="flex w-full items-center justify-between gap-3 text-left"
      :aria-expanded="open"
      @click="open = !open"
    >
      <span class="min-w-0">
        <span class="block text-xs text-slate-400">{{ fig.left > 0 ? $t('carpool.carpoolTripModal.leftToDriver') : $t('carpool.carpoolTripModal.surplus') }}</span>
        <span class="block text-base font-bold tabular-nums text-white">{{ fmt(Math.abs(fig.left)) }}</span>
      </span>
      <span class="flex items-center gap-2 text-xs text-slate-300">
        <span class="tabular-nums">{{ $t('carpool.editor.receivedOf', { received: pct(fig.receivedPct) }) }}</span>
        <ChevronUp class="h-4 w-4 transition-transform" :class="open ? 'rotate-180' : ''" aria-hidden="true" />
      </span>
    </button>

    <div v-if="!compact || open" :class="compact ? 'mt-3 max-h-[40dvh] space-y-3 overflow-y-auto' : 'space-y-4'">
      <div v-if="!compact">
        <div class="text-xs text-slate-400">{{ fig.left > 0 ? $t('carpool.carpoolTripModal.leftToDriver') : $t('carpool.carpoolTripModal.surplus') }}</div>
        <div class="text-2xl font-bold tabular-nums text-white">{{ fmt(Math.abs(fig.left)) }}</div>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-baseline justify-between text-xs">
          <span class="text-slate-400">{{ $t('carpool.carpoolTripModal.actualCost') }}</span>
          <strong class="tabular-nums text-slate-100">{{ fmt(fig.total) }}</strong>
        </div>
        <div class="flex h-2 overflow-hidden rounded-full bg-slate-800" role="img" :aria-label="$t('carpool.editor.splitLabel', { passengers: pct(fig.passengersPct), driver: pct(fig.driverPct) })">
          <span class="bg-slate-300" :style="{ width: `${fig.passengersPct}%` }"></span>
          <span class="bg-slate-600" :style="{ width: `${fig.driverPct}%` }"></span>
        </div>
        <dl class="space-y-1 text-xs text-slate-400">
          <div class="flex items-center justify-between gap-2">
            <dt class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full bg-slate-300"></span>{{ $t('carpool.carpoolTripModal.passengersShares') }}</dt>
            <dd class="font-semibold tabular-nums text-slate-200">{{ fmt(fig.passengersShare) }}</dd>
          </div>
          <div class="flex items-center justify-between gap-2">
            <dt class="flex items-center gap-1.5"><span class="h-2 w-2 rounded-full bg-slate-600"></span>{{ $t('carpool.carpoolTripModal.driverSShare') }}</dt>
            <dd class="font-semibold tabular-nums text-slate-200">{{ fmt(fig.driverShare) }}</dd>
          </div>
        </dl>
      </div>

      <div class="space-y-1.5">
        <div class="flex items-baseline justify-between text-xs">
          <span class="text-slate-400">{{ $t('carpool.carpoolTripModal.received') }}</span>
          <strong class="tabular-nums text-slate-100">{{ fmt(fig.received) }} · {{ pct(fig.receivedPct) }}</strong>
        </div>
        <div class="relative h-2 rounded-full bg-slate-800" role="img" :aria-label="$t('carpool.editor.receivedLabel', { received: pct(fig.receivedPct), fair: pct(fig.passengersPct) })">
          <span class="absolute inset-y-0 left-0 rounded-full bg-slate-200" :style="{ width: `${Math.min(fig.receivedPct, 100)}%` }"></span>
          <span class="absolute -inset-y-1 w-0.5 rounded bg-rose-400" :style="{ left: `${Math.min(fig.passengersPct, 100)}%` }" :title="$t('carpool.editor.fairTick')"></span>
        </div>
        <p class="text-xs text-slate-400">{{ $t('carpool.editor.fairTickHint', { fair: pct(fig.passengersPct) }) }}</p>
      </div>

      <button v-if="copyLabel" type="button" class="btn btn-secondary w-full justify-center" @click="emit('copy')">
        <component :is="copied ? Check : Copy" class="h-4 w-4" aria-hidden="true" />
        {{ copied ? $t('carpool.carpoolDetailModal.copied') : copyLabel }}
      </button>
    </div>
  </section>
</template>
