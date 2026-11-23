<script setup lang="ts">
import { Trash2 } from '@/icons'

// The delete of a row that opens no detail (a place, a rate, a template, a profile): a trash icon in the critical tone,
// with its tooltip and accessible name. It renders only when the record can be deleted by a user who can edit. The
// page deletes through `deleteRowsWithUndo` / `deleteWithUndo`, so there is no confirmation: Undo is in the snackbar.
// A row that opens a detail deletes from the bottom of that detail (`DetailDeleteButton`) instead.
withDefaults(defineProps<{ label: string; deletable?: boolean; canEdit?: boolean }>(), { deletable: true, canEdit: true })
const emit = defineEmits<{ delete: [] }>()
</script>

<template>
  <button
    v-if="deletable && canEdit"
    type="button"
    class="tap shrink-0 rounded-lg p-2 text-critical-400 transition-colors hover:bg-critical-500/10 hover:text-critical-300"
    :title="label"
    :aria-label="label"
    @click="emit('delete')"
  >
    <Trash2 class="h-4 w-4" aria-hidden="true" />
  </button>
</template>
