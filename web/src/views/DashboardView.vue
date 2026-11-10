<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import PageHeader from '@/components/PageHeader.vue'
import FleetOverview from '@/components/fleet/FleetOverview.vue'
import VehicleToday from '@/components/dashboard/VehicleToday.vue'
import { LayoutDashboard as PageIcon, Users } from 'lucide-vue-next'
import { useVehicleStore } from '@/stores/vehicle'
import { distanceUnit } from '@/units'
import { scopeFromQuery } from '@/utils/vehicleScope'

// The home page covers the active vehicle, or every vehicle of the account side by side (`?scope=all`, chosen in the
// vehicle selector). The scope is in the URL: it is shared by a link and survives a reload.
const route = useRoute()
const vehicleStore = useVehicleStore()
const scope = computed(() => scopeFromQuery(route.query.scope, vehicleStore.vehicles.length))
</script>

<template>
  <div class="space-y-6 max-md:pb-16">
    <PageHeader v-if="scope === 'all'" :title="$t('fleet.today.allVehicles')" :icon="Users">{{ $t('fleet.today.subtitle') }}</PageHeader>
    <PageHeader v-else :title="$t('shell.nav.dashboard')" :icon="PageIcon">
      {{ $t('dashboard.dashboardView.subtitle', { unit: distanceUnit(), name: vehicleStore.activeVehicle?.name || $t('dashboard.dashboardView.yourVehicle') }) }}
    </PageHeader>

    <FleetOverview v-if="scope === 'all'" />
    <VehicleToday v-else />
  </div>
</template>
