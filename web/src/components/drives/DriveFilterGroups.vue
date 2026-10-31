<script setup lang="ts">
import { Receipt, Users } from 'lucide-vue-next'
import ToQualifyFilter from '@/components/drives/ToQualifyFilter.vue'

// The toll and Work / Personal filters of the drives list, as one group of toggles. The qualification queue and the
// split between automatic and manual tolls exist only where the toll detection does; manual tolls are always listed.
defineProps<{ unqualifiedCount: number; proPersoEnabled: boolean; tollDetectionActive: boolean }>()
const unqualifiedOnly = defineModel<boolean>('unqualifiedOnly', { required: true })
const hasTollOnly = defineModel<boolean>('hasTollOnly', { required: true })
const tollSource = defineModel<string>('tollSource', { required: true })
const carpoolFilter = defineModel<'' | 'yes' | 'no'>('carpoolFilter', { required: true })
const selectedTag = defineModel<string>('selectedTag', { required: true })
</script>

<template>
  <div id="drives-filter-groups" class="flex flex-wrap items-center gap-2">
    <span class="hidden px-2 text-[0.65rem] font-semibold text-slate-400 sm:inline">{{ $t('drives.drivesView.quickFilters') }}</span>
    <ToQualifyFilter
      v-if="tollDetectionActive"
      :count="unqualifiedCount"
      :active="unqualifiedOnly"
      :title="$t('drives.drivesView.motorwayTypeDrivesWithNo')"
      @toggle="unqualifiedOnly = !unqualifiedOnly"
    />
    <button
      @click="hasTollOnly = !hasTollOnly"
      class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
      :class="hasTollOnly ? 'bg-sky-500/20 text-sky-400 border border-sky-500/30' : 'text-sky-400/80 hover:text-sky-300 border border-transparent'"
      :title="$t('drives.drivesView.drivesWithATollExpense')"
    >
      <Receipt class="w-3.5 h-3.5" />
      {{ $t('drives.drivesView.withToll') }}
    </button>
    <template v-if="hasTollOnly && tollDetectionActive">
      <label for="drives-toll-source" class="sr-only">{{ $t('drives.drivesView.tollSource') }}</label>
      <select
        id="drives-toll-source"
        v-model="tollSource"
        class="field"
      >
        <option value="">{{ $t('drives.drivesView.all') }}</option>
        <option value="AUTO_TOLL">{{ $t('drives.drivesView.auto') }}</option>
        <option value="MANUAL">{{ $t('drives.drivesView.manual') }}</option>
      </select>
    </template>
    <button
      @click="carpoolFilter = carpoolFilter === 'yes' ? '' : 'yes'"
      :aria-pressed="carpoolFilter === 'yes'"
      class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors flex items-center gap-1.5"
      :class="carpoolFilter === 'yes' ? 'bg-sky-500/20 text-sky-400 border border-sky-500/30' : 'text-sky-400/80 hover:text-sky-300 border border-transparent'"
      :title="$t('drives.drivesView.carpooledDrivesTitle')"
    >
      <Users class="w-3.5 h-3.5" />
      {{ $t('drives.drivesView.carpooled') }}
    </button>
    <button
      @click="carpoolFilter = carpoolFilter === 'no' ? '' : 'no'"
      :aria-pressed="carpoolFilter === 'no'"
      class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors"
      :class="carpoolFilter === 'no' ? 'bg-slate-700 text-white border border-slate-600' : 'text-slate-400 hover:text-white border border-transparent'"
      :title="$t('drives.drivesView.notCarpooledTitle')"
    >
      {{ $t('drives.drivesView.notCarpooled') }}
    </button>
    <template v-if="proPersoEnabled">
      <span class="hidden h-5 w-px bg-slate-700 sm:block" aria-hidden="true"></span>
      <button
        @click="selectedTag = ''"
        :aria-pressed="selectedTag === ''"
        class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors"
        :class="selectedTag === '' ? 'bg-slate-700 text-white border border-slate-600' : 'text-slate-400 hover:text-white border border-transparent'"
      >
        {{ $t('drives.drivesView.all') }}
      </button>
      <button
        @click="selectedTag = 'Pro'"
        :aria-pressed="selectedTag === 'Pro'"
        class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors"
        :class="selectedTag === 'Pro' ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : 'text-slate-400 hover:text-white border border-transparent'"
      >
        {{ $t('drives.drivesView.work') }}
      </button>
      <button
        @click="selectedTag = 'Perso'"
        :aria-pressed="selectedTag === 'Perso'"
        class="px-3 py-1.5 rounded-lg text-xs font-semibold transition-colors"
        :class="selectedTag === 'Perso' ? 'bg-success-500/20 text-success-400 border border-success-500/30' : 'text-slate-400 hover:text-white border border-transparent'"
      >
        {{ $t('drives.drivesView.personal') }}
      </button>
    </template>
  </div>
</template>
