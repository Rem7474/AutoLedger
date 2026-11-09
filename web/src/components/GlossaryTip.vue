<script setup lang="ts">
import { computed, ref } from 'vue'
import { Info } from 'lucide-vue-next'
import { t } from '@/i18n'
import ModalShell from '@/components/ModalShell.vue'

// Definition of a cost or consumption term (locales glossary.json). A touch or a click opens it in a small sheet (a hover
// tooltip is out of reach on a phone); the mouse still gets the tooltip, and screen readers get the full text as the label.
const props = defineProps<{ term: string }>()
const label = computed(() => t(`glossary.${props.term}.label`))
const definition = computed(() => t(`glossary.${props.term}.definition`))
const open = ref(false)
</script>

<template>
  <span class="ml-1 inline-flex align-middle">
    <!-- The button is the size of its glyph, so it neither widens the gap to its label nor stretches the line it sits in;
         the hit area (44 px) is a pseudo-element around it -->
    <button
      type="button"
      class="no-tap relative inline-flex items-center justify-center text-slate-400 hover:text-slate-200 focus-visible:text-slate-200 after:absolute after:-inset-3.5 after:content-['']"
      :title="definition"
      :aria-label="`${label}: ${definition}`"
      aria-haspopup="dialog"
      @click.stop="open = true"
    >
      <Info class="size-4" aria-hidden="true" />
    </button>
    <!-- In the body: a card that stacks its content (z-10 over its overlay) would keep the sheet under the bottom bar -->
    <Teleport to="body">
      <ModalShell v-model:open="open" size="sm" :icon="Info" icon-class="text-slate-400" :title="label">
        <p class="text-sm text-slate-300">{{ definition }}</p>
      </ModalShell>
    </Teleport>
  </span>
</template>
