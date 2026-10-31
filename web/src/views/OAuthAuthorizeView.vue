<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { AlertCircle, Check, ShieldCheck, X } from 'lucide-vue-next'
import { api, type OAuthAuthorizeParams, type OAuthAuthorizePreview } from '@/services/api'
import { t } from '@/i18n'
import { canApproveGrant, grantedVehicleIds, toggleVehicle, type GrantVehicleMode } from '@/utils/oauthVehicles'

const route = useRoute()

const first = (value: unknown): string => (Array.isArray(value) ? String(value[0] ?? '') : typeof value === 'string' ? value : '')

// The assistant states these in the address it sent the user to; the server checks them again on the decision.
const params = computed<OAuthAuthorizeParams>(() => {
  const q = route.query
  const out: OAuthAuthorizeParams = {
    client_id: first(q.client_id),
    redirect_uri: first(q.redirect_uri),
    response_type: first(q.response_type),
    code_challenge: first(q.code_challenge),
    code_challenge_method: first(q.code_challenge_method),
  }
  for (const key of ['scope', 'state', 'resource'] as const) {
    if (first(q[key])) out[key] = first(q[key])
  }
  return out
})

const preview = ref<OAuthAuthorizePreview | null>(null)
const error = ref('')
const loading = ref(true)
const deciding = ref(false)
const finished = ref<'approve' | 'deny' | null>(null)

// Which vehicles the access covers: all of them by default, or the ones the user ticks.
const vehicleMode = ref<GrantVehicleMode>('all')
const selected = ref<ReadonlySet<string>>(new Set())
const choices = computed(() => preview.value?.vehicles ?? [])
// With a single vehicle there is nothing to choose.
const canChoose = computed(() => choices.value.length > 1)
const approvable = computed(() => !canChoose.value || canApproveGrant(vehicleMode.value, selected.value, choices.value))

onMounted(async () => {
  try {
    preview.value = await api.previewOAuthAuthorization(params.value)
  } catch (err) {
    error.value = (err as Error)?.message || t('oauth.authorize.invalid')
  } finally {
    loading.value = false
  }
})

async function decide(decision: 'approve' | 'deny') {
  deciding.value = true
  error.value = ''
  try {
    const vehicleIds = decision === 'approve' && canChoose.value ? grantedVehicleIds(vehicleMode.value, selected.value, choices.value) : null
    const { redirect_url } = await api.decideOAuthAuthorization(params.value, decision, vehicleIds)
    finished.value = decision
    // Back to the assistant: a local address it listens on, or the application it opens.
    window.location.assign(redirect_url)
  } catch (err) {
    error.value = (err as Error)?.message || t('oauth.authorize.failed')
  } finally {
    deciding.value = false
  }
}
</script>

