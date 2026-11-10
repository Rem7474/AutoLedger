<script setup lang="ts">
import { Receipt, Users } from '@/icons'
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
      class="chip"
      :class="{ 'chip-selected': hasTollOnly }"
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
      class="chip"
      :class="{ 'chip-selected': carpoolFilter === 'yes' }"
      :title="$t('drives.drivesView.carpooledDrivesTitle')"
    >
      <Users class="w-3.5 h-3.5" />
      {{ $t('drives.drivesView.carpooled') }}
    </button>
    <button
      @click="carpoolFilter = carpoolFilter === 'no' ? '' : 'no'"
      :aria-pressed="carpoolFilter === 'no'"
      class="chip"
      :class="{ 'chip-selected': carpoolFilter === 'no' }"
      :title="$t('drives.drivesView.notCarpooledTitle')"
    >
      {{ $t('drives.drivesView.notCarpooled') }}
    </button>
    <template v-if="proPersoEnabled">
      <span class="hidden h-5 w-px bg-slate-700 sm:block" aria-hidden="true"></span>
      <button
        @click="selectedTag = ''"
        :aria-pressed="selectedTag === ''"
        class="chip"
        :class="{ 'chip-selected': selectedTag === '' }"
      >
        {{ $t('drives.drivesView.all') }}
      </button>
      <button
        @click="selectedTag = 'Pro'"
        :aria-pressed="selectedTag === 'Pro'"
        class="chip"
        :class="{ 'chip-selected': selectedTag === 'Pro' }"
      >
        {{ $t('drives.drivesView.work') }}
      </button>
      <button
        @click="selectedTag = 'Perso'"
        :aria-pressed="selectedTag === 'Perso'"
        class="chip"
        :class="{ 'chip-selected': selectedTag === 'Perso' }"
      >
        {{ $t('drives.drivesView.personal') }}
      </button>
    </template>
  </div>
</template>
