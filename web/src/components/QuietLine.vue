<script setup lang="ts">
// The quiet line under a row's title: short parts (a date, a distance, a count, a state) read as one sentence. Every direct
// child of the slot is a part. A part is never cut and a line break falls between two parts, with the "·" that
// separates it from the previous part written at the start of the part, so a separator never ends or starts a line alone.
</script>

<template>
  <span class="inline-flex flex-wrap items-baseline gap-x-1.5 whitespace-normal [&>*]:min-w-0 [&>*]:whitespace-nowrap [&>*+*]:before:mr-1.5 [&>*+*]:before:content-['·']">
    <slot />
  </span>
</template>
