<script setup lang="ts">
import { Plus, Radio, UploadCloud } from 'lucide-vue-next'
import { useVehicleStore } from '@/stores/vehicle'
import type { ExpensesTab } from '@/composables/useExpensesTabs'

// The page header buttons: which ones show depends on the active tab. The page opens the matching modal.
defineProps<{ activeTab: ExpensesTab }>()
defineEmits<{
  addToll: []
  addMaintenance: []
  openWebhook: []
  addReminder: []
  importCsv: []
  addCharge: []
  uploadDocument: []
}>()
const vehicleStore = useVehicleStore()
</script>

<template>
  <div v-if="vehicleStore.canEdit" class="flex items-center gap-2">
    <button
      v-if="activeTab === 'TOLLS'"
      @click="$emit('addToll')"
      class="btn btn-lg btn-primary"
    >
      <Plus class="w-3.5 h-3.5" />
      {{ $t('expenses.expensesView.add') }}
      <span class="hidden font-normal sm:inline">{{ $t('expenses.expensesView.tollParking') }}</span>
    </button>
    <button
      v-if="activeTab === 'MAINTENANCE' || activeTab === 'FIXED'"
      @click="$emit('addMaintenance')"
      class="btn btn-lg btn-primary"
    >
      <Plus class="w-3.5 h-3.5" />
      {{ $t('expenses.expensesView.add') }}
      <span class="hidden font-normal sm:inline">{{ activeTab === 'FIXED' ? $t('expenses.expensesView.fixedCosts') : $t('expenses.expensesView.maintenance') }}</span>
    </button>
    <button
      v-if="activeTab === 'REMINDERS'"
      @click="$emit('openWebhook')"
      class="btn btn-lg btn-secondary"
      :title="$t('expenses.expensesView.setUpTheWebhookTo')"
    >
      <Radio class="w-3.5 h-3.5" />
      <span class="hidden sm:inline">{{ $t('expenses.expensesView.homelabWebhook') }}</span>
      <span class="sm:hidden">{{ $t('expenses.expensesView.webhook') }}</span>
    </button>
    <button
      v-if="activeTab === 'REMINDERS'"
      @click="$emit('addReminder')"
      class="btn btn-lg btn-primary"
    >
      <Plus class="w-3.5 h-3.5" />
      {{ $t('expenses.expensesView.newReminder') }}
    </button>
    <button
      v-if="activeTab === 'FUEL'"
      @click="$emit('importCsv')"
      class="btn btn-lg btn-secondary"
    >
      <UploadCloud class="w-3.5 h-3.5" />
      <span>{{ $t('expenses.expensesView.importCsv') }}</span>
    </button>
    <template v-if="activeTab === 'CHARGES' && vehicleStore.canCharge">
      <button
        @click="$emit('importCsv')"
        class="btn btn-lg btn-secondary"
      >
        <UploadCloud class="w-3.5 h-3.5" />
        <span>{{ $t('expenses.expensesView.importCsv') }}</span>
      </button>
      <button
        @click="$emit('addCharge')"
        class="btn btn-lg btn-primary"
      >
        <Plus class="w-3.5 h-3.5" />
        {{ $t('expenses.expensesView.addCharge') }}
      </button>
    </template>
    <button
      v-if="activeTab === 'DOCUMENTS'"
      @click="$emit('uploadDocument')"
      class="btn btn-lg btn-primary"
    >
      <Plus class="w-3.5 h-3.5" />
      {{ $t('expenses.expensesView.addAReceipt') }}
    </button>
  </div>
</template>
