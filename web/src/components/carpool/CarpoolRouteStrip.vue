<script setup lang="ts">
import { computed, ref } from 'vue'
import { stopFromRatio, tapStop, withAlight, withBoard, type StopRange } from '@/utils/carpoolEditor'

// The route of a carpool as a strip of stops: the part a passenger rides is highlighted, and its two ends are handles to
// drag (or move with the arrow keys). A tap on a stop moves the nearer end there. `readonly` only shows the ride.
const props = defineProps<{ stops: string[]; board: number; alight: number; readonly?: boolean }>()
const emit = defineEmits<{ change: [range: StopRange] }>()

const track = ref<HTMLElement | null>(null)
const legCount = computed(() => Math.max(props.stops.length - 1, 1))
// Stops sit at the centre of equal columns, so each name can be centred under its stop
const left = (i: number) => `${((i + 0.5) / props.stops.length) * 100}%`
const range = computed<StopRange>(() => ({ board: props.board, alight: props.alight }))

function stopAt(clientX: number): number {
  const rect = track.value?.getBoundingClientRect()
  if (!rect || rect.width === 0) return 0
  const column = ((clientX - rect.left) / rect.width) * props.stops.length - 0.5
  return stopFromRatio(column / legCount.value, legCount.value)
}

function drag(which: 'board' | 'alight', event: PointerEvent) {
  const el = event.currentTarget as HTMLElement
  el.setPointerCapture(event.pointerId)
  const move = (e: PointerEvent) => {
    const stop = stopAt(e.clientX)
    emit('change', which === 'board' ? withBoard(range.value, stop) : withAlight(range.value, stop, legCount.value))
  }
  const end = () => {
    el.removeEventListener('pointermove', move)
    el.removeEventListener('pointerup', end)
    el.removeEventListener('pointercancel', end)
  }
  el.addEventListener('pointermove', move)
  el.addEventListener('pointerup', end)
  el.addEventListener('pointercancel', end)
}

function key(which: 'board' | 'alight', event: KeyboardEvent) {
  const current = which === 'board' ? props.board : props.alight
  const target =
    event.key === 'ArrowLeft' || event.key === 'ArrowDown' ? current - 1
    : event.key === 'ArrowRight' || event.key === 'ArrowUp' ? current + 1
    : event.key === 'Home' ? 0
    : event.key === 'End' ? legCount.value
    : null
  if (target === null) return
  event.preventDefault()
  emit('change', which === 'board' ? withBoard(range.value, target) : withAlight(range.value, target, legCount.value))
}
</script>

<template>
  <div>
    <div ref="track" class="relative touch-none select-none" :class="readonly ? 'h-6' : 'h-11'">
      <div class="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-slate-700" :style="{ left: left(0), width: `${((stops.length - 1) / stops.length) * 100}%` }"></div>
      <div
        class="absolute top-1/2 h-1.5 -translate-y-1/2 rounded-full bg-rose-500"
        :style="{ left: left(board), width: `${((alight - board) / stops.length) * 100}%` }"
      ></div>
      <span
        v-for="(name, i) in (readonly ? stops : [])"
        :key="`dot-${i}`"
        class="absolute top-1/2 h-3 w-3 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 bg-slate-900"
        :class="i === board || i === alight ? 'border-rose-500 bg-rose-500' : i > board && i < alight ? 'border-rose-500' : 'border-slate-500'"
        :style="{ left: left(i) }"
      ></span>
      <button
        v-for="(name, i) in (readonly ? [] : stops)"
        :key="i"
        type="button"
        class="absolute top-1/2 flex h-11 w-11 -translate-x-1/2 -translate-y-1/2 items-center justify-center"
        :style="{ left: left(i) }"
        :aria-label="name"
        @click="emit('change', tapStop(range, i, legCount))"
      >
        <span class="h-3 w-3 rounded-full border-2" :class="i >= board && i <= alight ? 'border-rose-500 bg-slate-900' : 'border-slate-500 bg-slate-900'"></span>
      </button>
      <div
        v-for="which in (readonly ? [] : (['board', 'alight'] as const))"
        :key="which"
        role="slider"
        tabindex="0"
        class="absolute top-1/2 z-10 flex h-11 w-11 -translate-x-1/2 -translate-y-1/2 cursor-grab items-center justify-center rounded-full outline-none active:cursor-grabbing focus-visible:ring-2 focus-visible:ring-rose-400"
        :style="{ left: left(which === 'board' ? board : alight) }"
        :aria-label="which === 'board' ? $t('carpool.editor.boardingStop') : $t('carpool.editor.alightingStop')"
        aria-orientation="horizontal"
        :aria-valuemin="which === 'board' ? 0 : board + 1"
        :aria-valuemax="which === 'board' ? alight - 1 : legCount"
        :aria-valuenow="which === 'board' ? board : alight"
        :aria-valuetext="stops[which === 'board' ? board : alight]"
        @pointerdown.prevent="drag(which, $event)"
        @keydown="key(which, $event)"
      >
        <span class="h-5 w-5 rounded-full border-2 border-slate-900 bg-rose-500 shadow"></span>
      </div>
    </div>
    <div class="flex text-xs text-slate-400" aria-hidden="true">
      <span
        v-for="(name, i) in stops"
        :key="i"
        class="min-w-0 flex-1 truncate px-0.5 text-center"
        :class="i >= board && i <= alight ? 'font-semibold text-slate-200' : ''"
        :title="name"
      >{{ name }}</span>
    </div>
  </div>
</template>
