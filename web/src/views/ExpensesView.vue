<script setup lang="ts">
import LoadError from '@/components/LoadError.vue'
import TabBar from '@/components/TabBar.vue'
import PageHeader from '@/components/PageHeader.vue'
import { t } from '@/i18n'
import { ref, computed, onMounted, watch } from 'vue'
import { useVehicleStore } from '@/stores/vehicle'
import { useConfirm } from '@/composables/useConfirm'
import { useExpensesTabs, type ExpensesTab } from '@/composables/useExpensesTabs'
import { useExpenseDeletions } from '@/composables/useExpenseDeletions'
import { useDocumentPreview } from '@/composables/useDocumentPreview'
import { useReminders } from '@/composables/useReminders'
import { api, type ExpenseDocumentHeader, type MaintenanceReminder } from '@/services/api'
import DocumentPreviewModal from '@/components/expenses/DocumentPreviewModal.vue'
import RemindersPanel from '@/components/expenses/RemindersPanel.vue'
import ChargesPanel from '@/components/expenses/ChargesPanel.vue'
import FuelLogsPanel from '@/components/manual/FuelLogsPanel.vue'
import EnergyEfficiencyPanel from '@/components/dashboard/EnergyEfficiencyPanel.vue'
import EstimatedEnergyPanel from '@/components/manual/EstimatedEnergyPanel.vue'
import ExpensesHeaderActions from '@/components/expenses/ExpensesHeaderActions.vue'
import DocumentsPanel from '@/components/expenses/DocumentsPanel.vue'
import ChargeModal from '@/components/expenses/ChargeModal.vue'
import UploadDocumentModal from '@/components/expenses/UploadDocumentModal.vue'
import ReminderModal from '@/components/expenses/ReminderModal.vue'
import CompleteReminderModal from '@/components/expenses/CompleteReminderModal.vue'
import WebhookModal from '@/components/expenses/WebhookModal.vue'
import CSVImportModal from '@/components/CSVImportModal.vue'
import QualifyChargesModal from '@/components/expenses/QualifyChargesModal.vue'
import { ChevronRight, Zap, Eye } from '@/icons'
import { useRefreshHandler } from '@/composables/usePullToRefresh'

// The page owns the lists, the active tab and which modal is open; each modal owns its form and its API
// call and reports back with "saved".
const vehicleStore = useVehicleStore()
const { showAlert } = useConfirm()

const vehicleId = computed(() => vehicleStore.activeVehicle?.id ?? '')
const currentOdometer = computed(() => vehicleStore.activeVehicle?.current_odometer || 0)
const { previewDoc, loadingDocId, closeDocPreview, viewOrDownloadDocument } = useDocumentPreview(() => vehicleStore.activeVehicle?.id)

const { activeTab, pageTitle, pageSubtitle, pageIcon, tabs } = useExpensesTabs()

const charges = ref<any[]>([])
const chargesWithoutCost = ref(0)
const chargesTotal = ref(0)
const chargesPage = ref(1)
const loadingMoreCharges = ref(false)
// This page lists only the charges to complete; the whole list is read on the Costs page
const missingCostOnly = true
const loading = ref(false)
const loadError = ref<string | null>(null)

// Maintenance reminders & webhook
const {
  reminders,
  loadingReminders,
  showReminderModal,
  editingReminder,
  reminderPreset,
  showCompleteReminderModal,
  completingReminder,
  showWebhookModal,
  vehicleWebhook,
  overdueReminders,
  dueSoonReminders,
  okReminders,
  unscheduledReminders,
  loadReminders,
  openAddReminderModal,
  openEditReminderModal,
  handleDeleteReminder,
  openCompleteReminder,
  onReminderCompleted,
  openWebhookModal,
} = useReminders({
  isRemindersTabActive: () => activeTab.value === 'REMINDERS',
  onLoadError: (message) => {
    loadError.value = message
  },
})

// Documents & Invoices
const documents = ref<ExpenseDocumentHeader[]>([])

