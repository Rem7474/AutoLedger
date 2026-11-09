<script setup lang="ts">
import { computed } from 'vue'
import { Paperclip } from 'lucide-vue-next'
import type { CostEntryLine } from '@/services/api'
import { formatAmount } from '@/currency'
import { fmt } from '@/components/dashboard/energyStats'
import { amountValue, entryDetail, entryLabel, iconOfEntry, themeOfEntry } from '@/utils/costEntries'
import { THEME_STYLES } from '@/utils/themes'

// One line of the cost journal: the glyph of its theme, what it is, where or how much was bought, its figure, and a
// paperclip when a receipt is attached.
const props = defineProps<{ entry: CostEntryLine; currency: string }>()
const style = computed(() => THEME_STYLES[themeOfEntry(props.entry.category)])
const icon = computed(() => iconOfEntry(props.entry.category))
const detail = computed(() => entryDetail(props.entry, (n) => fmt(n, 1)))
</script>

<template>
  <div class="card flex items-center gap-3 px-3 py-2.5" data-testid="cost-entry">
    <span class="grid size-10 shrink-0 place-items-center rounded-full" :class="style.tint" aria-hidden="true">
      <component :is="icon" class="size-5" />
    </span>
    <div class="min-w-0 flex-1">
      <p class="truncate text-sm font-semibold text-white" :title="entryLabel(entry)">{{ entryLabel(entry) }}</p>
      <p v-if="detail" class="truncate text-xs text-slate-400" :title="detail">{{ detail }}</p>
    </div>
    <Paperclip v-if="entry.has_receipt" class="size-4 shrink-0 text-slate-400" :aria-label="$t('costs.list.receipt')" role="img" />
    <p class="shrink-0 text-sm font-bold tabular-nums" :class="style.text">{{ formatAmount(amountValue(entry.amount), currency) }}</p>
  </div>
</template>
