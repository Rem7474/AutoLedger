<script setup lang="ts">
import ListSkeleton from '@/components/ListSkeleton.vue'
import EmptyState from '@/components/EmptyState.vue'
import CardOpenButton from '@/components/CardOpenButton.vue'
import RowMenu from '@/components/RowMenu.vue'
import { useRouter } from 'vue-router'
import { useVehicleStore } from '@/stores/vehicle'
import { Navigation, Layers, Users, Trash2, Paperclip, Plus, Receipt } from 'lucide-vue-next'
import ListSummary from '@/components/ListSummary.vue'
import CostHistory from '@/components/costs/CostHistory.vue'
import { formatDate } from '@/utils/expenses'
import { formatAmount } from '@/currency'
import { t } from '@/i18n'

defineProps<{ vehicleId: string; driveExpenses: any[]; loading: boolean }>()
const emit = defineEmits<{
  add: []
  edit: [expense: any]
  delete: [expense: any]
  'view-document': [docId: string | null | undefined, filename?: string | null, download?: boolean]
}>()
const router = useRouter()
const vehicleStore = useVehicleStore()

const TYPE_LABEL_KEYS: Record<string, string> = {
  TOLL: 'expenses.tollModal.toll',
  PARKING: 'expenses.tollModal.parking',
  FERRY: 'expenses.tollModal.ferry',
  OTHER: 'expenses.tollModal.other',
}
const expenseTypeLabel = (type: string) => (TYPE_LABEL_KEYS[type] ? t(TYPE_LABEL_KEYS[type]) : type)
</script>

<template>
  <div>
    <ListSkeleton v-if="loading" />
    <EmptyState v-else-if="!driveExpenses.length" :icon="Receipt" :title="$t('expenses.tollsPanel.noTollOrParkingRecorded')">
      {{ $t('expenses.tollsPanel.emptyHint') }}
      <template v-if="vehicleStore.canEdit" #actions>
        <button class="btn btn-primary" @click="emit('add')">
          <Plus class="h-3.5 w-3.5" /> {{ $t('expenses.tollsPanel.addFirst') }}
        </button>
      </template>
    </EmptyState>
    <div v-else class="space-y-3">
      <CostHistory :vehicle-id="vehicleId" kind="tolls" :currency="vehicleStore.currency" :sync-key="driveExpenses" />
      <ListSummary :chips="[]">
        {{ $t('expenses.tollsPanel.count', { count: driveExpenses.length }, driveExpenses.length) }}
      </ListSummary>
      <div
        v-for="e in driveExpenses"
        :key="e.id"
        class="relative bg-slate-900 border border-slate-800 hover:border-slate-700/90 p-4 rounded-2xl flex items-center justify-between gap-3 transition-colors"
      >
        <CardOpenButton v-if="vehicleStore.canEdit" :label="$t('expenses.tollsPanel.editThisToll')" @click="emit('edit', e)" />
        <div class="space-y-1.5 min-w-0 flex-1">
          <div class="flex items-center gap-2 flex-wrap">
            <span class="text-xs px-2 py-0.5 rounded-full font-bold bg-slate-800 text-slate-300 border border-slate-700/60 shrink-0">
              {{ expenseTypeLabel(e.type) }}
            </span>
            <span class="text-xs text-slate-400 shrink-0">{{ formatDate(e.date) }}</span>
            <span v-if="e.drive_title" class="text-xs px-2.5 py-0.5 rounded-lg bg-success-500/10 text-success-400 border border-success-500/20 flex items-center gap-1 truncate max-w-xs">
              <Navigation class="w-3 h-3 shrink-0" /> <span class="truncate">{{ e.drive_title }}</span>
            </span>
            <span v-else-if="e.trip_group_name" class="text-xs px-2.5 py-0.5 rounded-lg bg-sky-500/10 text-sky-400 border border-sky-500/20 flex items-center gap-1 truncate max-w-xs">
              <Layers class="w-3 h-3 shrink-0" /> <span class="truncate">{{ e.trip_group_name }}</span>
            </span>
            <button
              v-if="e.document_id"
              @click="emit('view-document', e.document_id, e.document_filename, false)"
              class="relative z-10 text-xs px-2 py-0.5 rounded-lg bg-sky-500/10 hover:bg-sky-500/20 text-sky-400 border border-sky-500/20 flex items-center gap-1 transition-colors max-w-[200px] truncate"
              :title="$t('expenses.tollsPanel.viewTheReceipt')"
            >
              <Paperclip class="w-3 h-3 shrink-0" />
              <span class="truncate">{{ e.document_filename || $t('expenses.invoice') }}</span>
            </button>
          </div>
          <p v-if="e.notes" class="text-sm text-slate-300">{{ e.notes }}</p>
        </div>
        <div class="flex items-center justify-end gap-2 shrink-0">
          <div class="text-lg font-extrabold text-white">
            {{ formatAmount(e.amount, e.currency || vehicleStore.currency) }}
          </div>
          <RowMenu v-if="vehicleStore.canEdit" :label="$t('common.moreActions')" :heading="formatDate(e.date)">
            <button
              v-if="e.drive_id || e.trip_group_id"
              type="button"
              role="menuitem"
              class="menu-item"
              :title="$t('expenses.tollsPanel.createACarpoolForThis')"
              @click="router.push({ path: '/carpools', query: e.drive_id ? { new_drive_id: e.drive_id } : { new_trip_group_id: e.trip_group_id } })"
            >
              <Users class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.tollsPanel.carpool') }}
            </button>
            <button type="button" role="menuitem" class="menu-item hover:!text-rose-300" @click="emit('delete', e)">
              <Trash2 class="w-3.5 h-3.5 text-slate-400" />{{ $t('common.delete') }}
            </button>
          </RowMenu>
        </div>
      </div>
    </div>
  </div>
</template>
