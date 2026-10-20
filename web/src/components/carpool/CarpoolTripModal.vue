<script setup lang="ts">
import { t } from '@/i18n'
import { computed, ref, watch } from 'vue'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { Users, Lock, ChevronLeft, ChevronRight, Pencil } from 'lucide-vue-next'
import AppDatePicker from '@/components/AppDatePicker.vue'
import DrivePicker from '@/components/drives/DrivePicker.vue'
import { COST_FIELDS, allocate, cents, clampPassengerStops as clampStops, driveCostGapCents, earliestSelectedDriveDate, emptyLeg, estimateTitle, followDrives as followLegsToDrives, keepOverrides, legOverrides, legsFromEstimate, legsFromTrip, newPassenger, passengersFromTrip, remapPassengerStops, stopNames, toDateInputString, type LegForm, type PassengerForm } from '@/utils/carpool'
import ModalShell from '@/components/ModalShell.vue'
import { formatAmount } from '@/currency'
import { formatDistance } from '@/units'
import { useIsMobile } from '@/composables/useIsMobile'

import CarpoolLegs from '@/components/carpool/CarpoolLegs.vue'
import CarpoolPassengers from '@/components/carpool/CarpoolPassengers.vue'
import CarpoolRecap from '@/components/carpool/CarpoolRecap.vue'

// Creates a carpool trip, or edits `editing`. A new trip can start from drives or a trip group (createOptions).
// The editor goes by phases: the drives (or the typed legs), then the passengers, then the costs and the recap. A phone shows
// one phase at a time; a wide screen shows the drives first, then everything else side by side with the recap.
// openToken changes every time the page asks to open the modal, so the form is initialised again even if it is already open.
const props = defineProps<{
  vehicleId: string
  editing: any | null
  createOptions: { driveIds?: string[]; tripGroupId?: string }
  openToken: number
  recentNames?: string[]
}>()
const emit = defineEmits<{ saved: [] }>()
const open = defineModel<boolean>('open', { required: true })
const { showAlert, showConfirm } = useConfirm()

const editingTripId = computed<string | null>(() => props.editing?.id ?? null)
const modalSubmitting = ref(false)
const estimating = ref(false)
const vehicleStore = useVehicleStore()
// Carpool amounts are in the vehicle's own currency
const fmt = (v: number) => formatAmount(Number(v || 0), vehicleStore.currency)
// Trips are built from TeslaMate drives when the vehicle has them, from a typed distance otherwise
const defaultSourceMode = (): 'DRIVES' | 'MANUAL' => (vehicleStore.hasTeslaMate ? 'DRIVES' : 'MANUAL')
const sourceMode = ref<'DRIVES' | 'MANUAL'>(defaultSourceMode())
// Drives listed by the picker (it reports them: the earliest selected one dates a carpool without an estimate date)
const recentDrives = ref<any[]>([])
// Date the picker opens on: the carpool's own date, or none for the latest drives
const pickerAnchor = ref('')
const selectedDriveIds = ref<string[]>([])
const titleTouched = ref(false)
const currentRates = ref<any>(null)
const expandedPassengerIndex = ref<number | null>(null)
const selectedPassenger = ref(0)
const isMobile = useIsMobile()
type Phase = 'drives' | 'passengers' | 'recap'
const PHASES: Phase[] = ['drives', 'passengers', 'recap']
const phase = ref<Phase>('drives')
const main = computed(() => phase.value !== 'drives')
const showPassengers = computed(() => (isMobile.value ? phase.value === 'passengers' : main.value))
const showSummary = computed(() => (isMobile.value ? phase.value === 'recap' : main.value))
// Typed legs are edited where the drives would be picked on a phone, with everything else on a wide screen
const showLegs = computed(() => (sourceMode.value === 'MANUAL' ? (isMobile.value ? phase.value === 'drives' : true) : showSummary.value))
const showRecapColumn = computed(() => !isMobile.value && main.value)
const showRecapBar = computed(() => isMobile.value && main.value)

const isDateDisabled = computed(() => sourceMode.value === 'DRIVES')

