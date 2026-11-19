<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import NumberInput from '@/components/NumberInput.vue'
import { Zap, ChevronDown } from '@/icons'
import { api } from '@/services/api'
import { t } from '@/i18n'
import { formatNumber } from '@/utils/numbers'
import { displayConsumptionToL100km, formatDistance, formatFuelConsumptionValue, fuelConsumptionUnit, l100kmToDisplayConsumption } from '@/units'
import { acceptBody, capacityBody, clearBody, percentIsValid, sharePercent, suggestionStatus, typedBody, type HybridShare } from '@/utils/hybridShare'

// The electric share of a plug-in hybrid: the part of the kilometres driven on the battery. The server suggests a value
// from the fill-ups and charges entered; nothing is applied until the owner accepts it or types another one, and a
// value only applies to trips saved afterwards.
const props = defineProps<{ vehicleId: string; canEdit: boolean; bare?: boolean }>()
const emit = defineEmits<{ saved: [HybridShare['accepted']] }>()

const res = ref<HybridShare | null>(null)
const error = ref('')
const busy = ref(false)
const editing = ref(false)
const percent = ref('')
const fuel = ref('')
const capacity = ref('')

const status = computed(() => (res.value ? suggestionStatus(res.value) : 'insufficient'))
const accepted = computed(() => res.value?.accepted ?? null)
const acceptedPercent = computed(() => (accepted.value?.electric_share == null ? null : sharePercent(accepted.value.electric_share)))
const suggestedPercent = computed(() => (res.value?.suggested_electric_share === undefined ? null : sharePercent(res.value.suggested_electric_share)))
const quality = computed(() => res.value?.quality)
const fuelPart = computed(() => (accepted.value?.fuel_l_100km ? `${formatFuelConsumptionValue(accepted.value.fuel_l_100km, 1)} ${fuelConsumptionUnit()}` : null))
const capacityKwh = computed(() => accepted.value?.battery_usable_kwh ?? res.value?.battery_usable_kwh ?? null)

function fillForm() {
  const a = res.value?.accepted
  percent.value = a?.electric_share == null ? '' : String(sharePercent(a.electric_share))
  const l100 = a?.fuel_l_100km ?? res.value?.fuel_l_100km_fuel_part
  fuel.value = l100 ? String(Math.round(l100kmToDisplayConsumption(l100) * 10) / 10) : ''
  capacity.value = a?.battery_usable_kwh == null ? '' : String(a.battery_usable_kwh)
}

async function load() {
  error.value = ''
  try {
    res.value = await api.getHybridShare(props.vehicleId)
    fillForm()
  } catch (e) {
    error.value = e instanceof Error ? e.message : ''
  }
}

async function send(data: Parameters<typeof api.saveHybridShare>[1]) {
  busy.value = true
  error.value = ''
  try {
    const saved = await api.saveHybridShare(props.vehicleId, data)
    emit('saved', saved)
    await load()
  } catch (e) {
    error.value = e instanceof Error ? e.message : ''
  } finally {
    busy.value = false
  }
}

const accept = () => send(acceptBody(accepted.value))

function saveTyped() {
  const body = typedBody({ percent: percent.value, fuel: fuel.value, capacity: capacity.value }, displayConsumptionToL100km)
  if (body) void send(body)
}

const clear = () => send(clearBody(capacity.value))
const saveCapacity = () => send(capacityBody(accepted.value, capacity.value))

watch(() => props.vehicleId, () => {
  res.value = null
  void load()
}, { immediate: true })

const levelLabel = computed(() => (quality.value?.level === 'good' ? t('vehicles.hybridShare.levelGood') : t('vehicles.hybridShare.levelLow')))
const levelClass = computed(() => (quality.value?.level === 'good' ? 'bg-success-500/15 text-success-400' : 'bg-warning-500/15 text-warning-400'))
const typedValid = computed(() => percentIsValid(percent.value))
</script>

