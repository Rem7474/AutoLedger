<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { formatDistance } from '@/units'
import { ChevronLeft, ChevronRight, Link2, Check } from 'lucide-vue-next'
import AppDatePicker from '@/components/AppDatePicker.vue'
import { api } from '@/services/api'
import { DRIVE_WINDOW_DAYS, driveWindow, formatDriveTime, pickerDrives, shiftDay, toDateInputString } from '@/utils/carpool'
import { drivesChain, findDriveChain, formatDriveClock, formatDriveDay, groupDrivesByDay, splitShortDrives } from '@/utils/carpoolEditor'

// The drives to tick to build a carpool or a trip. It lists the drives around `anchorDate` (the date of the record
// being edited), or the latest ones without a date, with a date field and arrows to move the window. The selected
// drives are always listed, even outside the window. The parent owns the selection: it gets `toggle`. `rich` lists the
// drives by day, suggests the drives that chain into one journey (`select` asks for them all at once), points out the
// drives that follow one another, hides the very short ones and totals the selection.
const props = defineProps<{ vehicleId: string; selectedIds: string[]; anchorDate: string; rich?: boolean }>()
const emit = defineEmits<{ toggle: [driveId: string]; select: [driveIds: string[]]; loaded: [drives: any[]] }>()

const pickerDate = ref('')
const windowDrives = ref<any[]>([])
const drives = ref<any[]>([])
const known = new Map<string, any>()
// Drives kept in the list once selected, until the window moves, so unticking one does not make it vanish
const sticky = new Set<string>()

async function fetchMissingSelected() {
  const missing = props.selectedIds.filter((id) => !known.has(id))
  await Promise.all(
    missing.map(async (id) => {
      const one = await api.getDrives(props.vehicleId, { driveId: id, limit: 1 })
      if (one.drives?.[0]) known.set(id, one.drives[0])
    }),
  )
}

function merge() {
  for (const id of props.selectedIds) sticky.add(id)
  drives.value = pickerDrives(windowDrives.value, known, [...sticky])
  emit('loaded', drives.value)
}

async function load() {
  if (!props.vehicleId) return
  try {
    const params = pickerDate.value ? { ...driveWindow(pickerDate.value), limit: 200 } : { limit: 200 }
    const res = await api.getDrives(props.vehicleId, params)
    windowDrives.value = res.drives || []
    for (const d of windowDrives.value) known.set(d.id, d)
    sticky.clear()
    await fetchMissingSelected()
    merge()
  } catch (err) {
    console.error('Failed to load the drives', err)
  }
}

function shiftWindow(days: number) {
  pickerDate.value = shiftDay(pickerDate.value || toDateInputString(new Date()), days)
  load()
}

watch(
  () => props.anchorDate,
  (date) => {
    pickerDate.value = date
    load()
  },
  { immediate: true },
)

// The selection also changes from outside (an estimate, a trip group): its drives are listed too
watch(
  () => props.selectedIds.join(','),
  async () => {
    try {
      await fetchMissingSelected()
      merge()
    } catch (err) {
      console.error('Failed to load the selected drives', err)
    }
  },
)

const showShort = ref(false)
const split = computed(() => splitShortDrives(drives.value, props.selectedIds))
const listed = computed(() => (showShort.value ? drives.value : split.value.shown))
const groups = computed(() => groupDrivesByDay(listed.value))
const chain = computed(() => findDriveChain(split.value.shown))
const chainApplied = computed(() => chain.value.length > 0 && chain.value.every((d) => props.selectedIds.includes(d.id)))
const selectedDrives = computed(() => drives.value.filter((d) => props.selectedIds.includes(d.id)))
const selectedKm = computed(() => selectedDrives.value.reduce((sum, d) => sum + (Number(d.distance_km) || 0), 0))
const place = (address?: string) => (address || '').split(',')[0]
</script>

