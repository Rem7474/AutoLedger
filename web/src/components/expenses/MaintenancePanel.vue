<script setup lang="ts">
import ListSkeleton from '@/components/ListSkeleton.vue'
import EmptyState from '@/components/EmptyState.vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import ListSummary from '@/components/ListSummary.vue'
import CostHistory from '@/components/costs/CostHistory.vue'
import { intlLocale } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import { computed, ref } from 'vue'
import { Repeat, Plus, Trash2, Paperclip, FileDown, Landmark, Wrench } from 'lucide-vue-next'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { saveBlob } from '@/utils/download'
import { t } from '@/i18n'
import { categoryLabel, categoryStyle, expensesOfKind, filterMaintenance, formatDate, maintenanceYears } from '@/utils/expenses'
import { formatAmount } from '@/currency'
import { distanceUnit, formatDistanceValue } from '@/units'
import BulkSelectionBar from '@/components/BulkSelectionBar.vue'
import RowSelectBox from '@/components/RowSelectBox.vue'
import SelectionHint from '@/components/SelectionHint.vue'
import { useRowSelection } from '@/composables/useRowSelection'
import { editDeleteActions, type SelectionAction } from '@/utils/selectionActions'

const props = defineProps<{ vehicleId: string; maintenanceExpenses: any[]; loading: boolean; kind: 'service' | 'fixed' }>()
const emit = defineEmits<{
  add: []
  edit: [expense: any]
  delete: [expense: any]
  'delete-many': [expenses: any[]]
  'view-document': [docId: string | null | undefined, filename?: string | null, download?: boolean]
}>()
const vehicleStore = useVehicleStore()
const { selected, selectedIds, clear, isSelected, toggle, showBox, press, onPointerdown, onRowClick } = useRowSelection(
  computed(() => props.maintenanceExpenses),
  computed(() => vehicleStore.canEdit),
  (m) => emit('edit', m),
)
const actions = computed<SelectionAction[]>(() => editDeleteActions(selected.value, { edit: (m) => emit('edit', m), remove: (items) => emit('delete-many', items) }))
const { showAlert } = useConfirm()
const bookWithProofs = ref(false)
const bookLoading = ref(false)

// The service book follows the year filter; the category filter does not apply to it.
async function downloadServiceBook() {
  bookLoading.value = true
  try {
    const year = filter.value.year
    const { blob, filename } = await api.downloadServiceBook(props.vehicleId, {
      from: year ? `${year}-01-01` : undefined,
      to: year ? `${year}-12-31` : undefined,
      attachments: bookWithProofs.value,
    })
    saveBlob(blob, filename)
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    bookLoading.value = false
  }
}

const filter = ref({ category: '', year: '' })
const ofKind = computed(() => expensesOfKind(props.maintenanceExpenses, props.kind))
const categories = computed(() => [...new Set(ofKind.value.map((m) => m.category as string))])
const years = computed(() => maintenanceYears(ofKind.value))
const visible = computed(() => filterMaintenance(ofKind.value, filter.value))
</script>