// Modals, with the item being edited (null when adding)
const showChargeModal = ref(false)
const editingCharge = ref<any | null>(null)
const showUploadDocModal = ref(false)

async function ensureDocumentsLoaded() {
  if (!vehicleStore.activeVehicle) return
  try {
    documents.value = await api.getDocuments(vehicleStore.activeVehicle.id)
  } catch (err) {
    console.error('Failed to load documents', err)
  }
}

function onDocumentAdded(doc: ExpenseDocumentHeader) {
  documents.value.unshift(doc)
}

const pendingChargesCount = ref(0)
const showQualifyModal = ref(false)

async function checkPendingCharges() {
  try {
    const res = await api.getPendingCharges()
    pendingChargesCount.value = res.pending_charges?.length || 0
  } catch {
    pendingChargesCount.value = 0
  }
}

const refreshData = () => loadData(true)
useRefreshHandler(refreshData)

async function onChargesAssigned() {
  await refreshData()
}

// The reminder edited in its panel is deleted from there: the panel closes and Undo is in the snackbar
function onDeleteEditedReminder(r: MaintenanceReminder) {
  showReminderModal.value = false
  handleDeleteReminder(r)
}

// The fill-ups panel loads its own data when mounted: remounting it shows what an import added
const fuelReloadKey = ref(0)
function onImported() {
  fuelReloadKey.value++
  void loadData()
}

// A new vehicle, tab or filter shows the skeleton. After an action on a row (save, delete, "Free") the list stays on
// screen with the pages already loaded: a skeleton there collapses the page and throws a phone back to the top.
async function loadData(keepList = false) {
  if (!vehicleStore.activeVehicle) return
  if (!keepList) loading.value = true
  loadError.value = null
  try {
    if (activeTab.value === 'REMINDERS') {
      await loadReminders()
    } else if (activeTab.value === 'CHARGES' && vehicleStore.capabilities.charges) {
      const pages = keepList ? chargesPage.value : 1
      const loaded: any[] = []
      let res: any
      for (let page = 1; page <= pages; page++) {
        res = await api.getCharges(vehicleStore.activeVehicle.id, { page, missingCost: missingCostOnly })
        loaded.push(...res.charges)
      }
      chargesPage.value = pages
      charges.value = loaded
      chargesTotal.value = res.total || 0
      chargesWithoutCost.value = res.charges_without_cost || 0
      void checkPendingCharges()
    } else if (activeTab.value === 'DOCUMENTS') {
      documents.value = await api.getDocuments(vehicleStore.activeVehicle.id)
    }
    void ensureDocumentsLoaded()
    if (activeTab.value !== 'REMINDERS') {
      void loadReminders()
    }
  } catch (err) {
    console.error('Failed to load expenses', err)
    loadError.value = (err as Error)?.message ?? ''
  } finally {
    loading.value = false
  }
}

const { handleDeleteCharges, handleDeleteCharge, handleDeleteDocument } = useExpenseDeletions(documents, refreshData, { charges })

watch(
  () => [vehicleStore.activeVehicle?.id, activeTab.value, vehicleStore.lastSyncTimestamp],
  () => {
    void loadData()
  }
)

onMounted(() => {
  void loadData()
  void loadReminders()
})

// Charges: manual entry, and the charges whose cost is missing
// Next page of charges, so that older charges can also be completed or corrected
async function loadMoreCharges() {
  if (!vehicleStore.activeVehicle) return
  loadingMoreCharges.value = true
  try {
    const res = await api.getCharges(vehicleStore.activeVehicle.id, { page: chargesPage.value + 1, missingCost: missingCostOnly })
    chargesPage.value += 1
    charges.value = [...charges.value, ...res.charges]
    chargesTotal.value = res.total || 0
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  } finally {
    loadingMoreCharges.value = false
  }
}

async function reviewChargeCost(c: any, reviewed: boolean) {
  try {
    await api.setChargeCostReview(vehicleId.value, c.id, reviewed)
    await refreshData()
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err.message }), t('shell.confirm.error'), 'danger')
  }
}