const form = ref({
  title: '',
  date: toDateInputString(new Date()),
  trip_group_id: null as string | null,
  notes: '',
  legs: [] as LegForm[],
  passengers: [] as PassengerForm[],
  // The legs made from drives take the cost of their drive; when off, the amounts stay as entered
  followDrives: true,
})

// ---------- Live form computations ----------
const stops = computed(() => stopNames(form.value.legs))
const live = computed(() => allocate(form.value.legs, form.value.passengers))
const liveDistance = computed(() => form.value.legs.reduce((s, l) => s + (Number(l.distance_km) || 0), 0))
const liveRevenue = computed(() => form.value.passengers.reduce((s, p) => s + cents(p.amount_paid), 0))
const canContinue = computed(() => form.value.legs.length > 0 && !estimating.value)

// A carpool without legs (an estimate that failed, every drive unticked) goes back to the drives
watch([() => form.value.legs.length, estimating], ([count, busy]) => {
  if (count === 0 && !busy && phase.value !== 'drives' && sourceMode.value === 'DRIVES') phase.value = 'drives'
})

function goTo(next: Phase) {
  if (next !== 'drives' && !canContinue.value) return
  phase.value = next
}
function step(delta: number) {
  goTo(PHASES[Math.min(Math.max(PHASES.indexOf(phase.value) + delta, 0), PHASES.length - 1)])
}


// After the drives are chosen (from a trip group or the drives page), the picker moves to their date
function anchorPickerOnSelection() {
  if (selectedDriveIds.value.length && form.value.date) pickerAnchor.value = form.value.date
}

function clampPassengerStops(previousLegCount: number) {
  clampStops(form.value.passengers, form.value.legs.length, previousLegCount)
}

// ---------- Estimation ----------
function applyEstimate(est: any) {
  currentRates.value = est
  const previousLegs = form.value.legs
  const previousLegCount = previousLegs.length
  form.value.legs = legsFromEstimate(est)
  keepOverrides(previousLegs, form.value.legs)
  remapPassengerStops(previousLegs, form.value.legs, form.value.passengers)
  clampPassengerStops(previousLegCount === 0 ? 0 : -1)
  if (est.start_date) {
    form.value.date = toDateInputString(est.start_date)
  }
  if (!titleTouched.value && form.value.legs.length) {
    form.value.title = estimateTitle(stops.value, form.value.legs.length)
  }
}

