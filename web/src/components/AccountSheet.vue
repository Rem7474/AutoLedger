<script setup lang="ts">
import { watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LogOut, UserRound, Settings } from '@/icons'
import ModalShell from '@/components/ModalShell.vue'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import { useAuthStore } from '@/stores/auth'
import { APP_NAME } from '@/brand'
import { APP_VERSION } from '@/version'

// What belongs to the person and not to a vehicle: the account page, the language, signing out. On a phone it replaces the
// "More" sheet of the bottom bar; it opens from the profile button of the top bar.
const open = defineModel<boolean>('open', { required: true })
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// The sheet closes when the page changes, so that the system Back guard is replaced by the navigation, not undone by it
watch(() => route.path, () => (open.value = false))

function go(path: string) {
  void router.push(path)
}

function signOut() {
  authStore.logout()
  void router.push('/login')
}
</script>

<template>
  <ModalShell v-model:open="open" :title="authStore.user?.email ?? $t('shell.nav.account')" :icon="UserRound" size="sm" data-testid="account-sheet">
    <ul class="space-y-1">
      <li>
        <button type="button" class="flex min-h-12 w-full items-center gap-3 rounded-xl px-3 text-base font-medium text-slate-200 hover:bg-slate-800" @click="go('/account')">
          <Settings class="h-5 w-5" aria-hidden="true" />
          {{ $t('shell.nav.account') }}
        </button>
      </li>
    </ul>
    <LanguageSwitcher />
    <button type="button" class="flex min-h-12 w-full items-center gap-3 rounded-xl px-3 text-base font-medium text-danger-300 hover:bg-danger-500/10" @click="signOut">
      <LogOut class="h-5 w-5" aria-hidden="true" />
      {{ $t('shell.navigation.signOut') }}
    </button>
    <p v-if="APP_VERSION !== 'dev'" class="text-center font-mono text-xs text-slate-400">{{ APP_NAME }} {{ APP_VERSION }}</p>
  </ModalShell>
</template>