<template>
  <div class="space-y-1.5">
    <!-- Which drives are listed: the latest ones, or those around a date -->
    <div class="flex items-center gap-1.5 text-xs text-slate-400">
      <button type="button" @click="shiftWindow(-2 * DRIVE_WINDOW_DAYS)" class="tap p-1 rounded-lg hover:bg-slate-800 hover:text-white" :title="$t('drives.drivePicker.earlierDrives')" :aria-label="$t('drives.drivePicker.earlierDrives')">
        <ChevronLeft class="w-4 h-4" />
      </button>
      <div class="w-44">
        <AppDatePicker
          id="drive-picker-window"
          v-model="pickerDate"
          size="xs"
          :clearable="true"
          :placeholder="$t('drives.drivePicker.latestDrives')"
          @change="load"
        />
      </div>
      <button type="button" @click="shiftWindow(2 * DRIVE_WINDOW_DAYS)" class="tap p-1 rounded-lg hover:bg-slate-800 hover:text-white" :title="$t('drives.drivePicker.laterDrives')" :aria-label="$t('drives.drivePicker.laterDrives')">
        <ChevronRight class="w-4 h-4" />
      </button>
      <span v-if="pickerDate" class="text-xs text-slate-400">{{ $t('drives.drivePicker.aroundTheDate', { days: DRIVE_WINDOW_DAYS }) }}</span>
    </div>
    <p v-if="!drives.length" class="text-xs text-slate-400">{{ $t('drives.drivePicker.noDriveInThisPeriod') }}</p>
    <template v-if="rich">
      <div v-if="chain.length" class="flex items-center justify-between gap-3 rounded-xl border border-slate-700 bg-slate-800/60 p-3 text-xs">
        <span class="flex min-w-0 items-start gap-2 text-slate-300">
          <Link2 class="mt-0.5 h-4 w-4 shrink-0 text-sky-400" aria-hidden="true" />
          <span>
            <strong class="block text-slate-100">{{ $t('drives.drivePicker.chainDetected') }}</strong>
            {{ $t('drives.drivePicker.chainHint', { count: chain.length, from: place(chain[0].start_address), to: place(chain[chain.length - 1].end_address) }) }}
          </span>
        </span>
        <span v-if="chainApplied" class="flex shrink-0 items-center gap-1 rounded-full border border-slate-600 px-2.5 py-1 font-semibold text-slate-200">
          <Check class="h-3.5 w-3.5" aria-hidden="true" /> {{ $t('drives.drivePicker.applied') }}
        </span>
        <button v-else type="button" class="btn btn-secondary shrink-0" @click="emit('select', chain.map((d) => d.id))">{{ $t('drives.drivePicker.useChain') }}</button>
      </div>
      <div class="max-h-[46dvh] space-y-3 overflow-y-auto pr-1">
        <section v-for="g in groups" :key="g.day" :aria-label="formatDriveDay(g.day)">
          <h5 class="mb-1 text-xs font-semibold capitalize text-slate-400">{{ formatDriveDay(g.day) }}</h5>
          <div>
            <template v-for="(d, k) in g.drives" :key="d.id">
              <div v-if="k > 0 && drivesChain(g.drives[k - 1], d)" class="ml-5 flex items-center gap-1.5 border-l border-dashed border-slate-600 py-0.5 pl-3 text-xs text-slate-400">
                <Link2 class="h-3 w-3" aria-hidden="true" /> {{ $t('drives.drivePicker.chained') }}
              </div>
              <label
                class="flex min-h-11 w-full cursor-pointer items-center justify-between gap-3 rounded-lg border p-2 text-left text-xs transition-colors"
                :class="selectedIds.includes(d.id) ? 'bg-rose-500/10 border-rose-500/40 text-rose-100' : 'bg-slate-800/60 border-slate-700 text-slate-300 hover:bg-slate-800'"
              >
                <span class="flex min-w-0 items-center gap-2">
                  <input type="checkbox" class="select-box" :checked="selectedIds.includes(d.id)" @change="emit('toggle', d.id)" />
                  <span class="shrink-0 tabular-nums text-slate-400">{{ formatDriveClock(d.start_time) }}</span>
                  <span class="truncate">{{ (d.start_address || $t('drives.driveCostModal.start')).split(',')[0] }} → {{ (d.end_address || $t('drives.driveCostModal.end')).split(',')[0] }}</span>
                </span>
                <span class="flex shrink-0 items-center gap-2">
                  <span v-if="d.carpools?.length" class="rounded-full border border-sky-500/30 bg-sky-500/15 px-1.5 py-0.5 text-[0.65rem] font-semibold text-sky-300" :title="d.carpools.map((c: any) => c.title).join(', ')">{{ $t('drives.drivePicker.alreadyCarpooledIn', { title: d.carpools[0].title }) }}</span>
                  <span class="font-mono text-xs text-slate-400">{{ formatDistance(Number(d.distance_km)) }}</span>
                </span>
              </label>
            </template>
          </div>
        </section>
      </div>
      <button v-if="split.hidden.length" type="button" class="tap-text text-xs text-sky-400 hover:text-sky-300" @click="showShort = !showShort">
        {{ showShort ? $t('drives.drivePicker.hideShort') : $t('drives.drivePicker.showShort', { count: split.hidden.length }) }}
      </button>
      <p v-if="selectedDrives.length" class="flex items-center justify-between border-t border-slate-800 pt-2 text-xs text-slate-300">
        <span>{{ $t('drives.drivePicker.selectedCount', { count: selectedDrives.length }) }}</span>
        <strong class="tabular-nums text-white">{{ formatDistance(selectedKm, 1) }}</strong>
      </p>
    </template>
    <div v-else class="max-h-44 overflow-y-auto space-y-1 pr-1">
      <label
        v-for="d in drives"
        :key="d.id"
        class="w-full flex items-center justify-between gap-3 p-2 rounded-lg text-xs border text-left cursor-pointer transition-colors"
        :class="selectedIds.includes(d.id) ? 'bg-rose-500/10 border-rose-500/40 text-rose-100' : 'bg-slate-800/60 border-slate-700 text-slate-300 hover:bg-slate-800'"
      >
        <span class="flex items-center gap-2 truncate">
          <input type="checkbox" class="select-box" :checked="selectedIds.includes(d.id)" @change="emit('toggle', d.id)" />
          <span class="truncate">{{ formatDriveTime(d.start_time) }}{{ $t('drives.tripGroupsPanel.dateSeparator') }}{{ (d.start_address || $t('drives.driveCostModal.start')).split(',')[0] }} → {{ (d.end_address || $t('drives.driveCostModal.end')).split(',')[0] }}</span>
        </span>
        <span class="flex items-center gap-2 shrink-0">
          <span v-if="d.carpools?.length" class="px-1.5 py-0.5 rounded-full text-[0.65rem] font-semibold bg-sky-500/15 text-sky-300 border border-sky-500/30">{{ $t('drives.drivePicker.alreadyCarpooled') }}</span>
          <span class="font-mono text-xs text-slate-400">{{ formatDistance(Number(d.distance_km)) }}</span>
        </span>
      </label>
    </div>
  </div>
</template>
