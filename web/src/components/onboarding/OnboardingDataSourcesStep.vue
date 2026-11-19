<script setup lang="ts">
import { canLinkTeslaMate } from '@/utils/vehicles'
import type { Powertrain } from '@/utils/onboarding'
import { ArrowRight, ExternalLink, KeyRound, Link2, Webhook } from '@/icons'

// Last step of the wizard: information only. AutoLedger works with what is typed by hand; the sources below are
// optional and are connected later, where the user needs them. Nothing is configured or created here.
defineProps<{ powertrain: Powertrain; saving: boolean }>()
const emit = defineEmits<{ back: []; finalize: [] }>()

const HOME_ASSISTANT_EXTENSION_URL = 'https://github.com/Rem7474/autoledger-homeassistant'
</script>

<template>
  <div>
    <div class="mb-6">
      <h2 class="text-lg font-semibold text-white flex items-center gap-2">
        <KeyRound class="w-5 h-5 text-rose-400" />
        {{ $t('onboarding.onboardingView.3TeslamateSynchronizationOptional') }}
      </h2>
      <p class="text-xs text-slate-400 mt-1">{{ $t('onboarding.sourcesIntro') }}</p>
    </div>

    <div class="space-y-4">
      <div class="flex items-start gap-3 p-4 bg-slate-800/60 border border-slate-700 rounded-xl">
        <Webhook class="w-5 h-5 shrink-0 mt-0.5 text-rose-400" aria-hidden="true" />
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-white">{{ $t('onboarding.homeAssistantTitle') }}</h3>
          <p class="text-xs text-slate-400 mt-0.5">{{ $t('onboarding.homeAssistantText') }}</p>
          <a
            :href="HOME_ASSISTANT_EXTENSION_URL"
            target="_blank"
            rel="noopener noreferrer"
            class="mt-2 inline-flex items-center gap-1.5 text-xs font-medium text-rose-300 hover:text-rose-200"
          >
            <ExternalLink class="w-3 h-3" aria-hidden="true" />{{ $t('onboarding.homeAssistantLink') }}
          </a>
        </div>
      </div>

      <div v-if="canLinkTeslaMate(powertrain)" class="flex items-start gap-3 p-4 bg-slate-800/60 border border-slate-700 rounded-xl">
        <Link2 class="w-5 h-5 shrink-0 mt-0.5 text-rose-400" aria-hidden="true" />
        <div class="min-w-0">
          <h3 class="text-sm font-medium text-white">{{ $t('onboarding.teslamateTitle') }}</h3>
          <p class="text-xs text-slate-400 mt-0.5">{{ $t('onboarding.teslamateText', { button: $t('vehicles.vehicleCard.dataSources') }) }}</p>
        </div>
      </div>

      <div class="flex gap-3 pt-2">
        <button
          type="button"
          @click="emit('back')"
          class="btn btn-lg btn-secondary"
        >
          {{ $t('onboarding.onboardingView.back') }}
        </button>
        <button
          type="button"
          :disabled="saving"
          @click="emit('finalize')"
          class="flex-1 py-3 px-4 bg-gradient-to-r from-rose-600 to-rose-500 hover:from-rose-500 hover:to-rose-400 text-white font-semibold rounded-xl shadow-lg shadow-rose-600/25 flex items-center justify-center gap-2 transition-all disabled:opacity-50"
        >
          <span>{{ saving ? $t('onboarding.saving') : $t('onboarding.finalize') }}</span>
          <ArrowRight class="w-4 h-4" />
        </button>
      </div>
    </div>
  </div>
</template>
