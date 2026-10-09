<script setup lang="ts">
import { FileText, Image as ImageIcon, Paperclip, Download, ExternalLink } from 'lucide-vue-next'
import ModalShell from '@/components/ModalShell.vue'

export interface DocumentPreviewState {
  id: string
  filename: string
  url: string
  isPdf: boolean
  isImage: boolean
}

const props = defineProps<{
  previewDoc: DocumentPreviewState | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

function downloadFile() {
  if (!props.previewDoc) return
  const a = document.createElement('a')
  a.href = props.previewDoc.url
  a.download = props.previewDoc.filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
}

function openInNewTab() {
  if (!props.previewDoc) return
  window.open(props.previewDoc.url, '_blank')
}
</script>

<template>
  <ModalShell
    v-if="previewDoc"
    :open="true"
    nested
    body-class="p-0! bg-slate-950/70 overflow-hidden! flex items-center justify-center min-h-0 relative"
    class="max-w-5xl! h-[92vh] sm:h-[90vh]"
    @update:open="emit('close')"
  >
    <template #title>
      <div class="flex items-center gap-2.5 min-w-0">
        <div class="w-8 h-8 rounded-xl bg-sky-500/10 border border-sky-500/20 flex items-center justify-center shrink-0">
          <FileText v-if="previewDoc.isPdf" class="w-4 h-4 text-sky-400" />
          <ImageIcon v-else-if="previewDoc.isImage" class="w-4 h-4 text-success-400" />
          <Paperclip v-else class="w-4 h-4 text-slate-400" />
        </div>
        <div class="min-w-0">
          <h3 class="text-sm font-bold text-white truncate" :title="previewDoc.filename">
            {{ previewDoc.filename }}
          </h3>
          <p class="text-xs text-slate-400 flex items-center gap-1.5">
            <span v-if="previewDoc.isPdf" class="text-sky-400 font-semibold">{{ $t('expenses.documentPreviewModal.pdfDocument') }}</span>
            <span v-else-if="previewDoc.isImage" class="text-success-400 font-semibold">{{ $t('expenses.documentPreviewModal.image') }}</span>
            <span v-else class="text-slate-400 font-semibold">{{ $t('expenses.documentPreviewModal.file') }}</span>
          </p>
        </div>
      </div>
    </template>

    <template #actions>
      <!-- Download Button -->
      <button
        type="button"
        @click="downloadFile"
        class="btn btn-secondary"
        :title="$t('expenses.documentPreviewModal.downloadTheFile')"
      >
        <Download class="w-3.5 h-3.5 text-sky-400" />
        <span class="hidden sm:inline">{{ $t('expenses.documentPreviewModal.download') }}</span>
      </button>

      <!-- Open in New Tab Button -->
      <button
        type="button"
        @click="openInNewTab"
        class="btn btn-secondary"
        :title="$t('expenses.documentPreviewModal.openInANewTab')"
      >
        <ExternalLink class="w-3.5 h-3.5 text-slate-400" />
        <span class="hidden md:inline">{{ $t('expenses.documentPreviewModal.newTab') }}</span>
      </button>
    </template>

    <!-- PDF preview -->
    <iframe
      v-if="previewDoc.isPdf"
      :src="previewDoc.url"
      class="w-full h-full border-0 bg-white"
      :title="previewDoc.filename"
    />

    <!-- Image preview -->
    <div
      v-else-if="previewDoc.isImage"
      class="w-full h-full p-4 flex items-center justify-center overflow-auto"
    >
      <img
        :src="previewDoc.url"
        :alt="previewDoc.filename"
        class="max-w-full max-h-full object-contain rounded-lg shadow-lg"
      />
    </div>

    <!-- Unsupported preview fallback -->
    <div v-else class="p-8 text-center space-y-3">
      <FileText class="w-12 h-12 text-slate-400 mx-auto" />
      <p class="text-sm text-slate-300">{{ $t('expenses.documentPreviewModal.thisFileFormatCannotBe') }}</p>
      <button
        type="button"
        @click="downloadFile"
        class="btn btn-lg btn-primary"
      >
        <Download class="w-4 h-4" />
        {{ $t('expenses.documentPreviewModal.downloadToView') }}
      </button>
    </div>
  </ModalShell>
</template>
