<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import ModalShell from '@/components/ModalShell.vue'
import NumberInput from '@/components/NumberInput.vue'
import { Receipt } from '@/icons'
import DetailDeleteButton from '@/components/DetailDeleteButton.vue'
import { api, type TollInvoiceMonth } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useStaleWrite } from '@/composables/useStaleWrite'
import { useSubmit } from '@/composables/useSubmit'
import { currencySymbol } from '@/currency'
import { formatAmount } from '@/currency'
import { intlLocale, t } from '@/i18n'
import { gapKind, gapOfTotal, monthDate } from '@/utils/tollInvoices'

// Enters the toll invoice of a month, or edits or deletes it when the month has one. In the totals the invoice takes the
// place of the tolls of the trips of its month; the form says what the trips add up to and what the gap would be.
const props = defineProps<{ vehicleId: string; row: TollInvoiceMonth | null; currency: string; canEdit: boolean }>()
const emit = defineEmits<{ saved: []; delete: [row: TollInvoiceMonth] }>()
const open = defineModel<boolean>('open', { required: true })
const { showAlert } = useConfirm()
const { explainStaleWrite } = useStaleWrite()
const { pending: submitting, run: runOnce } = useSubmit()

const total = ref('')
const note = ref('')
const invoice = computed(() => props.row?.invoice ?? null)
const tripTolls = computed(() => props.row?.trip_tolls ?? 0)
const monthLabel = computed(() =>
  props.row ? new Intl.DateTimeFormat(intlLocale(), { month: 'long', year: 'numeric' }).format(monthDate(props.row.month)) : ''
)
const gap = computed(() => gapOfTotal(total.value, tripTolls.value))
const gapText = computed(() => {
  if (gap.value === null) return ''
  return t(`costs.tollInvoice.preview.${gapKind(gap.value)}`, { amount: formatAmount(Math.abs(gap.value), props.currency) })
})

watch(open, (isOpen) => {
  if (!isOpen) return
  total.value = invoice.value ? String(invoice.value.total) : ''
  note.value = invoice.value?.note ?? ''
})

const fail = (err: any) => void showAlert(t('common.errorWithMessage', { message: err?.message ?? '' }), t('shell.confirm.error'), 'danger')

async function saveAction() {
  if (!props.row) return
  try {
    await api.saveTollInvoice(props.vehicleId, props.row.month, {
      total: Number(total.value),
      note: note.value.trim() || null,
      ...(invoice.value ? { expected_version: invoice.value.version } : {}),
    })
    open.value = false
    emit('saved')
  } catch (err: any) {
    if (await explainStaleWrite(err)) {
      open.value = false
      emit('saved')
      return
    }
    fail(err)
  }
}
const save = () => runOnce(saveAction)

/** The page deletes with an Undo and closes the form. */
function remove() {
  if (!props.row || !invoice.value) return
  open.value = false
  emit('delete', props.row)
}
</script>

<template>
  <ModalShell
    pane
    v-model:open="open"
    :title="invoice ? $t('costs.tollInvoice.edit', { month: monthLabel }) : $t('costs.tollInvoice.add', { month: monthLabel })"
    :icon="Receipt"
    icon-class="text-warning-400"
  >
    <form id="toll-invoice-form" class="space-y-4" @submit.prevent="save">
      <p class="text-xs text-slate-400">{{ $t('costs.tollInvoice.intro') }}</p>
      <div>
        <label for="toll-invoice-total" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('costs.tollInvoice.total', { currency: currencySymbol(currency) }) }}</label>
        <NumberInput text id="toll-invoice-total" v-model="total" min="0" required placeholder="0.00" class="field" :disabled="!canEdit" />
        <p class="mt-1 text-xs text-slate-500">{{ $t('costs.tollInvoice.totalHelp') }}</p>
      </div>
      <div class="rounded-xl border border-slate-700/60 bg-slate-800/50 p-3.5 text-xs space-y-1" data-testid="toll-invoice-trips">
        <p class="flex justify-between gap-3 text-slate-300">
          <span>{{ $t('costs.tollInvoice.onTrips') }}</span>
          <span class="font-semibold tabular-nums text-white">{{ formatAmount(tripTolls, currency) }}</span>
        </p>
        <p v-if="gapText" class="text-slate-400" data-testid="toll-invoice-gap">{{ gapText }}</p>
      </div>
      <div>
        <label for="toll-invoice-note" class="mb-1 block text-xs font-semibold text-slate-300">{{ $t('costs.tollInvoice.note') }}</label>
        <input id="toll-invoice-note" v-model="note" :placeholder="$t('costs.tollInvoice.notePlaceholder')" maxlength="500" class="field" :disabled="!canEdit" />
      </div>
    </form>
    <template #footer>
      <DetailDeleteButton v-if="invoice" class="mr-auto" :can-edit="canEdit" @delete="remove" />
      <button type="button" class="btn btn-lg btn-secondary" @click="open = false">{{ $t('common.cancel') }}</button>
      <button v-if="canEdit" :disabled="submitting" type="submit" form="toll-invoice-form" class="btn btn-lg btn-primary">
        {{ invoice ? $t('expenses.update') : $t('common.save') }}
      </button>
    </template>
  </ModalShell>
</template>
