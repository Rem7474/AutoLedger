<script setup lang="ts">
// The one row of a list: a card with four places, always in the same order. At the start the glyph of the row's theme
// (`lead`), then a title with its quiet line under it (`title`, `subtitle`), then the figure with its quiet line under
// it (`value`, `subvalue`). `meta` is the one optional mark between the two (a paperclip, a single chip), never a row of
// buttons. What opens the row goes in `open` and stretches over the card, so the whole row is one target. The quiet line
// is cut at the end of one line; `wrapSubtitle` lets it run over two lines when everything on it matters.
defineProps<{ wrapSubtitle?: boolean }>()
</script>

<template>
  <div class="card relative flex items-center gap-3 px-3 py-2.5">
    <slot name="open" />
    <div v-if="$slots.lead" class="shrink-0"><slot name="lead" /></div>
    <div class="min-w-0 flex-1">
      <p class="truncate text-sm font-semibold text-white"><slot name="title" /></p>
      <p v-if="$slots.subtitle" class="text-xs text-slate-400" :class="wrapSubtitle ? 'line-clamp-2' : 'truncate'"><slot name="subtitle" /></p>
    </div>
    <div v-if="$slots.meta" class="shrink-0"><slot name="meta" /></div>
    <div v-if="$slots.value" class="shrink-0 text-right">
      <p class="text-sm font-bold tabular-nums"><slot name="value" /></p>
      <p v-if="$slots.subvalue" class="text-xs text-slate-400 tabular-nums"><slot name="subvalue" /></p>
    </div>
  </div>
</template>