<template>
  <section class="space-y-4 text-xs" :class="{ 'card p-5 shadow-sm': !bare }" aria-labelledby="hybrid-share-title">
    <h3 id="hybrid-share-title" class="flex items-center gap-2 text-sm font-bold text-white" :class="{ 'sr-only': bare }">
      <Zap class="h-4 w-4 text-success-400" aria-hidden="true" />
      {{ $t('vehicles.hybridShare.title') }}
    </h3>
    <p class="text-slate-400">{{ $t('vehicles.hybridShare.intro') }}</p>

    <!-- What the owner accepted: the value the trips use -->
    <div v-if="acceptedPercent !== null" data-testid="hybrid-accepted">
      <div class="flex flex-wrap items-baseline gap-x-2">
        <span class="text-3xl font-extrabold text-success-400">{{ acceptedPercent }} %</span>
        <span class="text-slate-400">{{ $t('vehicles.hybridShare.electric') }}</span>
      </div>
      <div class="mt-2 flex h-2 overflow-hidden rounded-full bg-warning-500/60" role="meter" aria-valuemin="0" aria-valuemax="100" :aria-valuenow="acceptedPercent" :aria-label="$t('vehicles.hybridShare.title')">
        <div class="h-full bg-success-500" :style="{ width: `${acceptedPercent}%` }"></div>
      </div>
      <p class="mt-1 text-slate-400">
        {{ $t('vehicles.hybridShare.fuelPart', { percent: 100 - acceptedPercent }) }}<template v-if="fuelPart"> · {{ fuelPart }}</template>
      </p>
    </div>
    <p v-else class="text-slate-300" data-testid="hybrid-none">{{ $t('vehicles.hybridShare.noneAccepted') }}</p>

    <!-- What the history suggests -->
    <div class="space-y-2 rounded-xl bg-slate-800 p-3" data-testid="hybrid-suggestion">
      <template v-if="status === 'insufficient'">
        <p class="font-semibold text-slate-200">{{ $t('vehicles.hybridShare.nothingYet') }}</p>
        <p class="text-slate-400">{{ $t(`vehicles.hybridShare.because.${quality?.insufficient_because ?? 'not_enough_data'}`) }}</p>
      </template>
      <template v-else>
        <div class="flex flex-wrap items-center gap-2">
          <span class="font-semibold text-slate-200">{{ $t('vehicles.hybridShare.suggested', { percent: suggestedPercent }) }}</span>
          <span class="rounded-full px-2 py-0.5 font-semibold" :class="levelClass">{{ levelLabel }}</span>
        </div>
        <p class="text-slate-400">
          {{ $t('vehicles.hybridShare.basis', { periods: quality?.segments_used ?? 0, km: formatDistance(quality?.km_covered ?? 0) }) }}
          <template v-if="quality?.window_months">{{ $t('vehicles.hybridShare.window', { months: quality.window_months }) }}</template>
        </p>
        <p v-if="quality?.level === 'low'" class="text-slate-400">{{ $t('vehicles.hybridShare.lowHint') }}</p>
        <p v-if="quality?.less_precise_energy_charges" class="text-slate-400">{{ $t('vehicles.hybridShare.lessPrecise', { count: quality.less_precise_energy_charges }) }}</p>
        <button v-if="status === 'available' && canEdit" type="button" class="btn btn-primary" :disabled="busy" @click="accept">{{ $t('vehicles.hybridShare.use') }}</button>
        <p v-else-if="status === 'accepted'" class="text-success-400">{{ $t('vehicles.hybridShare.alreadyUsed') }}</p>
      </template>
    </div>

    <p class="text-slate-400">{{ $t('vehicles.hybridShare.appliesLater') }}</p>

    <!-- Battery capacity, deduced from the charges that carry percentages -->
    <p class="text-slate-300" data-testid="hybrid-capacity">
      <template v-if="capacityKwh !== null">{{ $t(accepted?.battery_usable_kwh ? 'vehicles.hybridShare.capacityTyped' : 'vehicles.hybridShare.capacityDeduced', { kwh: formatNumber(capacityKwh, 1) }) }}</template>
      <template v-else>{{ $t('vehicles.hybridShare.capacityUnknown') }}</template>
    </p>

    <div v-if="canEdit" class="rounded-xl bg-slate-800 px-3 py-2">
      <button type="button" class="flex w-full items-center justify-between gap-2 font-semibold text-slate-300" :aria-expanded="editing" aria-controls="hybrid-share-form" @click="editing = !editing">
        {{ $t('vehicles.hybridShare.edit') }}
        <ChevronDown class="h-4 w-4 transition-transform" :class="{ 'rotate-180': editing }" aria-hidden="true" />
      </button>
      <div v-show="editing" id="hybrid-share-form">
      <form class="mt-2 grid gap-2 sm:grid-cols-2" @submit.prevent="saveTyped">
        <label class="space-y-1">
          <span class="block text-slate-400">{{ $t('vehicles.hybridShare.percentLabel') }}</span>
          <NumberInput id="hybrid-share-percent" text v-model="percent" min="0" max="100" class="w-full field" />
        </label>
        <label class="space-y-1">
          <span class="block text-slate-400">{{ $t('vehicles.hybridShare.fuelLabel', { unit: fuelConsumptionUnit() }) }}</span>
          <NumberInput id="hybrid-share-fuel" text v-model="fuel" min="0" class="w-full field" />
        </label>
        <div class="flex flex-wrap gap-2 sm:col-span-2">
          <button type="submit" class="btn btn-secondary" :disabled="busy || !typedValid">{{ $t('vehicles.hybridShare.save') }}</button>
          <button v-if="acceptedPercent !== null" type="button" class="btn btn-secondary" :disabled="busy" @click="clear">{{ $t('vehicles.hybridShare.clear') }}</button>
        </div>
      </form>
      <form class="mt-3 flex flex-wrap items-end gap-2" @submit.prevent="saveCapacity">
        <label class="space-y-1">
          <span class="block text-slate-400">{{ $t('vehicles.hybridShare.capacityLabel') }}</span>
          <NumberInput id="hybrid-share-capacity" text v-model="capacity" min="0" class="w-36 field" />
        </label>
        <button type="submit" class="btn btn-secondary" :disabled="busy">{{ $t('vehicles.hybridShare.saveCapacity') }}</button>
      </form>
      <p class="mt-2 text-slate-400">{{ $t('vehicles.hybridShare.capacityHint') }}</p>
      </div>
    </div>
    <p v-if="error" class="text-danger-400" role="alert">{{ error }}</p>
  </section>
</template>
