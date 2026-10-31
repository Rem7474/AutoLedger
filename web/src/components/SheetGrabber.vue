<script setup lang="ts">
// The small bar at the top of a bottom sheet: it says the sheet can be pulled down to close. Decorative, the close
// button stays the accessible way out; the caller hides it where the panel is a centred dialog.
</script>

<template>
  <div class="flex shrink-0 justify-center touch-none" aria-hidden="true">
    <span class="h-1 w-10 rounded-full bg-slate-600"></span>
  </div>
</template>
