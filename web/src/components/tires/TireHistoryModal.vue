<script setup lang="ts">
import ModalShell from '@/components/ModalShell.vue'
import { intlLocale } from '@/i18n'
import { Archive, ChevronRight, ClipboardPaste, History, Pencil, Plus, Ruler, Zap } from '@/icons'
import DetailDeleteButton from '@/components/DetailDeleteButton.vue'
import StatusLabel from '@/components/StatusLabel.vue'
import TireSessionDetailModal from '@/components/tires/TireSessionDetailModal.vue'
import { computed, ref } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { apiMessageText } from '@/services/apiError'
import { formatAmount } from '@/currency'
import { formatDate, positionLabel, WEAR_TONE_TEXT, wearTone, type SessionForm } from '@/utils/tires'
import { formatNumber, formatPercent } from '@/utils/numbers'
import TireWearBar from '@/components/tires/TireWearBar.vue'
import { distanceUnit, formatDistance, formatDistanceValue, formatPerDistanceValue, perDistance } from '@/units'
import { formatCostPerDistance } from '@/utils/costPerDistance'

const vehicleStore = useVehicleStore()

// Full history of one tire: life KPIs, TeslaMate telemetry, mount sessions and tread depth measurements.
// It only displays: every action is emitted for the page to run.
const props = defineProps<{
  selectedTire: any | null
  selectedTireStats: any | null
  tireSessions: any[]
  tireLogs: any[]
  copiedSession: SessionForm | null
}>()
const emit = defineEmits<{
  'edit-tire': []
  'dispose-tire': []
  'delete-tire': []
  'paste-session': []
  'add-session': []
  'copy-session': [session: any]
  'duplicate-session': [session: any]
  'edit-session': [session: any]
  'delete-session': [session: any]
  'add-log': []
  'edit-log': [log: any]
}>()
const open = defineModel<boolean>('open', { required: true })

// The fitting whose detail is open; the distance of the one still on the vehicle is the live run of the tire.
const detail = ref<any | null>(null)
const detailOpen = computed({ get: () => detail.value !== null, set: (v) => { if (!v) detail.value = null } })
function sessionKm(s: any): number {
  return s.dismounted_date ? s.distance_km : props.selectedTireStats?.current_run_km ?? s.distance_km
}
</script>

