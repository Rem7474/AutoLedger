<script setup lang="ts">
import { CheckCircle2, X } from 'lucide-vue-next'
import { useToast } from '@/composables/useToast'

const { toasts, dismissToast } = useToast()
</script>

<template>
  <Teleport to="body">
    <div class="fixed inset-x-0 bottom-[calc(5.75rem+env(safe-area-inset-bottom))] md:bottom-4 z-[9998] flex flex-col items-center gap-2 px-4 pointer-events-none" role="status" aria-live="polite">
      <TransitionGroup
        enter-active-class="transition duration-200 ease-out"
        enter-from-class="opacity-0 translate-y-2"
        leave-active-class="transition duration-150 ease-in"
        leave-to-class="opacity-0"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          class="pointer-events-auto flex w-full max-w-sm items-center gap-2 rounded-xl border px-3 py-2.5 text-sm text-slate-100 shadow-xl shadow-black/60"
          :class="toast.action ? 'border-slate-700 bg-slate-800' : 'border-success-500/30 bg-slate-900'"
        >
          <CheckCircle2 v-if="!toast.action" class="mt-0.5 h-4 w-4 shrink-0 self-start text-success-400" aria-hidden="true" />
          <span class="min-w-0 flex-1 break-words">{{ toast.message }}</span>
          <button
            v-if="toast.action"
            type="button"
            class="tap -my-1 shrink-0 rounded-lg px-3 py-2 text-sm font-semibold text-rose-300 hover:bg-slate-700"
            @click="toast.action.run(); dismissToast(toast.id)"
          >
            {{ toast.action.label }}
          </button>
          <button type="button" class="tap -m-1 rounded-lg p-1 text-slate-400 hover:text-white" :aria-label="$t('common.close')" @click="dismissToast(toast.id)">
            <X class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
