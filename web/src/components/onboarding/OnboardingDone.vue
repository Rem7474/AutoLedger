<script setup lang="ts">
import { ref } from 'vue'
import { CheckCircle2, Copy } from 'lucide-vue-next'

// Last wizard screen: the webhook credentials when one was requested, and where to go next.
const props = defineProps<{
  webhook: { token: string; snippet: string } | null
  webhookFailed: boolean
}>()
const emit = defineEmits<{ finish: []; addEntry: [] }>()

const copiedField = ref<'token' | 'snippet' | ''>('')

async function copy(field: 'token' | 'snippet') {
  if (!props.webhook) return
  try {
    await navigator.clipboard.writeText(props.webhook[field])
    copiedField.value = field
    setTimeout(() => (copiedField.value = ''), 2000)
  } catch {
    copiedField.value = ''
  }
}
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
    <div v-if="webhook" class="text-left bg-slate-800/60 border border-slate-700 rounded-xl p-4 mb-6 space-y-3">
      <h3 class="text-sm font-semibold text-white">{{ $t('onboarding.webhookReadyTitle') }}</h3>
      <p class="text-xs text-slate-400">{{ $t('onboarding.webhookReadyHint') }}</p>
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-xs font-semibold text-slate-300 uppercase tracking-wider">{{ $t('onboarding.webhookToken') }}</span>
          <button type="button" @click="copy('token')" class="text-xs text-rose-400 hover:text-rose-300 flex items-center gap-1">
            <Copy class="w-3.5 h-3.5" />{{ copiedField === 'token' ? $t('onboarding.copied') : $t('onboarding.copy') }}
          </button>
        </div>
        <code class="block bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-success-300 break-all">{{ webhook.token }}</code>
      </div>
      <div>
        <div class="flex items-center justify-between mb-1">
          <span class="text-xs font-semibold text-slate-300 uppercase tracking-wider">{{ $t('onboarding.webhookExample') }}</span>
          <button type="button" @click="copy('snippet')" class="text-xs text-rose-400 hover:text-rose-300 flex items-center gap-1">
            <Copy class="w-3.5 h-3.5" />{{ copiedField === 'snippet' ? $t('onboarding.copied') : $t('onboarding.copy') }}
          </button>
        </div>
        <pre class="bg-slate-950 border border-slate-800 rounded-lg px-3 py-2 text-xs text-slate-300 overflow-x-auto whitespace-pre">{{ webhook.snippet }}</pre>
      </div>
    </div>
    <p v-else-if="webhookFailed" class="text-xs text-warning-300 bg-warning-500/10 border border-warning-500/20 rounded-xl p-3 mb-6">{{ $t('onboarding.webhookTokenFailed') }}</p>

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
