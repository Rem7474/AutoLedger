<script setup lang="ts">
import PageHeader from '@/components/PageHeader.vue'
import { UserRound as PageIcon } from 'lucide-vue-next'
import { t } from '@/i18n'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import DistanceUnitSwitcher from '@/components/DistanceUnitSwitcher.vue'
import VolumeUnitSwitcher from '@/components/VolumeUnitSwitcher.vue'
import ApiTokensSection from '@/components/account/ApiTokensSection.vue'
import TariffPlansPanel from '@/components/tariffs/TariffPlansPanel.vue'
import { computed, onMounted, ref } from 'vue'
import { useRefreshHandler } from '@/composables/usePullToRefresh'
import { useRoute, useRouter } from 'vue-router'
import { KeyRound, Laptop, LogOut, SlidersHorizontal, Smartphone, UserRound } from 'lucide-vue-next'
import { api } from '@/services/api'
import { useAuthStore } from '@/stores/auth'
import { usePreferencesStore } from '@/stores/preferences'
import { useConfirm } from '@/composables/useConfirm'
import { describeRelativeTime, describeUserAgent } from '@/utils/userAgent'
import { ssoLinkNotice } from '@/utils/sso'
import { COLLAPSED_SESSIONS, visibleSessions } from '@/utils/sessionList'

interface Session {
  id: string
  started_at: string
  last_used_at: string
  ip?: string
  user_agent?: string
  current: boolean
}

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const prefs = usePreferencesStore()
const { showConfirm } = useConfirm()

const sessions = ref<Session[]>([])
const showAllSessions = ref(false)
const listedSessions = computed(() => visibleSessions(sessions.value, showAllSessions.value))
const loading = ref(true)
const loadError = ref('')
const busyId = ref<string | null>(null)
const busyAll = ref(false)
const sessionError = ref('')

const hasPassword = computed(() => authStore.user?.has_password === true)

// Linking a single sign-on is offered to an account that signs in with a password, on an instance that has one
const oidcEnabled = ref(false)
const oidcProviderName = ref('SSO')
const oidcLinked = computed(() => authStore.user?.oidc_linked === true)
const ssoNotice = ref(ssoLinkNotice(route.query.sso))

const current = ref('')
const next = ref('')
const confirmation = ref('')
const saving = ref(false)
const passwordError = ref('')
const passwordDone = ref('')

const isMobile = (ua?: string) => /Android|iPhone|iPad|iPod|Mobile/i.test(ua ?? '')

async function loadSsoConfig() {
  try {
    const config = await api.getAuthConfig()
    oidcEnabled.value = !!config.oidc_enabled
    oidcProviderName.value = config.oidc_provider_name || 'SSO'
  } catch {
    // the link is simply not offered
  }
  // The server sends the user back here with the outcome, on a fresh page load that has read the profile again.
  if (route.query.sso) router.replace({ query: {} })
}

// `silent` reloads behind the list that is already shown (pull to refresh) instead of swapping it for the loading state.
async function load(silent = false) {
  if (!silent) loading.value = true
  loadError.value = ''
  try {
    sessions.value = await api.getSessions()
  } catch (err: any) {
    loadError.value = err?.message || t('account.loadFailed')
  } finally {
    loading.value = false
  }
}

// The server has cleared the cookies when the current session is closed: only the local state is left to reset.
async function endLocalSession() {
  await authStore.logout()
  router.push('/login')
}

async function revoke(session: Session) {
  sessionError.value = ''
  if (!session.current) {
    const ok = await showConfirm({
      title: t('account.disconnectThisDevice'),
      message: t('account.disconnectDeviceMessage', { device: describeUserAgent(session.user_agent) }),
      confirmText: t('account.disconnect'),
      type: 'warning',
    })
    if (!ok) return
  }
  busyId.value = session.id
  try {
    await api.revokeSession(session.id)
    if (session.current) {
      await endLocalSession()
      return
    }
    sessions.value = sessions.value.filter((s) => s.id !== session.id)
  } catch (err: any) {
    sessionError.value = err?.message || t('account.disconnectFailed')
  } finally {
    busyId.value = null
  }
}

async function logoutEverywhere() {
  sessionError.value = ''
  const ok = await showConfirm({
    title: t('account.signOutAllTitle'),
    message: t('account.signOutAllMessage'),
    confirmText: t('account.signOutAll'),
    type: 'danger',
  })
  if (!ok) return
  busyAll.value = true
  try {
    await api.logoutAll()
    await endLocalSession()
  } catch (err: any) {
    sessionError.value = err?.message || t('account.disconnectAllFailed')
  } finally {
    busyAll.value = false
  }
}

