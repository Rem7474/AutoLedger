<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

// The end of a list that grows as it is scrolled: asks for the next page when it comes near the screen. The button does
// the same for a keyboard or a screen reader, and retries after an error.
const props = defineProps<{ loading: boolean; error?: string | null; label: string }>()
const emit = defineEmits<{ more: [] }>()

const el = ref<HTMLElement | null>(null)
let observer: IntersectionObserver | null = null

function observe() {
  if (!el.value || !observer) return
  // Observing again reports the position at once: a page that did not push the end off the screen asks for the next
  observer.disconnect()
  observer.observe(el.value)
}

onMounted(() => {
  if (typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver(
    (entries) => {
      if (entries.some((e) => e.isIntersecting) && !props.loading && !props.error) emit('more')
    },
    // Starts reading before the end shows, so the scroll does not stop at it
    { rootMargin: '600px 0px' },
  )
  observe()
})
onBeforeUnmount(() => observer?.disconnect())
watch(() => props.loading, (loading) => {
  if (!loading) observe()
})
</script>

<template>
  <div ref="el" class="flex flex-col items-center gap-2 py-4">
    <p v-if="error" class="text-xs text-rose-300">{{ error }}</p>
    <button v-if="!loading" type="button" class="btn btn-secondary" @click="emit('more')">{{ label }}</button>
    <p v-else class="text-xs text-slate-400" role="status">{{ $t('common.loading') }}</p>
  </div>
</template>
