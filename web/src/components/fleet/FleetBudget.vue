<script setup lang="ts">
import { computed, ref } from 'vue'
import { api, type FleetSummaryResponse } from '@/services/api'
import { formatAmount } from '@/currency'
import { budgetUsage, parseBudgetInput } from '@/utils/fleetBudget'
import { useConfirm } from '@/composables/useConfirm'
import { t } from '@/i18n'

// The household's monthly budget: set, changed or removed here, compared with the month's cost in the fleet currency.
const props = defineProps<{ summary: FleetSummaryResponse }>()
const emit = defineEmits<{ saved: [budget: { monthly_budget: number | null; budget_currency: string | null }] }>()
const { showAlert } = useConfirm()

const editingBudget = ref(false)
const budgetInput = ref('')
const savingBudget = ref(false)
// A budget is compared with the month's cost only in its own currency: it is not converted when the first vehicle changes
const budgetInFleetCurrency = computed(() => !!props.summary.monthly_budget && props.summary.budget_currency === props.summary.currency)
const usage = computed(() => (props.summary.monthly_budget && budgetInFleetCurrency.value ? budgetUsage(props.summary.current_month_cost, props.summary.monthly_budget) : null))
// Vehicles in a currency other than the fleet one are not part of its totals, budget or chart
const otherCurrencyCount = computed(() => props.summary.totals.filter((x) => x.currency !== props.summary.currency).reduce((n, x) => n + x.vehicle_count, 0))

function startEditBudget() {
  budgetInput.value = props.summary.monthly_budget ? String(props.summary.monthly_budget) : ''
  editingBudget.value = true
}

async function saveBudget(amount: number | null) {
  savingBudget.value = true
  try {
    emit('saved', await api.setFleetBudget(amount))
    editingBudget.value = false
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    savingBudget.value = false
  }
}

function submitBudget() {
  const amount = parseBudgetInput(budgetInput.value)
  if (amount === null) {
    void showAlert(t('fleet.budget.invalid'), t('shell.confirm.error'), 'danger')
    return
  }
  void saveBudget(amount)
}
</script>

<template>
  <div class="bg-slate-900 border border-slate-800 rounded-xl p-4 shadow-sm space-y-3">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <span class="text-xs font-medium text-slate-400">{{ t('fleet.budget.title') }}</span>
      <div v-if="!editingBudget" class="flex items-center gap-3 text-xs font-semibold">
        <button type="button" @click="startEditBudget" class="tap-text px-2 text-rose-400 hover:text-rose-300">
          {{ summary.monthly_budget ? t('fleet.budget.edit') : t('fleet.budget.set') }}
        </button>
        <button v-if="summary.monthly_budget" type="button" :disabled="savingBudget" @click="saveBudget(null)" class="tap-text text-slate-400 hover:text-white">
          {{ t('fleet.budget.remove') }}
        </button>
      </div>
    </div>
    <form v-if="editingBudget" class="flex flex-wrap items-center gap-2" @submit.prevent="submitBudget">
      <input
        v-model="budgetInput"
        type="text"
        inputmode="decimal"
        :aria-label="t('fleet.budget.title')"
        :placeholder="t('fleet.budget.placeholder', { currency: summary.currency })"
        class="field flex-1"
      />
      <button type="submit" :disabled="savingBudget" class="btn btn-lg btn-primary">{{ t('common.save') }}</button>
      <button type="button" @click="editingBudget = false" class="btn btn-lg btn-secondary">{{ t('common.cancel') }}</button>
    </form>
    <p v-if="summary.monthly_budget && !budgetInFleetCurrency && !editingBudget" class="text-xs text-slate-300 break-words" data-testid="fleet-budget-stale-currency">
      {{ t('fleet.budget.staleCurrency', { budget: formatAmount(summary.monthly_budget, summary.budget_currency ?? ''), currency: summary.currency }) }}
    </p>
    <template v-else-if="usage && summary.monthly_budget && !editingBudget">
      <div class="h-2 rounded-full bg-slate-800 overflow-hidden" role="progressbar" :aria-valuenow="Math.round(usage.percent)" aria-valuemin="0" aria-valuemax="100" :aria-label="t('fleet.budget.title')">
        <div
          class="h-full rounded-full"
          :class="usage.status === 'over' ? 'bg-rose-500' : usage.status === 'warning' ? 'bg-warning-400' : 'bg-success-500'"
          :style="{ width: `${Math.min(100, usage.percent)}%` }"
        ></div>
      </div>
      <p class="text-xs text-slate-300 break-words">
        {{ t('fleet.budget.spent', { spent: formatAmount(summary.current_month_cost, summary.currency), budget: formatAmount(summary.monthly_budget, summary.currency), percent: Math.round(usage.percent) }) }}
        <span :class="usage.status === 'over' ? 'text-rose-400' : 'text-slate-400'">
          · {{ usage.status === 'over' ? t('fleet.budget.over', { amount: formatAmount(-usage.remaining, summary.currency) }) : t('fleet.budget.remaining', { amount: formatAmount(usage.remaining, summary.currency) }) }}
        </span>
      </p>
    </template>
    <p v-else-if="!editingBudget" class="text-xs text-slate-400">{{ t('fleet.budget.none') }}</p>
    <p v-if="otherCurrencyCount > 0" class="text-xs text-slate-400" data-testid="fleet-budget-other-currency">
      {{ t('fleet.budget.otherCurrency', { currency: summary.currency, count: otherCurrencyCount }, otherCurrencyCount) }}
    </p>
  </div>

</template>
