<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { Receipt } from '@/icons'
import { api, type TollInvoiceMonth } from '@/services/api'
import { formatAmount } from '@/currency'
import { intlLocale, t } from '@/i18n'
import { gapKind, listedInvoiceMonths, monthDate } from '@/utils/tollInvoices'

// The monthly toll invoices on the Costs page, under the Tolls filter: for each recent month, what was invoiced or what the
// trips recorded, and the gap between the two. Tapping a month opens its invoice form.
const props = defineProps<{ vehicleId: string; currency: string; canEdit: boolean; syncKey: number; hiddenMonths?: string[] }>()
const emit = defineEmits<{ edit: [row: TollInvoiceMonth] }>()

const months = ref<TollInvoiceMonth[] | null>(null)
const failed = ref(false)
const showAll = ref(false)

async function load() {
  failed.value = false
  try {
    months.value = (await api.getTollInvoices(props.vehicleId, 12)).months
  } catch (err) {
    console.error('Failed to load the toll invoices', err)
    failed.value = true
  }
}
watch(() => [props.vehicleId, props.syncKey] as const, () => void load(), { immediate: true })

// A month whose invoice is being deleted (Undo pending) reads as without one
const shown = computed(() => (months.value ?? []).map((m) => (m.invoice && props.hiddenMonths?.includes(m.month) ? { ...m, invoice: null } : m)))
const listed = computed(() => listedInvoiceMonths(shown.value, showAll.value))
const label = (m: string) => new Intl.DateTimeFormat(intlLocale(), { month: 'long', year: 'numeric' }).format(monthDate(m))
const gapText = (row: TollInvoiceMonth) => {
  if (!row.invoice) return ''
  return t(`costs.tollInvoice.gap.${gapKind(row.invoice.gap)}`, { amount: formatAmount(Math.abs(row.invoice.gap), props.currency) })
}
</script>

<template>
  <section v-if="months !== null || failed" class="space-y-2 rounded-2xl border border-slate-700/60 bg-slate-800/40 p-3.5" data-testid="toll-invoices-card">
    <header class="flex items-center gap-2">
      <Receipt class="size-4 text-warning-400" aria-hidden="true" />
      <h2 class="text-sm font-bold text-white">{{ $t('costs.tollInvoice.title') }}</h2>
    </header>
    <p v-if="failed" class="text-xs text-slate-400">{{ $t('costs.tollInvoice.loadError') }}</p>
    <template v-else>
      <p class="text-xs text-slate-400">{{ $t('costs.tollInvoice.hint') }}</p>
      <ul class="divide-y divide-slate-700/50">
        <li v-for="row in listed.rows" :key="row.month">
          <button
            type="button"
            class="flex min-h-11 w-full items-start justify-between gap-3 py-2.5 text-left"
            :data-testid="`toll-invoice-${row.month}`"
            :disabled="!canEdit && !row.invoice"
            @click="emit('edit', row)"
          >
            <span class="min-w-0">
              <span class="block text-sm font-semibold capitalize text-white">{{ label(row.month) }}</span>
              <span class="block text-xs text-slate-400">
                <template v-if="row.invoice">{{ gapText(row) }}</template>
                <template v-else-if="row.trip_tolls > 0">{{ $t('costs.tollInvoice.tripTolls', { amount: formatAmount(row.trip_tolls, currency) }) }}</template>
                <template v-else>{{ $t('costs.tollInvoice.noTolls') }}</template>
              </span>
            </span>
            <span class="shrink-0 text-right">
              <template v-if="row.invoice">
                <span class="block text-sm font-bold tabular-nums text-white">{{ formatAmount(row.invoice.total, currency) }}</span>
                <span class="block text-xs text-slate-400">{{ $t('costs.tollInvoice.invoiced') }}</span>
              </template>
              <span v-else-if="canEdit" class="text-xs font-semibold text-info-400">{{ $t('costs.tollInvoice.enter') }}</span>
            </span>
          </button>
        </li>
      </ul>
      <button v-if="listed.hidden > 0" type="button" class="tap-text text-xs font-semibold text-info-400" @click="showAll = true">
        {{ $t('costs.tollInvoice.showMore', { count: listed.hidden }) }}
      </button>
      <button v-else-if="showAll" type="button" class="tap-text text-xs font-semibold text-info-400" @click="showAll = false">
        {{ $t('costs.tollInvoice.showLess') }}
      </button>
    </template>
  </section>
</template>