const showCSVImportModal = ref(false)

function openCSVImportModal() {
  showCSVImportModal.value = true
}

function openAddChargeModal() {
  editingCharge.value = null
  showChargeModal.value = true
  void ensureDocumentsLoaded()
}

// A charge opens from the record the server holds (as the cost journal reads it), not from the row of the list
async function openEditChargeModal(c: any) {
  if (!vehicleStore.activeVehicle) return
  try {
    const [record] = await Promise.all([api.getCharge(vehicleStore.activeVehicle.id, c.id), ensureDocumentsLoaded()])
    editingCharge.value = record
    showChargeModal.value = true
  } catch (err: any) {
    void showAlert(t('common.errorWithMessage', { message: err?.message ?? '' }), t('shell.confirm.error'), 'danger')
  }
}

// Documents
function openUploadDocumentModal() {
  showUploadDocModal.value = true
}

</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <PageHeader :title="pageTitle" :icon="pageIcon">
      {{ pageSubtitle }}
      <template #actions>
        <ExpensesHeaderActions
          :active-tab="activeTab"
          @open-webhook="openWebhookModal"
          @add-reminder="openAddReminderModal()"
          @import-csv="openCSVImportModal"
          @add-charge="openAddChargeModal"
          @upload-document="openUploadDocumentModal"
        />
      </template>
    </PageHeader>

    <!-- Viewer mode banner -->
    <div
      v-if="!vehicleStore.canEdit"
      class="p-3.5 card flex items-center gap-3 text-xs text-slate-400"
    >
      <Eye class="w-4 h-4 text-slate-400 shrink-0" />
      <span>{{ $t('expenses.expensesView.youAreViewingThisVehicle') }} <strong>{{ $t('expenses.expensesView.readOnly') }}</strong>{{ $t('expenses.expensesView.modeAdditionsAndChangesAre') }}</span>
    </div>

    <TabBar v-if="tabs.length > 1" :model-value="activeTab" :tabs="tabs" :label="pageTitle" id-prefix="expenses-tab" @update:model-value="activeTab = $event as ExpensesTab" />

    <LoadError v-if="loadError !== null" :message="loadError" @retry="loadData()" />

    <!-- Content -->
    <RemindersPanel
      v-if="activeTab === 'REMINDERS'"
      :reminders="reminders"
      :overdue-reminders="overdueReminders"
      :due-soon-reminders="dueSoonReminders"
      :ok-reminders="okReminders"
      :unscheduled-reminders="unscheduledReminders"
      :loading-reminders="loadingReminders"
      :vehicle-webhook="vehicleWebhook"
      @open-webhook="openWebhookModal"
      @add="openAddReminderModal"
      @edit="openEditReminderModal"
      @complete="openCompleteReminder"
      @reload="loadReminders"
    />

    <div
      v-if="activeTab === 'CHARGES' && vehicleStore.capabilities.charges && pendingChargesCount > 0"
      class="p-4 bg-warning-500/10 border border-warning-500/30 rounded-2xl flex flex-col sm:flex-row sm:items-center justify-between gap-3 text-xs"
    >
      <div class="flex items-center gap-2.5 text-warning-300">
        <Zap class="w-4 h-4 text-warning-400 shrink-0" />
        <span>
          <strong>{{ pendingChargesCount }} {{ $t('pendingCharges.bannerCount', { count: pendingChargesCount }) }}</strong>
          — {{ $t('pendingCharges.bannerDescription') }}
        </span>
      </div>
      <button
        @click="showQualifyModal = true"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 bg-warning-600 hover:bg-warning-500 text-white font-semibold rounded-xl transition-colors shrink-0"
      >
        {{ $t('pendingCharges.qualifyButton') }}
      </button>
    </div>

    <EnergyEfficiencyPanel
      v-if="activeTab === 'EFFICIENCY' && vehicleStore.activeVehicle"
      :vehicle-id="vehicleStore.activeVehicle.id"
      :grafana-url="vehicleStore.activeVehicle.teslamate_grafana_url"
      :sync-key="vehicleStore.lastSyncTimestamp"
      show-empty
    />

    <EstimatedEnergyPanel
      v-if="activeTab === 'ESTIMATE' && vehicleStore.activeVehicle"
      :vehicle="vehicleStore.activeVehicle"
      :can-edit="vehicleStore.canEdit"
    />

    <FuelLogsPanel
      v-if="activeTab === 'FUEL' && vehicleStore.capabilities.refuels && vehicleStore.activeVehicle"
      :key="fuelReloadKey"
      :vehicle-id="vehicleStore.activeVehicle.id"
      :can-edit="vehicleStore.canEdit"
    />

    <router-link
      v-if="activeTab === 'CHARGES' && vehicleStore.capabilities.charges"
      to="/costs?theme=energy"
      class="card flex items-center justify-between gap-3 px-4 py-3"
      data-testid="charges-list-link"
    >
      <span class="min-w-0">
        <span class="block text-sm font-semibold text-white">{{ $t('expenses.chargesTools.openList') }}</span>
        <span class="block text-xs text-slate-400">{{ $t('expenses.chargesTools.listHint') }}</span>
      </span>
      <ChevronRight class="size-5 shrink-0 text-slate-400" aria-hidden="true" />
    </router-link>

    <ChargesPanel
      v-if="activeTab === 'CHARGES' && vehicleStore.capabilities.charges"
      to-complete-only
      :charges="charges"
      :charges-total="chargesTotal"
      :charges-without-cost="chargesWithoutCost"
      :loading="loading"
      :loading-more-charges="loadingMoreCharges"
      :missing-cost-only="missingCostOnly"
      @import-csv="openCSVImportModal"
      @load-more="loadMoreCharges"
      @edit="openEditChargeModal"
      @delete="handleDeleteCharge"
      @delete-many="handleDeleteCharges"
      @review-cost="reviewChargeCost"
      @view-document="viewOrDownloadDocument"
    />

    <DocumentsPanel
      v-if="activeTab === 'DOCUMENTS'"
      :documents="documents"
      :loading="loading"
      :loading-doc-id="loadingDocId"
      @upload="openUploadDocumentModal"
      @delete="handleDeleteDocument"
      @view-document="viewOrDownloadDocument"
    />

    <ChargeModal
      v-model:open="showChargeModal"
      :vehicle-id="vehicleId"
      :editing="editingCharge"
      :documents="documents"
      :current-odometer="currentOdometer"
      @saved="refreshData"
      @delete="handleDeleteCharge"
      @document-added="onDocumentAdded"
      @view-document="viewOrDownloadDocument"
    />

    <UploadDocumentModal
      v-model:open="showUploadDocModal"
      :vehicle-id="vehicleId"
      @document-added="onDocumentAdded"
    />

    <ReminderModal
      v-model:open="showReminderModal"
      :vehicle-id="vehicleId"
      :editing="editingReminder"
      :preset="reminderPreset"
      :current-odometer="currentOdometer"
      @saved="loadReminders"
      @delete="onDeleteEditedReminder"
    />

    <CompleteReminderModal
      v-model:open="showCompleteReminderModal"
      :vehicle-id="vehicleId"
      :reminder="completingReminder"
      :current-odometer="currentOdometer"
      @saved="onReminderCompleted"
    />

    <WebhookModal
      v-model:open="showWebhookModal"
      :vehicle-id="vehicleId"
      v-model:webhook="vehicleWebhook"
    />

    <!-- Modal: Document In-App Preview (Modular Component) -->
    <DocumentPreviewModal :preview-doc="previewDoc" @close="closeDocPreview" />

    <CSVImportModal
      v-model:open="showCSVImportModal"
      :vehicle-id="vehicleId"
      :default-type="activeTab === 'FUEL' ? 'FUEL' : 'CHARGES'"
      @imported="onImported"
    />

    <QualifyChargesModal
      v-model:open="showQualifyModal"
      @assigned="onChargesAssigned"
    />
  </div>
</template>
