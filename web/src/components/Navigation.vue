<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useQuickAddStore } from '@/stores/quickAdd'
import { useVehicleStore } from '@/stores/vehicle'
import {
  LayoutDashboard,
  Navigation as NavIcon,
  Users,
  Disc,
  Receipt,
  Wrench,
  Car,
  LogOut,
  Zap,
  Scale,
  ClipboardList,
  UserRound,
  Plus,
} from 'lucide-vue-next'
import { APP_NAME } from '@/brand'
import LanguageSwitcher from '@/components/LanguageSwitcher.vue'
import { APP_VERSION } from '@/version'
import { activePhoneTab, groupNavItems, phoneTabs } from '@/utils/navigation'
import { quickKindForRoute } from '@/utils/quickAdd'
import NavBarItem from '@/components/NavBarItem.vue'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const vehicleStore = useVehicleStore()
const quickAdd = useQuickAddStore()

const allNavItems = [
  { name: 'dashboard', labelKey: 'shell.nav.dashboard', mobileLabelKey: 'shell.nav.today', path: '/', icon: LayoutDashboard },
  { name: 'drives', labelKey: 'shell.nav.drives', path: '/drives', icon: NavIcon },
  { name: 'carpools', labelKey: 'shell.nav.carpools', path: '/carpools', icon: Users },
  { name: 'tires', labelKey: 'shell.nav.tires', path: '/tires', icon: Disc },
  { name: 'odometer', labelKey: 'shell.nav.odometer', path: '/odometer', icon: ClipboardList },
  { name: 'energy', labelKey: 'shell.nav.energy', path: '/energy', icon: Zap },
  { name: 'costs', labelKey: 'shell.nav.expenses', path: '/costs', icon: Receipt },
  { name: 'maintenance', labelKey: 'shell.nav.maintenance', path: '/maintenance', icon: Wrench },
  { name: 'comparison', labelKey: 'shell.nav.comparison', path: '/comparison', icon: Scale },
  { name: 'vehicle', labelKey: 'shell.nav.vehicle', path: '/vehicle', icon: Car },
  { name: 'vehicles', labelKey: 'shell.nav.vehicles', path: '/vehicles', icon: Car },
  { name: 'account', labelKey: 'shell.nav.account', path: '/account', icon: UserRound },
]

// Drives and carpooling apply to any non-combustion vehicle (connected or manually tracked)
const navItems = computed(() =>
  allNavItems.filter((item) => {
    if (item.name === 'drives') return vehicleStore.canCharge
    if (item.name === 'carpools') return vehicleStore.canCharge
    if (item.name === 'energy') return vehicleStore.canCharge || vehicleStore.canRefuel
    return true
  }),
)

const currentRouteName = computed(() => route.name)
const openQuickAdd = () => quickAdd.open(quickKindForRoute(route.name, vehicleStore))
const navSections = computed(() => groupNavItems(navItems.value))

// Phone bar: four tabs around the quick entry button; the other pages are reached from them (see `phoneTabs`).
const tabs = computed(() => phoneTabs(vehicleStore.canCharge))
const primaryItems = computed(() => tabs.value.map((tab) => navItems.value.find((item) => item.name === tab.name)).filter((item) => !!item))
const activeTab = computed(() => activePhoneTab(tabs.value, currentRouteName.value))
const leftItems = computed(() => primaryItems.value.slice(0, 2))
const rightItems = computed(() => primaryItems.value.slice(2))

const canQuickAdd = computed(() => !!vehicleStore.activeVehicle && vehicleStore.canEdit)
function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <!-- Desktop Sidebar -->
  <aside class="hidden md:flex flex-col w-64 bg-slate-900 border-r border-slate-800 p-4 shrink-0">
    <div class="flex items-center gap-3 px-3 py-4 mb-4">
      <div class="p-2 bg-gradient-to-tr from-rose-500 to-warning-500 rounded-xl shadow-lg shadow-rose-500/20">
        <Zap class="w-6 h-6 text-white" />
      </div>
      <div>
        <div class="flex items-center gap-2">
          <p class="text-xl font-bold tracking-tight bg-gradient-to-r from-white to-slate-400 bg-clip-text text-transparent">
            {{ APP_NAME }}
          </p>
        </div>
        <p class="text-xs text-slate-400">{{ $t('shell.navigation.tagline') }}</p>
      </div>
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

    <nav :aria-label="$t('shell.navigation.mainNavigation')" class="flex-1 overflow-y-auto space-y-0.5">
      <div
        v-for="(section, index) in navSections"
        :key="section.headingKey ?? index"
        role="group"
        :aria-labelledby="section.headingKey ? 'nav-section-' + index : undefined"
        class="space-y-0.5"
      >
        <p v-if="section.headingKey" :id="'nav-section-' + index" class="px-3 pb-0.5 pt-3 text-xs font-semibold text-slate-400">
          {{ $t(section.headingKey) }}
        </p>
        <router-link
          v-for="item in section.items"
          :key="item.name"
          :to="item.path"
          :aria-current="currentRouteName === item.name ? 'page' : undefined"
          class="flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium transition-all border border-transparent"
          :class="
            currentRouteName === item.name
              ? 'bg-rose-500/10 text-rose-400 border border-rose-500/20 shadow-sm'
              : 'text-slate-400 hover:text-slate-200 hover:bg-slate-800/50'
          "
        >
          <component :is="item.icon" class="w-5 h-5" />
          {{ $t(item.labelKey) }}
        </router-link>
      </div>
    </nav>

    <div class="pt-4 border-t border-slate-800 mt-auto space-y-2">
      <div class="flex items-center justify-between px-3 py-1">
        <div class="truncate">
          <p class="text-xs font-semibold text-slate-200 truncate">{{ authStore.user?.email }}</p>
          <p class="text-xs text-slate-400">{{ $t('shell.navigation.signedIn') }}</p>
        </div>
        <button
          @click="handleLogout"
          class="p-2 text-slate-400 hover:text-rose-400 hover:bg-rose-500/10 rounded-lg transition-colors"
          :title="$t('shell.navigation.signOut')" :aria-label="$t('shell.navigation.signOut')"
        >
          <LogOut class="w-4 h-4" />
        </button>
      </div>
      <div class="px-3 pt-2 text-xs text-slate-400 flex items-center justify-between border-t border-slate-800/60">
        <span>{{ $t('shell.navigation.version') }}</span>
        <span class="font-mono text-slate-400 font-medium">{{ APP_VERSION }}</span>
      </div>
      <div class="px-3 pt-2">
        <LanguageSwitcher />
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