<template>
  <div class="max-w-lg mx-auto py-8">
    <div class="rounded-2xl border border-slate-800 bg-slate-900 p-6 space-y-5">
      <div class="flex items-center gap-3">
        <ShieldCheck class="w-6 h-6 text-rose-400 shrink-0" />
        <h1 class="text-xl font-bold text-white">{{ t('oauth.authorize.title') }}</h1>
      </div>

      <p v-if="loading" class="text-sm text-slate-400">{{ t('common.loading') }}</p>

      <div v-else-if="error && !preview" class="p-3 bg-danger-500/10 border border-danger-500/20 rounded-xl flex items-start gap-2 text-sm text-danger-400" role="alert">
        <AlertCircle class="w-4 h-4 shrink-0 mt-0.5" />
        <span>{{ error }}</span>
      </div>

      <p v-else-if="finished" class="text-sm text-slate-300" role="status">
        {{ finished === 'approve' ? t('oauth.authorize.approved') : t('oauth.authorize.denied') }}
      </p>

      <template v-else-if="preview">
        <p class="text-sm text-slate-300">
          {{ t('oauth.authorize.intro', { name: preview.client_name }) }}
        </p>

        <div>
          <h2 class="text-xs font-semibold text-slate-400 mb-2">{{ t('oauth.authorize.canTitle') }}</h2>
          <ul class="space-y-1.5 text-sm text-slate-200">
            <li class="flex items-start gap-2"><Check class="w-4 h-4 text-success-400 shrink-0 mt-0.5" />{{ t('oauth.authorize.canVehicles') }}</li>
            <li class="flex items-start gap-2"><Check class="w-4 h-4 text-success-400 shrink-0 mt-0.5" />{{ t('oauth.authorize.canCosts') }}</li>
            <li class="flex items-start gap-2"><Check class="w-4 h-4 text-success-400 shrink-0 mt-0.5" />{{ t('oauth.authorize.canGaps') }}</li>
            <li class="flex items-start gap-2"><Check class="w-4 h-4 text-success-400 shrink-0 mt-0.5" />{{ t('oauth.authorize.canNames') }}</li>
          </ul>
        </div>
        <fieldset v-if="canChoose" class="space-y-2 min-w-0">
          <legend class="text-xs font-semibold text-slate-400 mb-2">{{ t('oauth.authorize.vehiclesTitle') }}</legend>
          <label class="flex items-start gap-2 text-sm text-slate-200">
            <input v-model="vehicleMode" type="radio" name="vehicle-mode" value="all" class="mt-1 shrink-0" />
            <span>{{ t('oauth.authorize.vehiclesAll') }}</span>
          </label>
          <label class="flex items-start gap-2 text-sm text-slate-200">
            <input v-model="vehicleMode" type="radio" name="vehicle-mode" value="some" class="mt-1 shrink-0" />
            <span>{{ t('oauth.authorize.vehiclesSome') }}</span>
          </label>
          <ul v-if="vehicleMode === 'some'" class="space-y-1 pl-6 min-w-0">
            <li v-for="v in choices" :key="v.id" class="min-w-0">
              <label class="flex items-center gap-2 text-sm text-slate-200 min-w-0">
                <input type="checkbox" :checked="selected.has(v.id)" class="shrink-0" @change="selected = toggleVehicle(selected, v.id)" />
                <span class="truncate" :title="v.name">{{ v.name }}</span>
                <span class="text-xs text-slate-400 shrink-0">{{ t(`oauth.authorize.vehiclesRole.${v.role}`) }}</span>
              </label>
            </li>
          </ul>
          <p v-if="!approvable" class="text-xs text-warning-400" role="status">{{ t('oauth.authorize.vehiclesNone') }}</p>
        </fieldset>
        <div>
          <h2 class="text-xs font-semibold text-slate-400 mb-2">{{ t('oauth.authorize.cannotTitle') }}</h2>
          <ul class="space-y-1.5 text-sm text-slate-200">
            <li class="flex items-start gap-2"><X class="w-4 h-4 text-danger-400 shrink-0 mt-0.5" />{{ t('oauth.authorize.cannotWrite') }}</li>
            <li class="flex items-start gap-2"><X class="w-4 h-4 text-danger-400 shrink-0 mt-0.5" />{{ t('oauth.authorize.cannotPrivate') }}</li>
          </ul>
        </div>

        <p class="text-xs text-slate-400">
          {{ t('oauth.authorize.returnTo', { host: preview.redirect_host }) }}
          {{ t('oauth.authorize.unverified') }}
        </p>
        <p class="text-xs text-slate-400">{{ t('oauth.authorize.revokeHint') }}</p>

        <div v-if="error" class="p-3 bg-danger-500/10 border border-danger-500/20 rounded-xl text-sm text-danger-400" role="alert">{{ error }}</div>

        <div class="flex gap-3">
          <button type="button" :disabled="deciding || !approvable" class="btn btn-lg btn-primary flex-1" @click="decide('approve')">{{ t('oauth.authorize.approve') }}</button>
          <button type="button" :disabled="deciding" class="btn btn-lg btn-secondary flex-1" @click="decide('deny')">{{ t('oauth.authorize.deny') }}</button>
        </div>
      </template>
    </div>
  </div>
</template>
