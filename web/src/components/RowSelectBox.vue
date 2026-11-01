<script setup lang="ts">
// The selection mark of a list row (round check on a touch screen): above the row's open button, so a tap on it
// toggles the row instead of opening it.
defineProps<{ id: string; checked: boolean; label: string }>()
const emit = defineEmits<{ toggle: [] }>()
</script>

<template>
  <label :for="`select-${id}`" class="relative z-10 tap -ml-1 p-1 shrink-0 flex items-center cursor-pointer" :title="label">
    <span class="sr-only">{{ label }}</span>
    <input :id="`select-${id}`" type="checkbox" :checked="checked" class="select-box" @change="emit('toggle')" />
  </label>
</template>
