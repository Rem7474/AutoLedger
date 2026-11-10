<script setup lang="ts">
import { computed } from 'vue'
import { Paperclip } from '@/icons'
import type { CostEntryLine } from '@/services/api'
import { formatAmount } from '@/currency'
import { fmt } from '@/components/dashboard/energyStats'
import CardOpenButton from '@/components/CardOpenButton.vue'
import ListItem from '@/components/ListItem.vue'
import { amountValue, entryDetail, entryLabel, entryTarget, iconOfEntry, themeOfEntry } from '@/utils/costEntries'
import { THEME_STYLES } from '@/utils/themes'

// One line of the cost journal: the glyph of its theme, what it is, where or how much was bought, its figure, and a
// paperclip when a receipt is attached. The whole card opens the record it comes from, or the screen that holds it.
const props = defineProps<{ entry: CostEntryLine; currency: string; readonly?: boolean }>()
const emit = defineEmits<{ open: [entry: CostEntryLine] }>()
const target = computed(() => entryTarget(props.entry))
const style = computed(() => THEME_STYLES[themeOfEntry(props.entry.category)])
const icon = computed(() => iconOfEntry(props.entry.category))
const detail = computed(() => entryDetail(props.entry, (n) => fmt(n, 1)))
</script>

<template>
  <ListItem data-testid="cost-entry">
    <template #open>
      <CardOpenButton v-if="target.type === 'edit' && !readonly" :label="entryLabel(entry)" @click="emit('open', entry)" />
      <router-link v-else-if="target.type === 'link'" :to="target.to" class="card-open absolute inset-0 rounded-[inherit]" :aria-label="entryLabel(entry)" />
    </template>
    <template #lead>
      <span class="grid size-10 place-items-center rounded-full" :class="style.tint" aria-hidden="true">
        <component :is="icon" class="size-5" />
      </span>
    </template>
    <template #title>
      <span :title="entryLabel(entry)">{{ entryLabel(entry) }}</span>
    </template>
    <template v-if="detail" #subtitle>
      <span :title="detail">{{ detail }}</span>
    </template>
    <template v-if="entry.has_receipt" #meta>
      <Paperclip class="size-4 text-slate-400" :aria-label="$t('costs.list.receipt')" role="img" />
    </template>
    <template #value>
      <span :class="style.text">{{ formatAmount(amountValue(entry.amount), currency) }}</span>
    </template>
  </ListItem>
</template>