<template>
  <ModalShell
    v-if="selectedTire"
    v-model:open="open"
    size="lg"
    body-class="space-y-6"
    footer-class="justify-end"
  >
    <template #title>
      <div>
        <div class="flex items-center gap-2">
          <h3 class="text-base font-bold text-white">{{ selectedTire.brand }} {{ selectedTire.model }}</h3>
          <span class="text-xs font-mono bg-slate-800 px-2 py-0.5 rounded text-slate-300">
            {{ selectedTire.dimension }}
          </span>
        </div>
        <div class="text-xs text-slate-400 mt-0.5">
          {{ $t('tires.tireHistoryModal.boughtOn', { purchase_date: formatDate(selectedTire.purchase_date), purchase_price: formatAmount(Number(selectedTire.purchase_price || 0), vehicleStore.currency) }) }}
        </div>
      </div>
    </template>
    <template #actions>
      <button @click="emit('edit-tire')" class="tap p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white rounded-lg transition-colors" :title="$t('tires.tireHistoryModal.editTheTire')" :aria-label="$t('tires.tireHistoryModal.editTheTire')">
        <Pencil class="w-4 h-4" />
      </button>
      <button
        v-if="selectedTire.current_position !== 'DISPOSED'"
        @click="emit('dispose-tire')"
        class="tap p-1.5 bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-warning-400 rounded-lg transition-colors"
        :title="$t('tires.tireHistoryModal.scrapWornPuncturedSold')" :aria-label="$t('tires.tireHistoryModal.scrapWornPuncturedSold')"
      >
        <Archive class="w-4 h-4" />
      </button>
    </template>

    <!-- Life KPI Card -->
    <div class="bg-slate-800 p-4 rounded-2xl space-y-3">
      <div class="space-y-0.5">
        <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.totalLifetimeMileage') }}</div>
        <div class="flex flex-wrap items-baseline gap-x-2">
          <span class="text-base font-bold text-white">{{ formatDistance(selectedTireStats?.total_distance_km || 0) }}</span>
          <span class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.kmEstimated', { unit: distanceUnit(), estimated_lifespan_km: formatDistanceValue(selectedTire.estimated_lifespan_km || 45000) }) }}</span>
        </div>
      </div>

      <TireWearBar thick :pct="selectedTireStats?.life_progress_pct || 0" :condition="selectedTireStats?.condition" />

      <div class="grid grid-cols-3 gap-2 text-center text-xs pt-1">
        <div class="bg-slate-900/80 p-2 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.currentTread') }}</div>
          <div class="font-bold" :class="WEAR_TONE_TEXT[wearTone(0, selectedTireStats?.condition)]">{{ selectedTireStats?.current_depth_mm != null ? `${formatNumber(selectedTireStats.current_depth_mm, 1)} mm` : '—' }}</div>
        </div>
        <div class="bg-slate-900/80 p-2 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.lifespanUsed') }}</div>
          <div class="font-bold text-slate-200">{{ formatPercent(Number(selectedTireStats?.life_progress_pct || 0)) }}</div>
        </div>
        <div class="bg-slate-900/80 p-2 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.actualCostKm', { unit: distanceUnit() }) }}</div>
          <div class="font-bold text-warning-400" :title="Number(selectedTire?.purchase_price) > 0 ? undefined : $t('tires.costPerKmUnset')">{{ formatCostPerDistance(selectedTireStats?.cost_per_km, vehicleStore.currency, 4) }}</div>
        </div>
      </div>
    </div>

    <!-- TeslaMate Driving Telemetry & Stress Analysis Card -->
    <div v-if="vehicleStore.capabilities.sync && selectedTireStats?.driving_stress_index > 0" class="bg-gradient-to-br from-slate-950 to-slate-900 border border-slate-800 p-4 rounded-2xl space-y-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Zap class="w-4 h-4 text-warning-400" />
          <h4 class="text-xs font-bold text-white">{{ $t('tires.tireHistoryModal.teslamateDrivingTelemetry') }}</h4>
        </div>
        <span
          class="px-2.5 py-0.5 rounded-full text-xs font-bold border"
          :class="
            selectedTireStats.driving_style === 'SPORT'
              ? 'bg-rose-500/15 text-rose-400 border-rose-500/30'
              : selectedTireStats.driving_style === 'ECO'
              ? 'bg-success-500/15 text-success-400 border-success-500/30'
              : 'bg-info-500/15 text-info-400 border-info-500/30'
          "
        >
          {{ selectedTireStats.driving_style === 'SPORT' ? $t('tires.tireHistoryModal.sport') : selectedTireStats.driving_style === 'ECO' ? $t('tires.tireHistoryModal.eco') : $t('tires.tireHistoryModal.balanced') }} {{ $t('tires.tireHistoryModal.index', { index: formatNumber(selectedTireStats.driving_stress_index, 2) }) }}
        </span>
      </div>

      <div class="grid grid-cols-2 sm:grid-cols-4 gap-2 text-center text-xs">
        <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.peakAcceleration') }}</div>
          <div class="font-bold text-rose-400 text-sm mt-0.5">+{{ formatNumber(selectedTireStats.avg_power_max_kw, 0) }} kW</div>
        </div>
        <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.peakRegeneration') }}</div>
          <div class="font-bold text-success-400 text-sm mt-0.5">{{ formatNumber(selectedTireStats.avg_power_min_kw, 0) }} kW</div>
        </div>
        <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.averageConsumption') }}</div>
          <div class="font-bold text-info-400 text-sm mt-0.5">{{ $t('tires.tireHistoryModal.kwh', { unit: distanceUnit(), avg_consumption_kwh_100km: formatPerDistanceValue(Number(selectedTireStats.avg_consumption_kwh_100km)) }) }}</div>
        </div>
        <div class="bg-slate-900 p-2.5 rounded-xl border border-slate-800">
          <div class="text-xs text-slate-400">{{ $t('tires.tireHistoryModal.adjustedLongevity') }}</div>
          <div class="font-bold text-sky-300 text-sm mt-0.5">~{{ formatDistance(selectedTireStats.dynamic_lifespan_km || selectedTire.estimated_lifespan_km) }}</div>
        </div>
      </div>

      <p v-if="selectedTireStats.wear_explanation" class="text-xs text-slate-300 bg-slate-900/60 p-2.5 rounded-xl border border-slate-800/80">
        {{ apiMessageText(selectedTireStats.wear_explanation) }}
      </p>
    </div>

    <!-- Timeline: Mount/Dismount Sessions -->
    <div class="space-y-3">
      <div class="flex items-center justify-between gap-3">
        <h4 class="flex min-w-0 items-center gap-1.5 text-xs font-bold text-white">
          <History class="h-4 w-4 shrink-0 text-slate-400" />
          {{ $t('tires.tireHistoryModal.historyOfFittingsRemovalsAnd') }}
        </h4>
        <div class="flex shrink-0 items-center gap-2">
          <button
            v-if="copiedSession"
            type="button"
            @click="emit('paste-session')"
            class="btn btn-secondary whitespace-nowrap border border-slate-700"
            :title="$t('tires.tireHistoryModal.pasteCopied', { date: copiedSession.mounted_date ? formatDate(copiedSession.mounted_date) : '' })" :aria-label="$t('tires.tireHistoryModal.pasteCopied', { date: copiedSession.mounted_date ? formatDate(copiedSession.mounted_date) : '' })"
          >
            <ClipboardPaste class="h-4 w-4 text-rose-400" aria-hidden="true" />
            <span>{{ $t('tires.tireHistoryModal.paste') }}</span>
          </button>
          <button type="button" @click="emit('add-session')" class="btn btn-secondary whitespace-nowrap border border-slate-700">
            <Plus class="h-4 w-4 text-rose-400" aria-hidden="true" />
            <span>{{ $t('tires.tireHistoryModal.addAPastSession') }}</span>
          </button>
        </div>
      </div>

      <div v-if="tireSessions.length === 0" class="p-6 text-center bg-slate-800 rounded-2xl text-xs text-slate-400">
        {{ $t('tires.tireHistoryModal.noSessionRecordedForThis') }}
      </div>

      <div v-else class="space-y-2">
        <button
          v-for="s in tireSessions"
          :key="s.id"
          type="button"
          class="flex w-full items-center gap-3 rounded-2xl bg-slate-800 p-3.5 text-left text-xs transition-colors hover:bg-slate-700"
          @click="detail = s"
        >
          <div class="min-w-0 flex-1">
            <div class="truncate text-sm font-bold text-white">{{ positionLabel(s.position) }}</div>
            <div class="mt-0.5 flex min-w-0 flex-wrap items-center gap-x-1.5 text-slate-400">
              <StatusLabel class="shrink-0" :tone="s.dismounted_date ? 'neutral' : 'info'">
                {{ s.dismounted_date ? $t('tires.tireHistoryModal.sessionOver') : $t('tires.tireHistoryModal.currentlyFitted') }}
              </StatusLabel>
              <span>{{ formatDate(s.mounted_date) }}</span>
            </div>
          </div>
          <span class="shrink-0 font-bold text-white">{{ $t('tires.tireHistoryModal.kmDriven', { unit: distanceUnit(), distance_km: formatDistanceValue(sessionKm(s)) }) }}</span>
          <ChevronRight class="h-4 w-4 shrink-0 text-slate-500" aria-hidden="true" />
        </button>
      </div>
    </div>

    <!-- Tread Logs (Mesures de gomme) -->
    <div class="space-y-2 pt-3 border-t border-slate-800">
      <div class="flex items-center justify-between gap-3">
        <h4 class="flex min-w-0 items-center gap-1.5 text-xs font-bold text-white">
          <Ruler class="h-4 w-4 shrink-0 text-slate-400" />
          {{ $t('tires.tireHistoryModal.treadDepthReadings', { length: tireLogs.length }) }}
        </h4>
        <button type="button" @click="emit('add-log')" class="btn btn-secondary shrink-0 whitespace-nowrap border border-slate-700">
          <Plus class="h-4 w-4 text-rose-400" aria-hidden="true" />
          <span>{{ $t('tires.tireHistoryModal.addAReading') }}</span>
        </button>
      </div>

      <div v-if="tireLogs.length > 0" class="grid grid-cols-2 sm:grid-cols-3 gap-2">
        <button
          v-for="l in tireLogs"
          :key="l.id"
          type="button"
          class="rounded-xl bg-slate-800 p-2.5 text-left text-xs transition-colors hover:bg-slate-700"
          :aria-label="$t('tires.tireHistoryModal.editTheReading')"
          @click="emit('edit-log', l)"
        >
          <div class="flex flex-wrap items-baseline justify-between gap-x-2">
            <span class="font-bold text-white">{{ l.depth_mm }} mm</span>
            <span class="text-slate-400">{{ formatDate(l.date) }}</span>
          </div>
          <div class="text-slate-400">{{ $t('common.atKm', { unit: distanceUnit(), km: formatDistanceValue(l.odometer) }) }}</div>
        </button>
      </div>
    </div>
    <template #footer>
      <DetailDeleteButton class="mr-auto" :aria-label="$t('tires.tireHistoryModal.deleteEntryError')" @delete="emit('delete-tire')" />
      <button
        type="button"
        @click="open = false"
        class="btn btn-lg btn-secondary"
      >
        {{ $t('common.close') }}
      </button>
    </template>
    <TireSessionDetailModal
      v-model:open="detailOpen"
      :session="detail"
      :km-driven="detail ? sessionKm(detail) : null"
      @edit="emit('edit-session', $event)"
      @copy="emit('copy-session', $event)"
      @duplicate="emit('duplicate-session', $event)"
      @delete="emit('delete-session', $event)"
    />
  </ModalShell>
</template>
