<script setup lang="ts">
import { Car } from 'lucide-vue-next'
import EmptyState from '@/components/EmptyState.vue'

// What every page about a vehicle shows when the account has none.
</script>

<template>
  <EmptyState :icon="Car" :title="$t('shell.noVehicle.title')">
    {{ $t('shell.noVehicle.hint') }}
    <template #actions>
      <router-link to="/vehicles" class="btn btn-lg btn-primary">{{ $t('shell.noVehicle.action') }}</router-link>
    </template>
  </EmptyState>
</template>
