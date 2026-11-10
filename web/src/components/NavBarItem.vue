<script setup lang="ts">
import { computed, type Component } from 'vue'
import { filledIcon } from '@/icons/filled'
import { scrollShellToTop } from '@/utils/scrollToTop'

// One destination of the bottom bar of a phone: the icon sits in a pill that is filled when the destination is the
// current one (or its sheet is open), so the active place reads from the shape and not from the colour alone.
// A link when it goes to a route, a button when it opens something. Touching the link of the page already shown brings
// that page back to its top.
const props = defineProps<{ icon: Component; label: string; active: boolean; to?: string; expanded?: boolean }>()
// The current destination is drawn in the filled version of its icon
const shown = computed(() => (props.active ? filledIcon(props.icon) : props.icon))
const onLinkClick = () => {
  if (props.active) scrollShellToTop()
}
</script>

<template>
  <router-link v-if="to" :to="to" class="nav-item" :class="{ 'nav-item-active': active }" :aria-current="active ? 'page' : undefined" @click="onLinkClick">
    <span class="nav-pill"><component :is="shown" class="h-5 w-5" aria-hidden="true" /></span>
    <span>{{ label }}</span>
  </router-link>
  <button v-else type="button" class="nav-item h-full w-full" :class="{ 'nav-item-active': active }" aria-haspopup="dialog" :aria-expanded="expanded">
    <span class="nav-pill"><component :is="shown" class="h-5 w-5" aria-hidden="true" /></span>
    <span>{{ label }}</span>
  </button>
</template>
