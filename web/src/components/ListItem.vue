<script setup lang="ts">
// The one row of a list: a card with four places, always in the same order. At the start the glyph of the row's theme
// (`lead`), then a title with its quiet line under it (`title`, `subtitle`), then the figure with its quiet line under
// it (`value`, `subvalue`). `meta` is the one optional mark between the two (a paperclip, a single chip, the two taps of
// a queue), never a menu. `footer` is a line inside the card under the row, for what does not fit beside the figure on a
// phone (the same two taps at full width). `wrapTitle` lets a long title run over two lines instead of being cut. What
// opens the row goes in `open` and stretches over the card, so the whole row is one target. `compact` is the same card
// one step smaller (padding, gap, type) for a row nested under another row (the legs of a trip).
defineProps<{ wrapTitle?: boolean; compact?: boolean }>()
</script>

<template>
  <div class="card relative px-3" :class="compact ? 'py-1.5' : 'py-2.5'">
    <slot name="open" />
    <div class="flex items-center" :class="compact ? 'gap-2.5' : 'gap-3'">
      <div v-if="$slots.lead" class="shrink-0"><slot name="lead" /></div>
      <div class="min-w-0 flex-1">
        <p class="font-semibold text-white" :class="[compact ? 'text-[13px] leading-snug' : 'text-sm', wrapTitle ? 'line-clamp-2' : 'truncate']"><slot name="title" /></p>
        <p v-if="$slots.subtitle" class="truncate text-xs text-slate-400"><slot name="subtitle" /></p>
      </div>
      <div v-if="$slots.meta" class="shrink-0"><slot name="meta" /></div>
      <div v-if="$slots.value" class="shrink-0 text-right">
        <p class="font-bold tabular-nums" :class="compact ? 'text-[13px]' : 'text-sm'"><slot name="value" /></p>
        <p v-if="$slots.subvalue" class="text-xs text-slate-400 tabular-nums"><slot name="subvalue" /></p>
      </div>
    </div>
    <div v-if="$slots.footer" class="relative z-10 mt-2"><slot name="footer" /></div>
  </div>
</template>
