<script setup lang="ts">
// The quiet line under a row's title, built from explicit groups. Every direct child of the slot is a group (one line on a
// phone) and the parts inside a group are read as one sentence, joined by "·". A group with `data-plain` carries a state
// or a control (a status text, a toggle): its parts and the group itself have no "·" before them. From `lg` the groups
// share one line, joined by "·" too. A "·" is only ever written between two parts of one line, never at the edge of a line.
</script>

<template>
  <span
    class="flex flex-col items-start gap-y-0.5 lg:flex-row lg:items-baseline lg:gap-x-1.5 lg:gap-y-0
      [&>*]:flex [&>*]:min-w-0 [&>*]:max-w-full [&>*]:items-baseline [&>*]:gap-x-1.5 [&>*]:whitespace-nowrap
      [&>*:not([data-plain])>*+*]:before:mr-1.5 [&>*:not([data-plain])>*+*]:before:content-['·']
      lg:[&>*+*:not([data-plain])]:before:content-['·']"
  >
    <slot />
  </span>
</template>
