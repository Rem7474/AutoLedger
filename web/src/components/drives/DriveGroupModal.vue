<script setup lang="ts">
import NumberInput from '@/components/NumberInput.vue'
import { t } from '@/i18n'
import { computed, ref, watch } from 'vue'
import { api } from '@/services/api'
import { useConfirm } from '@/composables/useConfirm'
import { useVehicleStore } from '@/stores/vehicle'
import { Layers } from 'lucide-vue-next'
import { currencySymbol } from '@/currency'
import ModalShell from '@/components/ModalShell.vue'
import { useSubmit } from '@/composables/useSubmit'

// Merges the selected drives into a trip group, optionally with one expense (toll, parking, ferry) for the whole trip.
const props = defineProps<{ vehicleId: string; selectedDriveIds: string[]; selectedList: any[] }>()
const emit = defineEmits<{ saved: [] }>()
const open = defineModel<boolean>('open', { required: true })
const { showAlert } = useConfirm()
const vehicleStore = useVehicleStore()

const groupName = ref('')
const tollAmount = ref<number | ''>('')
const expenseType = ref('TOLL')

const submitted = ref(false)
const nameError = computed(() => (!groupName.value.trim() ? t('drives.driveGroupModal.nameRequired') : ''))
watch(open, (isOpen) => {
  if (isOpen) submitted.value = false
})

const { pending: submitting, run: runOnce } = useSubmit()

async function handleCreateGroupAndExpenseAction() {
  if (!props.vehicleId || !props.selectedDriveIds.length) return
  submitted.value = true
  if (nameError.value) {
    document.getElementById('drive-group-name')?.focus()
    return
  }

  try {
    if (tollAmount.value && Number(tollAmount.value) > 0) {
      // Group and expense are created atomically by the backend; the expense is dated at the trip start
      const firstStart = props.selectedList.length ? new Date(props.selectedList[0].start_time).getTime() : undefined
      await api.createDriveExpense(props.vehicleId, {
        drive_ids: props.selectedDriveIds,
        type: expenseType.value,
        amount: Number(tollAmount.value),
        currency: vehicleStore.currency,
        date: new Date(firstStart || Date.now()).toISOString(),
        notes: groupName.value,
      })
    } else {
      await api.createTripGroup(props.vehicleId, {
        name: groupName.value,
        drive_ids: props.selectedDriveIds,
      })
    }

    showAlert(t('drives.driveGroupModal.created'), t('common.success'), 'success')
    open.value = false
    groupName.value = ''
    tollAmount.value = ''
    emit('saved')
  } catch (err: any) {
    showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}
const handleCreateGroupAndExpense = () => runOnce(handleCreateGroupAndExpenseAction)
</script>

<template>
  <ModalShell v-model:open="open" size="sm">
    <template #title>
      <div class="flex items-center gap-2">
        <Layers class="w-5 h-5 text-rose-400" aria-hidden="true" />
        <h3 class="text-base font-bold text-white">{{ $t('drives.driveGroupModal.createATripMerge') }}</h3>
        <span class="px-2 py-0.5 bg-rose-500/10 text-rose-300 text-xs font-semibold rounded-lg border border-rose-500/20">
          {{ $t('drives.driveGroupModal.drives', { length: selectedDriveIds.length }) }}
        </span>
      </div>
    </template>

    <div>
      <label for="drive-group-name" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('drives.driveGroupModal.tripGroupName') }}</label>
      <input id="drive-group-name"
        v-model="groupName"
        type="text"
        :placeholder="$t('drives.driveGroupModal.eGBrittanyHolidayOutbound')"
        :aria-invalid="submitted && !!nameError"
        :aria-describedby="submitted && nameError ? 'drive-group-name-error' : undefined"
        class="field"
      />
      <p v-if="submitted && nameError" id="drive-group-name-error" class="text-xs text-danger-400 mt-1">{{ nameError }}</p>
    </div>

    <div class="grid grid-cols-2 gap-3">
      <div>
        <label for="drive-expense-type" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('drives.driveGroupModal.costType') }}</label>
        <select id="drive-expense-type"
          v-model="expenseType"
          class="field"
        >
          <option value="TOLL">{{ $t('drives.driveGroupModal.toll') }}</option>
          <option value="PARKING">{{ $t('drives.driveGroupModal.parking') }}</option>
          <option value="FERRY">{{ $t('drives.driveGroupModal.ferry') }}</option>
        </select>
      </div>
      <div>
        <label for="drive-toll-amount" class="block text-xs font-semibold text-slate-300 mb-1">{{ $t('drives.driveGroupModal.amount', { cur: currencySymbol(vehicleStore.currency) }) }}</label>
        <NumberInput text id="drive-toll-amount"
          v-model="tollAmount"
          placeholder="0.00"
          class="field"
        />
      </div>
    </div>

    <template #footer>
      <button type="button" @click="open = false" class="btn btn-lg btn-secondary">
        {{ $t('common.cancel') }}
      </button>
      <button :disabled="submitting" type="button" @click="handleCreateGroupAndExpense" class="btn btn-lg btn-primary">
        {{ $t('drives.driveGroupModal.saveTheGroup') }}
      </button>
    </template>
  </ModalShell>
</template>