async function estimateFromDrives() {
  if (!props.vehicleId) return
  if (!selectedDriveIds.value.length) {
    form.value.legs = []
    form.value.date = toDateInputString(new Date())
    return
  }
  estimating.value = true
  try {
    const est = await api.estimateCarpoolCosts(props.vehicleId, { drive_ids: selectedDriveIds.value })
    applyEstimate(est)
    if (est.start_date) {
      form.value.date = toDateInputString(est.start_date)
    } else {
      const first = earliestSelectedDriveDate(recentDrives.value, selectedDriveIds.value)
      if (first) form.value.date = first
    }
  } catch (err: any) {
    void showAlert(t('carpool.carpoolTripModal.estimateError', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    estimating.value = false
  }
}

async function toggleDrive(driveId: string) {
  const idx = selectedDriveIds.value.indexOf(driveId)
  if (idx > -1) selectedDriveIds.value.splice(idx, 1)
  else selectedDriveIds.value.push(driveId)
  await estimateFromDrives()
}

async function selectDrives(ids: string[]) {
  for (const id of ids) if (!selectedDriveIds.value.includes(id)) selectedDriveIds.value.push(id)
  await estimateFromDrives()
}

async function estimateManualLeg(index: number) {
  const leg = form.value.legs[index]
  if (!props.vehicleId || !(Number(leg.distance_km) > 0)) {
    void showAlert(t('carpool.carpoolTripModal.legDistanceFirst'), t('common.requiredField'), 'warning')
    return
  }
  estimating.value = true
  try {
    const est = await api.estimateCarpoolCosts(props.vehicleId, { distance_km: Number(leg.distance_km) })
    currentRates.value = est
    const estimated = est.legs?.[0] || est
    leg.electricity_cost = estimated.electricity_cost
    leg.tires_cost = estimated.tires_cost
    leg.maintenance_cost = estimated.maintenance_cost
    leg.insurance_cost = estimated.insurance_cost
  } catch (err: any) {
    void showAlert(t('carpool.carpoolTripModal.estimateError', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    estimating.value = false
  }
}

function addManualLeg() {
  const previous = form.value.legs.length
  const leg = emptyLeg()
  if (previous) leg.start_label = form.value.legs[previous - 1].end_label
  form.value.legs.push(leg)
  clampPassengerStops(previous)
}

function removeLeg(index: number) {
  if (form.value.legs.length <= 1) return
  const previous = form.value.legs.length
  form.value.legs.splice(index, 1)
  clampPassengerStops(previous)
}

function switchSource(mode: 'DRIVES' | 'MANUAL') {
  if (sourceMode.value === mode) return
  sourceMode.value = mode
  if (mode === 'MANUAL') {
    selectedDriveIds.value = []
    form.value.legs = form.value.legs.map((l) => ({ ...l, drive_id: null }))
    if (!form.value.legs.length) addManualLeg()
    if (!isMobile.value) phase.value = 'passengers'
  } else if (!isMobile.value) {
    phase.value = 'drives'
  }
}

// ---------- Initialisation ----------
function resetForm() {
  titleTouched.value = false
  currentRates.value = null
  expandedPassengerIndex.value = null
  selectedPassenger.value = 0
  selectedDriveIds.value = []
  pickerAnchor.value = ''
  sourceMode.value = defaultSourceMode()
  form.value = {
    title: '',
    date: toDateInputString(new Date()),
    trip_group_id: null,
    notes: '',
    legs: [],
    passengers: [],
    followDrives: true,
  }
  form.value.passengers.push(newPassenger(0, form.value.legs.length))
}

async function initCreate(options: { driveIds?: string[]; tripGroupId?: string }) {
  resetForm()
  const preselected = !!(options.tripGroupId || options.driveIds?.length)
  phase.value = preselected || (sourceMode.value === 'MANUAL' && !isMobile.value) ? 'passengers' : 'drives'
  if (sourceMode.value === 'MANUAL') {
    addManualLeg()
  }
  if (!props.vehicleId) return

  if (options.tripGroupId) {
    form.value.trip_group_id = options.tripGroupId
    estimating.value = true
    try {
      const est = await api.estimateCarpoolCosts(props.vehicleId, { trip_group_id: options.tripGroupId })
      selectedDriveIds.value = (est.legs || []).map((l: any) => l.drive_id).filter(Boolean)
      applyEstimate(est)
      const groups = await api.getTripGroups(props.vehicleId)
      const group = (groups || []).find((g: any) => g.id === options.tripGroupId)
      if (group?.name) {
        form.value.title = group.name
        titleTouched.value = true
      }
      if (est.start_date) {
        form.value.date = toDateInputString(est.start_date)
      } else if (group?.start_time) {
        form.value.date = toDateInputString(group.start_time)
      }
      anchorPickerOnSelection()
    } catch (err: any) {
      void showAlert(t('carpool.carpoolTripModal.estimateError', { message: err.message }), t('shell.confirm.error'), 'danger')
    } finally {
      estimating.value = false
    }
  } else if (options.driveIds?.length) {
    selectedDriveIds.value = [...options.driveIds]
    await estimateFromDrives()
    anchorPickerOnSelection()
  }
}

function initEdit(trip: any) {
  resetForm()
  titleTouched.value = true
  const legs = legsFromTrip(trip)
  sourceMode.value = legs.some((l) => l.drive_id) ? 'DRIVES' : 'MANUAL'
  selectedDriveIds.value = legs.map((l) => l.drive_id).filter((id): id is string => !!id)
  form.value = {
    title: trip.title,
    date: toDateInputString(trip.date),
    trip_group_id: trip.trip_group_id || null,
    notes: trip.notes || '',
    legs,
    passengers: passengersFromTrip(trip, legs.length),
    followDrives: !trip.costs_frozen_at,
  }
  if (!form.value.passengers.length) form.value.passengers.push(newPassenger(0, legs.length))
  pickerAnchor.value = form.value.date
  phase.value = 'passengers'
}

watch([open, () => props.openToken], ([isOpen]) => {
  if (!isOpen) return
  submitted.value = false
  if (props.editing) initEdit(props.editing)
  else initCreate(props.createOptions)
})

const submitted = ref(false)
const titleError = computed(() => (!form.value.title.trim() ? t('carpool.carpoolTripModal.titleRequired') : ''))
const legError = computed(() => (!form.value.legs.length ? t('carpool.carpoolTripModal.legRequired') : ''))

async function handleSave() {
  if (!props.vehicleId) return
  submitted.value = true
  if (titleError.value) {
    document.getElementById('carpool-title')?.focus()
    return
  }
  if (legError.value) return
  modalSubmitting.value = true
  try {
    const payload = {
      title: form.value.title,
      date: new Date(form.value.date).toISOString(),
      trip_group_id: form.value.trip_group_id,
      notes: form.value.notes || null,
      costs_frozen: !form.value.followDrives,
      legs: form.value.legs.map(({ drive_cost: _driveCost, ...l }) => ({
        ...l,
        distance_km: Number(l.distance_km) || 0,
        ...Object.fromEntries(COST_FIELDS.map((f) => [f.key, Number(l[f.key]) || 0])),
        // A frozen carpool keeps its amounts as they are: nothing is adjusted against the drives
        overrides: form.value.followDrives ? legOverrides(l as LegForm) : {},
      })),
      passengers: form.value.passengers.map((p) => ({
        ...p,
        seats: Number(p.seats) || 1,
        amount_paid: Number(p.amount_paid) || 0,
        notes: p.notes || null,
      })),
    }
    if (editingTripId.value) {
      await api.updateCarpool(props.vehicleId, editingTripId.value, payload)
    } else {
      await api.createCarpool(props.vehicleId, payload)
    }
    open.value = false
    emit('saved')
  } catch (err: any) {
    void showAlert(t('carpool.carpoolTripModal.saveError', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    modalSubmitting.value = false
  }
}

// Following the drives again replaces the amounts with what the drives cost; the difference is shown before it happens
async function setFollowDrives(follow: boolean) {
  if (follow === form.value.followDrives) return
  if (follow) {
    const gap = driveCostGapCents(form.value.legs)
    if (gap !== 0) {
      const ok = await showConfirm({
        title: t('carpool.linked.resyncTitle'),
        message: t('carpool.linked.resyncMessage', { amount: fmt(Math.abs(gap) / 100), direction: gap > 0 ? t('carpool.linked.more') : t('carpool.linked.less') }),
        confirmText: t('carpool.linked.resync'),
        type: 'warning',
      })
      if (!ok) return
    }
    followLegsToDrives(form.value.legs)
  }
  form.value.followDrives = follow
}

async function handleModalRecalculate() {
  if (!props.vehicleId) return
  if (sourceMode.value === 'DRIVES') {
    await estimateFromDrives()
  } else {
    for (let i = 0; i < form.value.legs.length; i++) {
      if (Number(form.value.legs[i].distance_km) > 0) {
        await estimateManualLeg(i)
      }
    }
  }
  void showAlert(t('carpool.carpoolTripModal.reestimated'), t('carpool.carpoolTripModal.reestimatedTitle'), 'success')
}
</script>

<template>
  <ModalShell
    v-model:open="open"
    size="lg"
    :class="main && !isMobile ? 'sm:max-w-5xl!' : ''"
    :icon="Users"
    icon-class="text-rose-400"
    :title="editingTripId ? $t('carpool.carpoolTripModal.edit') : $t('carpool.carpoolView.newCarpool')"
    body-class="space-y-5"
    footer-class="flex-col gap-2"
  >

      <!-- Phases (phone) -->
      <ol v-if="isMobile" class="flex items-center gap-2 text-xs font-semibold" :aria-label="$t('carpool.editor.steps')">
        <li v-for="(ph, i) in PHASES" :key="ph" class="flex-1">
          <button
            type="button"
            class="flex min-h-11 w-full items-center justify-center gap-1.5 rounded-xl border px-2"
            :class="phase === ph ? 'border-rose-500/60 bg-rose-500/10 text-rose-100' : 'border-slate-700 bg-slate-800 text-slate-300'"
            :aria-current="phase === ph ? 'step' : undefined"
            :disabled="ph !== 'drives' && !canContinue"
            @click="goTo(ph)"
          >
            <span class="tabular-nums">{{ i + 1 }}</span>
            <span class="truncate">{{ $t(`carpool.editor.step.${ph}`) }}</span>
          </button>
        </li>
      </ol>

      <!-- Source -->
      <div v-if="phase === 'drives' || (!isMobile && sourceMode === 'MANUAL' && vehicleStore.hasTeslaMate)" class="space-y-3">
        <div v-if="vehicleStore.hasTeslaMate || sourceMode === 'DRIVES'" class="flex items-center gap-1 bg-slate-950 border border-slate-800 p-1 rounded-xl w-fit text-xs font-semibold">
          <button
            @click="switchSource('DRIVES')"
            class="px-3 py-1.5 rounded-lg"
            :class="sourceMode === 'DRIVES' ? 'bg-rose-600 text-white' : 'text-slate-400 hover:text-white'"
          >
            {{ $t('carpool.carpoolTripModal.teslamateDrives') }}
          </button>
          <button
            @click="switchSource('MANUAL')"
            class="px-3 py-1.5 rounded-lg"
            :class="sourceMode === 'MANUAL' ? 'bg-rose-600 text-white' : 'text-slate-400 hover:text-white'"
          >
            {{ $t('carpool.carpoolTripModal.manualEntry') }}
          </button>
        </div>

        <div v-if="sourceMode === 'DRIVES' && phase === 'drives'" class="space-y-1.5">
          <div class="flex items-center justify-between text-xs">
            <span class="text-slate-400">{{ $t('carpool.carpoolTripModal.tickTheDrivesThatMake') }}</span>
            <span class="text-rose-400 font-semibold">{{ $t('carpool.carpoolTripModal.legsSelected', { count: selectedDriveIds.length }) }}{{ estimating ? $t('carpool.carpoolTripModal.estimating') : '' }}</span>
          </div>
          <DrivePicker
            :key="openToken"
            rich
            :vehicle-id="vehicleId"
            :selected-ids="selectedDriveIds"
            :anchor-date="pickerAnchor"
            @toggle="toggleDrive"
            @select="selectDrives"
            @loaded="recentDrives = $event"
          />
        </div>
      </div>

      <div :class="showRecapColumn ? 'grid grid-cols-[minmax(0,1fr)_20rem] items-start gap-6' : ''">
        <div class="min-w-0 space-y-5">
          <!-- The drives chosen, with the way back to the picker -->
          <div v-if="main && !isMobile && sourceMode === 'DRIVES'" class="flex items-center justify-between gap-3 rounded-xl border border-slate-800 bg-slate-950/50 px-3 py-2 text-xs">
            <span class="min-w-0 truncate text-slate-300">
              {{ $t('carpool.editor.drivesSummary', { count: selectedDriveIds.length, distance: formatDistance(liveDistance, 1) }) }}
              <span class="text-slate-400">· {{ stops[0] }} → {{ stops[stops.length - 1] }}</span>
            </span>
            <button type="button" class="tap-text flex shrink-0 items-center gap-1 font-semibold text-sky-400 hover:text-sky-300" @click="phase = 'drives'">
              <Pencil class="h-3.5 w-3.5" aria-hidden="true" /> {{ $t('carpool.editor.editDrives') }}
            </button>
          </div>

          <!-- On a phone the recap is the first thing of the last phase -->
          <CarpoolRecap
            v-if="isMobile && phase === 'recap'"
            :live="live"
            :revenue="liveRevenue"
            :currency="vehicleStore.currency"
          />

          <!-- Title & date -->
          <div v-if="showSummary" class="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <div class="sm:col-span-2">
              <label for="carpool-title" class="block text-xs font-semibold text-slate-400 mb-1">{{ $t('carpool.carpoolTripModal.title') }}</label>
              <input
                id="carpool-title"
                v-model="form.title"
                @input="titleTouched = true"
                :placeholder="$t('carpool.carpoolTripModal.eGAnnecyValence')"
                :aria-invalid="submitted && !!titleError"
                :aria-describedby="submitted && titleError ? 'carpool-title-error' : undefined"
                class="field"
              />
              <p v-if="submitted && titleError" id="carpool-title-error" class="text-xs text-danger-400 mt-1">{{ titleError }}</p>
            </div>
            <div>
              <div class="flex items-center justify-between mb-1">
                <label for="carpool-date" class="block text-xs font-semibold text-slate-400">{{ $t('common.date') }}</label>
                <span v-if="isDateDisabled" class="text-xs text-slate-400 flex items-center gap-1 font-normal" :title="$t('carpool.carpoolTripModal.theDateIsAutomaticallyLinked')">
                  <Lock class="w-3 h-3 text-slate-400" />
                  {{ $t('carpool.carpoolTripModal.tripDate') }}
                </span>
              </div>
              <AppDatePicker
                id="carpool-date"
                v-model="form.date"
                :disabled="isDateDisabled"
                required
              />
            </div>
          </div>

          <!-- Legs -->
          <CarpoolLegs
            v-if="showLegs"
            :legs="form.legs"
            :source-mode="sourceMode"
            :is-editing="!!editingTripId"
            :submitted="submitted"
            :estimating="estimating"
            :live="live"
            :live-distance="liveDistance"
            :currency="vehicleStore.currency"
            :follow-drives="form.followDrives"
            @update:follow-drives="setFollowDrives"
            @recalculate="handleModalRecalculate"
            @add-leg="addManualLeg"
            @estimate="estimateManualLeg"
            @remove="removeLeg"
          />

          <!-- Passengers -->
          <CarpoolPassengers
            v-if="showPassengers"
            v-model:passengers="form.passengers"
            v-model:expanded="expandedPassengerIndex"
            v-model:selected="selectedPassenger"
            :legs="form.legs"
            :stops="stops"
            :live="live"
            :currency="vehicleStore.currency"
            :recent-names="recentNames"
            @to-drives="phase = 'drives'"
          />

          <div v-if="showSummary">
            <label for="carpool-notes" class="block text-xs font-semibold text-slate-400 mb-1">{{ $t('carpool.carpoolTripModal.notesOptional') }}</label>
            <input id="carpool-notes" v-model="form.notes" class="field" />
          </div>
        </div>

        <div v-if="showRecapColumn" class="sticky top-0">
          <CarpoolRecap
            :live="live"
            :revenue="liveRevenue"
            :currency="vehicleStore.currency"
          />
        </div>
      </div>

    <template #footer>
      <CarpoolRecap
        v-if="showRecapBar && phase !== 'recap'"
        compact
        :live="live"
        :revenue="liveRevenue"
        :currency="vehicleStore.currency"
      />
      <div class="flex justify-end gap-2">
        <button
          v-if="isMobile && phase !== 'drives'"
          type="button"
          @click="step(-1)"
          class="btn btn-lg btn-secondary"
        >
          <ChevronLeft class="h-4 w-4" aria-hidden="true" /> {{ $t('carpool.editor.back') }}
        </button>
        <button
          v-else
          type="button"
          @click="open = false"
          class="btn btn-lg btn-secondary"
        >
          {{ $t('common.cancel') }}
        </button>
        <button
          v-if="phase === 'drives' || (isMobile && phase === 'passengers')"
          type="button"
          @click="step(1)"
          :disabled="!canContinue"
          class="btn btn-lg btn-primary"
        >
          {{ $t('carpool.editor.continue') }} <ChevronRight class="h-4 w-4" aria-hidden="true" />
        </button>
        <button
          v-else
          type="button"
          @click="handleSave"
          :disabled="modalSubmitting || estimating"
          class="btn btn-lg btn-primary"
        >
          {{ modalSubmitting ? $t('carpool.carpoolTripModal.saving') : $t('common.save') }}
        </button>
      </div>
    </template>
  </ModalShell>
</template>
