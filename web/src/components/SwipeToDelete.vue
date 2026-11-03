<script setup lang="ts">
import { computed, ref } from 'vue'
import { Trash2 } from 'lucide-vue-next'
import { useIsMobile } from '@/composables/useIsMobile'
import { createSwipeGesture } from '@/utils/swipeRow'

// A list row on a phone that deletes when dragged to the left. The delete itself is the page's (`delete`), which
// takes the row off the list and offers to undo; the actions of the row stay reachable without the gesture.
const props = defineProps<{ disabled?: boolean }>()
const emit = defineEmits<{ delete: [] }>()

const isMobile = useIsMobile()
const active = computed(() => isMobile.value && !props.disabled)
const gesture = createSwipeGesture()
const offset = ref(0)
const armed = ref(false)
const dragging = ref(false)
let swiped = false
// The edges of the screen belong to the system's back gesture
const EDGE_PX = 24

function onPointerdown(e: PointerEvent) {
  swiped = false
  if (!active.value || e.pointerType !== 'touch') return
  if (e.clientX < EDGE_PX || e.clientX > window.innerWidth - EDGE_PX) return
  gesture.start(e.clientX, e.clientY)
}
function onPointermove(e: PointerEvent) {
  const move = gesture.move(e.clientX, e.clientY)
  if (!move) return
  if (!dragging.value) {
    dragging.value = true
    ;(e.currentTarget as HTMLElement).setPointerCapture?.(e.pointerId)
  }
  offset.value = move.offset
  armed.value = move.armed
}
function finish(cancelled: boolean) {
  swiped = gesture.dragging
  const result = gesture.end(cancelled)
  dragging.value = false
  offset.value = 0
  armed.value = false
  if (result === 'delete') emit('delete')
}
// What follows a swipe is not a tap on the row
function onClickCapture(e: Event) {
  if (!swiped) return
  swiped = false
  e.stopPropagation()
  e.preventDefault()
}
</script>

<template>
  <div
    class="relative overflow-hidden rounded-2xl"
    :class="{ 'touch-pan-y': active }"
    @pointerdown="onPointerdown"
    @pointermove="onPointermove"
    @pointerup="finish(false)"
    @pointercancel="finish(true)"
    @click.capture="onClickCapture"
  >
    <div v-if="offset < 0" class="absolute inset-0 flex items-center justify-end bg-danger-600 pr-6 text-white" aria-hidden="true">
      <Trash2 class="h-5 w-5 transition-transform" :class="armed ? 'scale-125' : 'scale-100'" />
    </div>
    <div :class="dragging ? '' : 'transition-transform duration-150 ease-out'" :style="offset ? { transform: `translateX(${offset}px)` } : undefined">
      <slot />
    </div>
  </div>
</template>