<template>
  <div>
    <ListSkeleton v-if="loading" />
    <EmptyState
      v-else-if="!ofKind.length"
      :icon="kind === 'fixed' ? Landmark : Wrench"
      :title="kind === 'fixed' ? $t('expenses.maintenancePanel.noFixedExpense') : $t('expenses.maintenancePanel.noMaintenanceExpense')"
    >
      <template v-if="kind === 'fixed'">{{ $t('expenses.maintenancePanel.noFixedExpenseHint') }}</template>
      <template v-if="vehicleStore.canEdit" #actions>
        <button class="btn btn-primary" @click="emit('add')">
          <Plus class="h-3.5 w-3.5" /> {{ $t('expenses.expensesView.add') }}
        </button>
      </template>
    </EmptyState>
    <div v-else class="space-y-3">
      <BulkSelectionBar v-if="vehicleStore.canEdit" :count="selectedIds.length" :actions="actions" @clear="clear" />
      <SelectionHint v-if="vehicleStore.canEdit" />
      <CostHistory :vehicle-id="vehicleId" :kind="kind" :currency="vehicleStore.currency" :sync-key="maintenanceExpenses" />
      <div class="grid grid-cols-2 items-center gap-2 sm:flex sm:flex-wrap">
        <label class="sr-only" for="maint-filter-category">{{ $t('expenses.maintenancePanel.categoryFilter') }}</label>
        <select id="maint-filter-category" v-model="filter.category" class="field !w-auto">
          <option value="">{{ $t('expenses.maintenancePanel.allCategories') }}</option>
          <option v-for="c in categories" :key="c" :value="c">{{ categoryLabel(c) }}</option>
        </select>
        <label class="sr-only" for="maint-filter-year">{{ $t('expenses.maintenancePanel.yearFilter') }}</label>
        <select id="maint-filter-year" v-model="filter.year" class="field !w-auto">
          <option value="">{{ $t('expenses.maintenancePanel.allYears') }}</option>
          <option v-for="y in years" :key="y" :value="String(y)">{{ y }}</option>
        </select>
      </div>
      <ListSummary :chips="[]">
        {{ $t('expenses.maintenancePanel.count', { n: visible.length }, visible.length) }}
      </ListSummary>
      <div v-if="kind === 'service'" class="flex flex-wrap items-center gap-x-4 gap-y-2 rounded-2xl border border-slate-800 bg-slate-900 px-4 py-3">
        <button
          type="button"
          class="px-3 py-2 bg-slate-800 hover:bg-slate-700 text-white text-xs font-semibold rounded-xl border border-slate-700 inline-flex items-center gap-2 disabled:opacity-50"
          :disabled="bookLoading"
          @click="downloadServiceBook"
        >
          <FileDown class="w-4 h-4" />
          {{ $t('expenses.maintenancePanel.serviceBook') }}
        </button>
        <label class="flex items-center gap-2 text-xs text-slate-300 cursor-pointer">
          <input v-model="bookWithProofs" type="checkbox" class="select-box" />
          {{ $t('expenses.maintenancePanel.serviceBookWithProofs') }}
        </label>
        <p class="text-xs text-slate-400 basis-full">{{ $t('expenses.maintenancePanel.serviceBookHint') }}</p>
      </div>
      <EmptyState v-if="!visible.length">{{ $t('expenses.maintenancePanel.noMatch') }}</EmptyState>
      <div
        v-for="m in visible"
        :key="m.id"
        class="relative bg-slate-900 border border-slate-800 hover:border-slate-700/90 border-l-4 p-4 rounded-2xl flex items-center justify-between gap-3 transition-colors max-sm:select-none max-sm:[-webkit-touch-callout:none]"
        :class="[categoryStyle(m.category).accent, { '!border-rose-500/40 bg-slate-800/40': isSelected(m) }]"
      >
        <CardOpenButton
          v-if="vehicleStore.canEdit"
          :label="$t('expenses.maintenancePanel.editThisExpense')"
          @click="onRowClick(m)"
          @pointerdown="onPointerdown($event, m)"
          @pointermove="press.onPointermove"
          @pointerup="press.onPointerup"
          @pointercancel="press.onPointercancel"
          @contextmenu="press.onContextmenu"
        />
        <RowSelectBox v-if="showBox" :id="m.id" :checked="isSelected(m)" :label="$t('shell.bulkSelectionBar.selectRow')" @toggle="toggle(m)" />
        <div class="space-y-1 min-w-0 flex-1">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="text-xs px-2 py-0.5 rounded-full font-bold border shrink-0" :class="categoryStyle(m.category).badge">
              {{ categoryLabel(m.category) }}
            </span>
            <span class="text-xs text-slate-400 shrink-0">{{ formatDate(m.date) }}</span>
            <span v-if="m.odometer" class="text-xs text-slate-400 shrink-0">· {{ $t('common.atKmCapitalized', { unit: distanceUnit(), km: formatDistanceValue(m.odometer) }) }}</span>
            <span v-if="m.is_recurring" class="text-xs text-slate-400 flex items-center gap-1 shrink-0">
              <Repeat class="w-3 h-3 text-slate-400" /> {{ $t('expenses.maintenancePanel.everyMonths', { recurrence_interval_months: m.recurrence_interval_months }) }}
              <template v-if="m.recurrence_end_date">{{ $t('expenses.maintenancePanel.until', { recurrence_end_date: formatDate(m.recurrence_end_date) }) }}</template>
            </span>
            <span v-else-if="m.amortization_mode === 'DISTANCE'" class="text-xs px-2 py-0.5 rounded-full font-medium bg-success-500/10 text-success-400 border border-success-500/20 shrink-0">
              {{ $t('expenses.maintenancePanel.smoothedOverKm', { unit: distanceUnit(), coverage_km: m.coverage_km ? formatDistanceValue(m.coverage_km) : formatDistanceValue(50000) }) }}
            </span>
            <span v-else-if="m.amortization_mode === 'DURATION'" class="text-xs px-2 py-0.5 rounded-full font-medium bg-sky-500/10 text-sky-400 border border-sky-500/20 shrink-0">
              {{ $t('expenses.maintenancePanel.smoothedOverMonths', { coverage_months: m.coverage_months || 24 }) }}
            </span>
            <span v-else-if="m.amortization_mode === 'HYBRID'" class="text-xs px-2 py-0.5 rounded-full font-medium bg-sky-500/10 text-sky-400 border border-sky-500/20 shrink-0">
              {{ $t('expenses.maintenancePanel.mixedSmoothingKmMonths', { unit: distanceUnit(), coverage_km: m.coverage_km ? formatDistanceValue(m.coverage_km) : formatDistanceValue(50000), coverage_months: m.coverage_months || 24 }) }}
            </span>
            <span v-if="m.closes_maintenance_id" class="text-xs px-2 py-0.5 rounded-full font-medium bg-slate-800 text-slate-300 border border-slate-700/60 shrink-0">
              {{ $t('expenses.maintenancePanel.closesThePreviousService') }}
            </span>
            <button
              v-if="m.document_id"
              @click="emit('view-document', m.document_id, m.document_filename, false)"
              class="relative z-10 text-xs px-2 py-0.5 rounded-lg bg-sky-500/10 hover:bg-sky-500/20 text-sky-400 border border-sky-500/20 flex items-center gap-1 transition-colors max-w-[200px] truncate"
              :title="$t('expenses.maintenancePanel.viewTheReceipt')"
            >
              <Paperclip class="w-3 h-3 shrink-0" />
              <span class="truncate">{{ m.document_filename || $t('expenses.invoice') }}</span>
            </button>
          </div>
          <p class="text-sm font-semibold text-slate-200 truncate">{{ m.description }}</p>
        </div>
        <div class="flex items-center justify-end gap-2 shrink-0">
          <div class="text-lg font-extrabold text-white">
            {{ formatAmount(m.amount, m.currency || vehicleStore.currency) }}
          </div>
          <RowMenu v-if="vehicleStore.canEdit" :label="$t('common.moreActions')" :heading="formatDate(m.date)">
            <button type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', m)">
              <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
            </button>
          </RowMenu>
        </div>
      </div>
    </div>
  </div>
</template>
