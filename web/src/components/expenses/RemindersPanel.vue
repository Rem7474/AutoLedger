<script setup lang="ts">
import ListSkeleton from '@/components/ListSkeleton.vue'
import { computed, ref } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import type { MaintenanceReminder, VehicleWebhook } from '@/services/api'
import { Plus, AlertTriangle, Bell, Clock, CheckCircle2, Radio, Sparkles, CircleDashed, Settings } from 'lucide-vue-next'
import EmptyState from '@/components/EmptyState.vue'
import ListSummary from '@/components/ListSummary.vue'
import { reminderPresets, type ReminderPreset } from '@/utils/expenses'
import { groupRemindersByDue } from '@/utils/dashboard'
import ReminderRow from '@/components/expenses/ReminderRow.vue'
import ReminderTemplatesBar from '@/components/expenses/ReminderTemplatesBar.vue'

const props = defineProps<{
  reminders: MaintenanceReminder[]
  overdueReminders: MaintenanceReminder[]
  dueSoonReminders: MaintenanceReminder[]
  okReminders: MaintenanceReminder[]
  unscheduledReminders: MaintenanceReminder[]
  loadingReminders: boolean
  vehicleWebhook: VehicleWebhook | null
}>()
const emit = defineEmits<{
  'open-webhook': []
  add: [preset: ReminderPreset | null]
  edit: [reminder: MaintenanceReminder]
  complete: [reminder: MaintenanceReminder]
  delete: [reminder: MaintenanceReminder]
  reload: []
}>()
const vehicleStore = useVehicleStore()
const showSettings = ref(false)
const groups = computed(() => groupRemindersByDue(props.reminders))
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-2">
      <ListSummary
        v-if="reminders.length"
        role="group"
        :aria-label="$t('expenses.remindersPanel.summary')"
        :chips="[
          ...(overdueReminders.length ? [{ label: $t('expenses.remindersPanel.overdue'), value: String(overdueReminders.length), tone: 'rose' as const }] : []),
          ...(dueSoonReminders.length ? [{ label: $t('expenses.remindersPanel.comingUp'), value: String(dueSoonReminders.length), tone: 'amber' as const }] : []),
          ...(okReminders.length ? [{ label: $t('expenses.remindersPanel.upToDate'), value: String(okReminders.length), tone: 'emerald' as const }] : []),
          ...(unscheduledReminders.length ? [{ label: $t('expenses.remindersPanel.groupUnscheduled'), value: String(unscheduledReminders.length) }] : []),
        ]"
      >
        {{ $t('expenses.remindersPanel.totalCount', { count: reminders.length }, reminders.length) }}
      </ListSummary>
      <span v-else></span>
      <button
        type="button"
        class="btn btn-secondary tap-text"
        :aria-expanded="showSettings"
        @click="showSettings = !showSettings"
      >
        <Settings class="w-3.5 h-3.5" />
        {{ $t('expenses.remindersPanel.settings') }}
        <span v-if="vehicleWebhook?.enabled" class="w-1.5 h-1.5 rounded-full bg-emerald-400" :title="$t('expenses.remindersPanel.active', { type: vehicleWebhook.type })"></span>
      </button>
    </div>

    <div v-if="showSettings" class="space-y-4">
      <div class="p-4 bg-slate-900 border border-slate-800 rounded-2xl flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs">
        <div class="flex items-center gap-3">
          <div class="p-2 rounded-xl bg-sky-500/10 border border-sky-500/20 text-sky-400 shrink-0">
            <Radio class="w-5 h-5" />
          </div>
          <div>
            <div class="font-bold text-white flex items-center gap-2">
              <span>{{ $t('expenses.remindersPanel.homelabWebhook') }}</span>
              <span
                class="px-2 py-0.5 text-xs rounded-full font-bold border"
                :class="vehicleWebhook?.enabled ? 'bg-success-500/10 text-success-400 border-success-500/20' : 'bg-slate-800 text-slate-400 border-slate-700'"
              >
                {{ vehicleWebhook?.enabled ? $t('expenses.remindersPanel.active', { type: vehicleWebhook.type }) : $t('expenses.remindersPanel.notConfigured') }}
              </span>
            </div>
            <p class="text-slate-400 text-xs mt-0.5">
              {{ vehicleWebhook?.enabled ? $t('expenses.remindersPanel.alertsSent') : $t('expenses.remindersPanel.alertsHint') }}
            </p>
          </div>
        </div>
        <button
          v-if="vehicleStore.canEdit"
          @click="emit('open-webhook')"
          class="btn btn-secondary shrink-0 self-start sm:self-auto"
        >
          {{ vehicleWebhook ? $t('expenses.remindersPanel.editWebhook') : $t('expenses.remindersPanel.setUpWebhook') }}
        </button>
      </div>

      <ReminderTemplatesBar v-if="vehicleStore.canEdit" :has-reminders="reminders.length > 0" @changed="emit('reload')" />
    </div>

    <ListSkeleton v-if="loadingReminders" :label="$t('expenses.remindersPanel.loadingTheReminders')" />

    <!-- Empty state -->
    <EmptyState v-else-if="!reminders.length" :icon="Bell" :title="$t('expenses.remindersPanel.noMaintenanceReminderConfigured')">
      {{ $t('expenses.remindersPanel.trackTireWearCabinFilter') }}
      <template v-if="vehicleStore.canEdit" #actions>
        <button @click="emit('add', null)" class="btn btn-primary">
          <Plus class="w-4 h-4" />
          {{ $t('expenses.remindersPanel.createACustomReminder') }}
        </button>
        <div class="pt-2">
          <p class="text-xs font-semibold text-slate-300 mb-3">{{ $t('expenses.remindersPanel.addAStandardReminderIn') }}</p>
          <div class="flex flex-wrap justify-center gap-2 max-w-lg mx-auto">
            <button
              v-for="preset in reminderPresets()"
              :key="preset.title"
              type="button"
              @click="emit('add', preset)"
              class="btn btn-secondary"
            >
              <Sparkles class="w-3 h-3 text-sky-400" />
              {{ preset.title }}
            </button>
          </div>
        </div>
      </template>
    </EmptyState>

    <!-- Reminders by due date -->
    <div v-else class="space-y-5">
      <section v-for="g in groups" :key="g.key" class="space-y-2">
        <h4
          class="flex items-center gap-2 text-xs font-bold"
          :class="g.key === 'overdue' ? 'text-rose-300' : g.key === 'soon' ? 'text-warning-300' : 'text-slate-400'"
        >
          <AlertTriangle v-if="g.key === 'overdue'" class="w-3.5 h-3.5" />
          <Clock v-else-if="g.key === 'soon'" class="w-3.5 h-3.5" />
          <CircleDashed v-else-if="g.key === 'unscheduled'" class="w-3.5 h-3.5" />
          <CheckCircle2 v-else class="w-3.5 h-3.5" />
          {{ $t(`expenses.remindersPanel.group${g.key.charAt(0).toUpperCase()}${g.key.slice(1)}`) }}
          <span class="font-semibold opacity-70">{{ g.items.length }}</span>
        </h4>
        <ReminderRow
          v-for="r in g.items"
          :key="r.id"
          :reminder="r"
          @edit="emit('edit', $event)"
          @complete="emit('complete', $event)"
          @delete="emit('delete', $event)"
        />
      </section>
    </div>
  </div>
</template>
