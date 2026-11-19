<script setup lang="ts">
import { CheckCircle2 } from '@/icons'

// Last wizard screen: where to go next.
const emit = defineEmits<{ finish: []; addEntry: [] }>()
</script>

<template>
  <div class="text-center py-6">
    <div class="inline-flex p-4 bg-success-500/10 border border-success-500/20 text-success-400 rounded-full mb-4">
      <CheckCircle2 class="w-12 h-12" />
    </div>
    <h2 class="text-2xl font-bold text-white mb-2">{{ $t('onboarding.onboardingView.congratulations') }}</h2>
    <p class="text-slate-400 text-sm max-w-sm mx-auto mb-6">
      {{ $t('onboarding.onboardingView.yourAdministratorAccountAndYour') }}
    </p>
    <button
      @click="emit('finish')"
      class="w-full py-3.5 px-6 bg-gradient-to-r from-rose-600 to-rose-500 hover:from-rose-500 hover:to-rose-400 text-white font-semibold rounded-xl shadow-lg shadow-rose-600/25 transition-all"
    >
      {{ $t('onboarding.onboardingView.goToMyDashboard') }}
    </button>
    <button
      type="button"
      @click="emit('addEntry')"
      class="btn btn-lg btn-secondary mt-3 w-full"
    >
      {{ $t('onboarding.onboardingView.addFirstEntry') }}
    </button>
  </div>
</template>
