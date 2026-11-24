<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useOfflineStore } from '@/stores/offline'
import { useQuickAddStore } from '@/stores/quickAdd'
import { useVehicleStore } from '@/stores/vehicle'
import { LogOut, Zap, Plus, WifiOff } from '@/icons'
import { APP_NAME } from '@/brand'
import { activeDestination, activeSidebarEntry, GARAGE_DESTINATION, navDestinations } from '@/utils/navigation'
import { quickKindForRoute } from '@/utils/quickAdd'
import NavBarItem from '@/components/NavBarItem.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const vehicleStore = useVehicleStore()
const offlineStore = useOfflineStore()
const quickAdd = useQuickAddStore()

// One list of four destinations feeds the sidebar of a desktop and the bottom bar of a phone (see `navDestinations`).
const destinations = computed(() => navDestinations(vehicleStore.capabilities.drives))
const currentRouteName = computed(() => route.name)
const openQuickAdd = () => quickAdd.open(quickKindForRoute(route.name, vehicleStore.capabilities))

const sidebarActive = computed(() => activeSidebarEntry(destinations.value, currentRouteName.value))
const activeTab = computed(() => activeDestination(destinations.value, currentRouteName.value))
const accountActive = computed(() => currentRouteName.value === 'account')
const leftItems = computed(() => destinations.value.slice(0, 2))
const rightItems = computed(() => destinations.value.slice(2))

const canQuickAdd = computed(() => !!vehicleStore.activeVehicle && vehicleStore.canEdit)
function handleLogout() {
  authStore.logout()
  void router.push('/login')
}

const LINK_CLASS = 'flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium transition-all border border-transparent'
const linkTone = (active: boolean) =>
  active ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20 shadow-sm' : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
</script>

<template>
  <!-- Desktop Sidebar -->
  <aside class="hidden md:flex flex-col w-64 bg-slate-900 border-r border-slate-800 p-4 shrink-0">
    <div class="flex items-center gap-3 px-3 py-4 mb-4">
      <div class="p-2 bg-gradient-to-tr from-rose-500 to-warning-500 rounded-xl shadow-lg shadow-rose-500/20">
        <Zap class="w-6 h-6 text-white" />
      </div>
      <p class="text-xl font-bold tracking-tight bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">
        {{ APP_NAME }}
      </p>
    </div>

    <button
      v-if="canQuickAdd"
      type="button"
      class="btn btn-primary mb-3"
      @click="openQuickAdd"
    >
      <Plus class="h-4 w-4" aria-hidden="true" />
      {{ $t('shell.navigation.quickAdd') }}
    </button>

    <nav :aria-label="$t('shell.navigation.mainNavigation')" class="flex-1 overflow-y-auto">
      <ul class="space-y-0.5">
        <li v-for="item in destinations" :key="item.name">
          <router-link :to="item.path" :aria-current="sidebarActive === item.name ? 'page' : undefined" :class="[LINK_CLASS, linkTone(sidebarActive === item.name)]">
            <component :is="item.icon" class="w-5 h-5" />
            {{ $t(item.labelKey) }}
          </router-link>
        </li>
      </ul>
      <ul class="mt-4 space-y-0.5 border-t border-slate-800 pt-4">
        <li>
          <router-link
            :to="GARAGE_DESTINATION.path"
            :aria-current="sidebarActive === GARAGE_DESTINATION.name ? 'page' : undefined"
            :class="[LINK_CLASS, linkTone(sidebarActive === GARAGE_DESTINATION.name)]"
          >
            <component :is="GARAGE_DESTINATION.icon" class="w-5 h-5" />
            {{ $t(GARAGE_DESTINATION.labelKey) }}
          </router-link>
        </li>
      </ul>
    </nav>

    <div class="pt-4 border-t border-slate-800 mt-auto space-y-1">
      <p v-if="!offlineStore.isOnline" class="flex items-center gap-2 px-3 py-1 text-xs text-warning-300" role="status">
        <WifiOff class="h-3.5 w-3.5 shrink-0" aria-hidden="true" />
        {{ $t('shell.topBar.offline') }}
      </p>
      <div class="flex items-center justify-between gap-1">
        <router-link
          to="/account"
          :aria-current="accountActive ? 'page' : undefined"
          :title="$t('shell.nav.account')"
          class="min-w-0 flex-1 truncate rounded-xl px-3 py-2 text-xs font-semibold border border-transparent"
          :class="accountActive ? 'bg-rose-500/10 text-rose-400 border-rose-500/20' : 'text-slate-200 hover:bg-slate-800/50'"
        >
          <span class="sr-only">{{ $t('shell.nav.account') }}: </span>{{ authStore.user?.email }}
        </router-link>
        <button
          @click="handleLogout"
          class="p-2 text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
          :title="$t('shell.navigation.signOut')" :aria-label="$t('shell.navigation.signOut')"
        >
          <LogOut class="w-4 h-4" />
        </button>
      </div>
    </div>
  </aside>

  <!-- Mobile Bottom Navigation Bar -->
  <nav :aria-label="$t('shell.navigation.mainNavigation')" class="md:hidden fixed bottom-0 left-0 right-0 z-50 [view-transition-name:bottom-bar] bg-slate-900/95 backdrop-blur-md border-t border-slate-800 px-1 safe-area-pb">
    <ul class="flex items-stretch">
      <li v-for="item in leftItems" :key="item.name" class="flex-1">
        <NavBarItem :to="item.path" :icon="item.icon" :label="$t(item.mobileLabelKey ?? item.labelKey)" :active="activeTab === item.name" />
      </li>

      <li v-if="canQuickAdd" class="flex flex-1 items-center justify-center">
        <button
          type="button"
          class="-mt-5 flex h-14 w-14 items-center justify-center rounded-full bg-rose-600 text-white shadow-lg shadow-rose-600/40 ring-4 ring-slate-950 transition-colors hover:bg-rose-500 active:bg-rose-700"
          :aria-label="$t('shell.navigation.quickAdd')"
          @click="openQuickAdd"
        >
          <Plus class="h-7 w-7" aria-hidden="true" />
        </button>
      </li>

      <li v-for="item in rightItems" :key="item.name" class="flex-1">
        <NavBarItem :to="item.path" :icon="item.icon" :label="$t(item.mobileLabelKey ?? item.labelKey)" :active="activeTab === item.name" />
      </li>
    </ul>
  </nav>
</template>
