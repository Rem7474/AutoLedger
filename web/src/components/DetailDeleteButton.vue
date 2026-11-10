<script setup lang="ts">
// The delete of a record, at the bottom of its detail (a sheet or a panel): a quiet text button in the destructive
// tone, not a coloured one. It renders only when the record can be deleted by a user who can edit. The page deletes
// through `deleteRowsWithUndo` / `deleteWithUndo` and closes the detail, so there is no confirmation: Undo is in the snackbar.
withDefaults(defineProps<{ deletable?: boolean; canEdit?: boolean }>(), { deletable: true, canEdit: true })
const emit = defineEmits<{ delete: [] }>()
</script>

<template>
  <button
    v-if="deletable && canEdit"
    type="button"
    class="tap-text rounded-lg px-2 text-xs font-semibold text-red-300 transition-colors hover:bg-red-500/10 hover:text-red-200"
    @click="emit('delete')"
  >
    {{ $t('common.delete') }}
  </button>
</template>