async function changePassword() {
  passwordError.value = ''
  passwordDone.value = ''
  if (next.value.length < 8) {
    passwordError.value = t('account.passwordTooShort')
    return
  }
  if (new TextEncoder().encode(next.value).length > 72) {
    passwordError.value = t('account.passwordTooLong')
    return
  }
  if (next.value !== confirmation.value) {
    passwordError.value = t('account.passwordsDiffer')
    return
  }
  saving.value = true
  try {
    const res = await api.changePassword({ current_password: current.value, new_password: next.value })
    current.value = next.value = confirmation.value = ''
    const others = res?.sessions_revoked ?? 0
    passwordDone.value = others > 0
      ? t('account.passwordChangedOthers', { count: others })
      : t('account.passwordChanged')
    await load()
  } catch (err: any) {
    passwordError.value = err?.message || t('account.passwordChangeFailed')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  load()
  loadSsoConfig()
})
useRefreshHandler(() => load(true))
</script>

<template>
  <div class="mx-auto max-w-3xl space-y-6">
    <PageHeader :title="$t('shell.nav.account')" :icon="PageIcon">
      {{ $t('account.accountView.signInPasswordAndConnected') }}
    </PageHeader>

    <section class="card p-5" aria-labelledby="account-identity">
      <h2 id="account-identity" class="mb-3 flex items-center gap-2 text-sm font-bold text-white">
        <UserRound class="h-4 w-4 text-slate-400" aria-hidden="true" />
        {{ $t('account.accountView.identity') }}
      </h2>
      <dl class="divide-y divide-slate-800">
        <div class="settings-row">
          <dt class="settings-row-label">{{ $t('account.language') }}</dt>
          <dd><LanguageSwitcher /></dd>
        </div>
        <div class="settings-row">
          <dt class="settings-row-label">{{ $t('account.accountView.emailAddress') }}</dt>
          <dd class="min-w-0 break-all text-right text-slate-300">{{ authStore.user?.email }}</dd>
        </div>
        <div class="settings-row">
          <dt class="settings-row-label">{{ $t('account.accountView.signIn') }}</dt>
          <dd class="text-right text-slate-300">{{ hasPassword ? $t('account.localPassword') : $t('account.ssoProvider') }}</dd>
        </div>
      </dl>
      <p
        v-if="ssoNotice"
        role="status"
        class="mt-3 rounded-xl border px-3 py-2 text-xs"
        :class="ssoNotice.tone === 'success' ? 'border-success-500/30 bg-success-500/10 text-success-300' : 'border-warning-500/30 bg-warning-500/10 text-warning-300'"
      >
        {{ $t(ssoNotice.key) }}
      </p>
      <div v-if="oidcEnabled && hasPassword" class="mt-3 flex flex-wrap items-center justify-between gap-3 border-t border-slate-800 pt-3">
        <p class="max-w-md text-xs text-slate-400">
          {{ oidcLinked ? $t('account.sso.linked', { provider: oidcProviderName }) : $t('account.sso.linkHint', { provider: oidcProviderName }) }}
        </p>
        <a v-if="!oidcLinked" href="/api/auth/oidc/link" class="btn btn-secondary">{{ $t('account.sso.link', { provider: oidcProviderName }) }}</a>
      </div>
    </section>

    <section class="card p-5" aria-labelledby="account-preferences">
      <h2 id="account-preferences" class="mb-3 flex items-center gap-2 text-sm font-bold text-white">
        <SlidersHorizontal class="h-4 w-4 text-slate-400" aria-hidden="true" />
        {{ $t('account.accountView.preferences') }}
      </h2>
      <div class="divide-y divide-slate-800">
        <label class="settings-row cursor-pointer !items-start">
          <span class="settings-row-label">
            <span class="block">{{ $t('account.accountView.proPersoTitle') }}</span>
            <span class="block text-xs font-normal text-slate-400">{{ $t('account.accountView.proPersoHelp') }}</span>
          </span>
          <input
            type="checkbox"
            role="switch"
            class="mt-1 h-5 w-9 shrink-0 cursor-pointer appearance-none rounded-full bg-slate-700 transition-colors checked:bg-rose-500 relative before:absolute before:left-0.5 before:top-0.5 before:h-4 before:w-4 before:rounded-full before:bg-white before:transition-transform checked:before:translate-x-4"
            :checked="prefs.proPersoEnabled"
            @change="prefs.setProPersoEnabled(($event.target as HTMLInputElement).checked)"
          />
        </label>
        <div class="settings-row">
          <span class="settings-row-label">{{ $t('account.distanceUnit') }}</span>
          <DistanceUnitSwitcher />
        </div>
        <div class="settings-row">
          <span class="settings-row-label">{{ $t('account.volumeUnit') }}</span>
          <VolumeUnitSwitcher />
        </div>
      </div>
      <p class="mt-3 text-xs text-slate-400">{{ $t('account.currencyPerVehicle') }}</p>
    </section>

    <section v-if="hasPassword" class="card p-5" aria-labelledby="account-password">
      <h2 id="account-password" class="mb-3 flex items-center gap-2 text-sm font-bold text-white">
        <KeyRound class="h-4 w-4 text-slate-400" aria-hidden="true" />
        {{ $t('account.accountView.password') }}
      </h2>
      <form class="space-y-4" novalidate @submit.prevent="changePassword">
        <div>
          <label for="pw-current" class="field-label">{{ $t('account.accountView.currentPassword') }}</label>
          <input id="pw-current" v-model="current" type="password" autocomplete="current-password" class="field" />
        </div>
        <div>
          <label for="pw-new" class="field-label">{{ $t('account.accountView.newPassword') }}</label>
          <input id="pw-new" v-model="next" type="password" autocomplete="new-password" class="field" aria-describedby="pw-hint" />
          <p id="pw-hint" class="mt-1 text-xs text-slate-400">{{ $t('account.accountView.8CharactersMinimum72Bytes') }}</p>
        </div>
        <div>
          <label for="pw-confirm" class="field-label">{{ $t('account.accountView.confirmTheNewPassword') }}</label>
          <input id="pw-confirm" v-model="confirmation" type="password" autocomplete="new-password" class="field" />
        </div>
        <p v-if="passwordError" role="alert" class="rounded-lg border border-danger-500/30 bg-danger-500/10 px-3 py-2 text-xs text-danger-300">{{ passwordError }}</p>
        <p v-if="passwordDone" role="status" class="rounded-lg border border-success-500/30 bg-success-500/10 px-3 py-2 text-xs text-success-300">{{ passwordDone }}</p>
        <button
          type="submit"
          :disabled="saving || !current || !next || !confirmation"
          class="btn btn-lg btn-primary min-h-12"
        >
          {{ saving ? $t('account.saving') : $t('account.changePassword') }}
        </button>
        <p class="text-xs text-slate-400">{{ $t('account.accountView.otherDevicesAreSignedOut') }}</p>
      </form>
    </section>

    <TariffPlansPanel />

    <ApiTokensSection />

    <section class="card p-5" aria-labelledby="account-sessions">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 id="account-sessions" class="flex items-center gap-2 text-sm font-bold text-white">
          <Laptop class="h-4 w-4 text-slate-400" aria-hidden="true" />
          {{ $t('account.accountView.connectedDevices') }}
        </h2>
        <button
          type="button"
          :disabled="busyAll || loading"
          class="min-h-11 rounded-xl border border-rose-500/30 bg-rose-500/10 px-4 text-xs font-semibold text-rose-300 hover:bg-rose-500/20 disabled:opacity-50"
          @click="logoutEverywhere"
        >
          {{ $t('account.accountView.signOutEverywhere') }}
        </button>
      </div>

      <p v-if="loading" class="text-sm text-slate-400">{{ $t('account.accountView.loading') }}</p>
      <p v-else-if="loadError" role="alert" class="text-sm text-danger-300">{{ loadError }}</p>
      <ul v-else class="space-y-2">
        <li
          v-for="s in listedSessions"
          :key="s.id"
          class="flex items-center justify-between gap-3 rounded-xl border bg-slate-950/60 p-3"
          :class="s.current ? 'border-success-500/30' : 'border-slate-800'"
        >
          <div class="flex min-w-0 items-start gap-3">
            <component :is="isMobile(s.user_agent) ? Smartphone : Laptop" class="mt-0.5 h-5 w-5 shrink-0 text-slate-400" aria-hidden="true" />
            <div class="min-w-0">
              <p class="flex flex-wrap items-center gap-2 text-sm font-semibold text-white">
                {{ describeUserAgent(s.user_agent) }}
                <span v-if="s.current" class="rounded-full bg-success-500/15 px-2 py-0.5 text-xs font-bold text-success-300">{{ $t('account.accountView.thisDevice') }}</span>
              </p>
              <p class="text-xs text-slate-400">
                {{ $t('account.accountView.active', { value: describeRelativeTime(s.last_used_at) }) }}<template v-if="s.ip"> · {{ s.ip }}</template>
              </p>
              <p class="text-xs text-slate-400">{{ $t('account.accountView.signedIn', { value: describeRelativeTime(s.started_at) }) }}</p>
            </div>
          </div>
          <button
            type="button"
            :disabled="busyId === s.id"
            class="btn btn-secondary min-h-11 shrink-0"
            :aria-label="s.current ? $t('account.signOutThisDevice') : $t('account.signOutDevice', { device: describeUserAgent(s.user_agent) })"
            @click="revoke(s)"
          >
            <LogOut class="h-3.5 w-3.5" aria-hidden="true" />
            <span class="hidden sm:inline">{{ s.current ? $t('account.signOut') : $t('account.disconnect') }}</span>
          </button>
        </li>
        <li v-if="sessions.length === 0" class="text-sm text-slate-400">{{ $t('account.accountView.noActiveSession') }}</li>
      </ul>
      <button
        v-if="!loading && !loadError && sessions.length > COLLAPSED_SESSIONS"
        type="button"
        class="btn btn-secondary mt-3 w-full justify-center"
        :aria-expanded="showAllSessions"
        @click="showAllSessions = !showAllSessions"
      >
        {{ showAllSessions ? $t('account.accountView.showFewerDevices') : $t('account.accountView.showAllDevices', { count: sessions.length }) }}
      </button>
      <p v-if="sessionError" role="alert" class="mt-3 text-xs text-danger-300">{{ sessionError }}</p>
      <p class="mt-3 text-xs text-slate-400">{{ $t('account.accountView.aSignedOutDeviceKeeps') }}</p>
    </section>
  </div>
</template>
