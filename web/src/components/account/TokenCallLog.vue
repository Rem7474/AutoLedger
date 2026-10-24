<script setup lang="ts">
import { ref } from 'vue'
import { Activity, Check, X } from 'lucide-vue-next'
import { api, type APITokenCall } from '@/services/api'
import { t } from '@/i18n'
import { useVehicleStore } from '@/stores/vehicle'
import { describeCall } from '@/utils/tokenCalls'
import { describeRelativeTime } from '@/utils/userAgent'
import LoadError from '@/components/LoadError.vue'

const props = defineProps<{ tokenId: string }>()

const vehicleStore = useVehicleStore()
const open = ref(false)
const loading = ref(false)
const loadError = ref<string | null>(null)
const calls = ref<APITokenCall[] | null>(null)

async function load() {
  loading.value = true
  loadError.value = null
  try {
    calls.value = (await api.getAPITokenCalls(props.tokenId)).calls
  } catch (err) {
    loadError.value = (err as Error)?.message ?? ''
  } finally {
    loading.value = false
  }
}

// The calls are fetched when the list is opened, then again each time it is reopened.
async function toggle() {
  open.value = !open.value
  if (open.value) await load()
}
</script>

<template>
  <div class="mt-2">
    <button type="button" class="tap-text inline-flex items-center gap-1 text-xs text-sky-400 hover:text-sky-300" :aria-expanded="open" @click="toggle">
      <Activity class="h-3.5 w-3.5" aria-hidden="true" />
      {{ open ? t('account.tokens.activityHide') : t('account.tokens.activity') }}
    </button>
    <div v-if="open" class="mt-2 space-y-2">
      <p class="text-xs text-slate-400">{{ t('account.tokens.activityHelp') }}</p>
      <p v-if="loading" class="text-xs text-slate-400" role="status">{{ t('common.loading') }}</p>
      <LoadError v-else-if="loadError !== null" :message="loadError" @retry="load" />
      <p v-else-if="calls && !calls.length" class="text-xs text-slate-400">{{ t('account.tokens.activityEmpty') }}</p>
      <ul v-else-if="calls" class="max-h-64 overflow-y-auto divide-y divide-slate-800 rounded-lg border border-slate-800">
        <li v-for="(c, i) in calls" :key="i" class="flex items-start gap-2 px-2 py-1.5 text-xs">
          <component :is="describeCall(c, vehicleStore.vehicles).ok ? Check : X" class="mt-0.5 h-3.5 w-3.5 shrink-0" :class="describeCall(c, vehicleStore.vehicles).ok ? 'text-success-400' : 'text-danger-400'" :aria-label="describeCall(c, vehicleStore.vehicles).ok ? t('account.tokens.activityOk') : t('account.tokens.activityError')" role="img" />
          <div class="min-w-0 flex-1">
            <p class="truncate font-mono text-slate-200" :title="c.tool">{{ c.tool }}</p>
            <p class="truncate text-slate-400">
              <span v-if="describeCall(c, vehicleStore.vehicles).vehicle">{{ describeCall(c, vehicleStore.vehicles).vehicle }} · </span>{{ describeCall(c, vehicleStore.vehicles).duration }} · {{ describeRelativeTime(c.at) }}
            </p>
          </div>
        </li>
      </ul>
    </div>
  </div>
</template>
