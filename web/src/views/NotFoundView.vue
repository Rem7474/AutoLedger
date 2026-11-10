<script setup lang="ts">
import { SearchX } from '@/icons'
import EmptyState from '@/components/EmptyState.vue'

// An address the app does not know: say so, and offer the way home, instead of silently landing on the dashboard.
</script>

<template>
  <div class="mx-auto max-w-md pt-8">
    <EmptyState :icon="SearchX" :title="$t('shell.notFound.title')">
      {{ $t('shell.notFound.message') }}
      <template #actions>
        <router-link to="/" class="btn btn-lg btn-primary tap-text">{{ $t('shell.nav.home') }}</router-link>
      </template>
    </EmptyState>
  </div>
</template>
