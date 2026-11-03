<script setup lang="ts">
import type { Component } from 'vue'

// One destination of the bottom bar of a phone: the icon sits in a pill that is filled when the destination is the
// current one (or its sheet is open), so the active place reads from the shape and not from the colour alone.
// A link when it goes to a route, a button when it opens something.
defineProps<{ icon: Component; label: string; active: boolean; to?: string; expanded?: boolean }>()
</script>

<template>
  <router-link v-if="to" :to="to" class="nav-item" :class="{ 'nav-item-active': active }" :aria-current="active ? 'page' : undefined">
    <span class="nav-pill"><component :is="icon" class="h-5 w-5" aria-hidden="true" /></span>
    <span>{{ label }}</span>
  </router-link>
  <button v-else type="button" class="nav-item w-full" :class="{ 'nav-item-active': active }" aria-haspopup="dialog" :aria-expanded="expanded">
    <span class="nav-pill"><component :is="icon" class="h-5 w-5" aria-hidden="true" /></span>
    <span>{{ label }}</span>
  </button>
</template>
