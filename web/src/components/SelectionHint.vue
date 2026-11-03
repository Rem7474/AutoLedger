<script setup lang="ts">
import { Lightbulb } from 'lucide-vue-next'
import { useIsMobile } from '@/composables/useIsMobile'
import { useSelectionHint } from '@/composables/useSelectionHint'

// One line telling a phone user that a long press selects a row (the row actions live in the contextual bar that
// opens then). It shows until it is dismissed or a selection has been made once, and never on a desktop.
const props = defineProps<{ visible?: boolean }>()
const isMobile = useIsMobile()
const { seen, dismiss } = useSelectionHint()
defineExpose({ dismiss })
</script>

<template>
  <div
    v-if="isMobile && !seen && props.visible !== false"
    class="flex items-center gap-2.5 rounded-xl border border-info-500/30 bg-info-500/10 px-3 py-2 text-xs font-medium text-info-300"
  >
    <Lightbulb class="h-4 w-4 shrink-0" aria-hidden="true" />
    <span class="flex-1">{{ $t('shell.bulkSelectionBar.hint') }}</span>
    <button type="button" class="tap -my-1 -mr-1 rounded-lg px-2 font-semibold text-info-200 hover:bg-info-500/15" @click="dismiss">
      {{ $t('shell.bulkSelectionBar.hintDismiss') }}
    </button>
  </div>
</template>
