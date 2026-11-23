<script setup lang="ts">
// A month of a list: its name, a figure for the month on the right once it is known (the sub-total), then the rows.
// The default slot holds the rows, `summary` replaces the plain figure when a month needs more than one.
defineProps<{ label: string; subtotal?: string | null }>()
</script>

<template>
  <section class="space-y-2">
    <div class="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-0.5 px-1 pt-2">
      <h2 class="text-sm font-bold capitalize text-white">{{ label }}</h2>
      <slot name="summary">
        <p v-if="subtotal" class="text-xs font-semibold tabular-nums text-slate-300">{{ subtotal }}</p>
      </slot>
    </div>
    <slot />
  </section>
</template>
