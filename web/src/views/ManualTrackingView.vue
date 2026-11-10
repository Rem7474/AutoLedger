<script setup lang="ts">
import PageHeader from '@/components/PageHeader.vue'
import { ClipboardList as PageIcon } from '@/icons'
import { ref } from 'vue'
import { UploadCloud, Download } from 'lucide-vue-next'
import RowMenu from '@/components/RowMenu.vue'
import { useVehicleStore } from '@/stores/vehicle'
import OdometerReadingsPanel from '@/components/manual/OdometerReadingsPanel.vue'
import CSVImportModal from '@/components/CSVImportModal.vue'
import ExportDataModal from '@/components/ExportDataModal.vue'

const showImport = ref(false)
const showExport = ref(false)
const reloadKey = ref(0)
// The panel loads its data when mounted: remounting it shows what the import added
const onImported = () => {
  reloadKey.value++
}

const vehicleStore = useVehicleStore()
</script>

<template>
  <div class="space-y-5">
    <PageHeader :title="$t('shell.nav.odometer')" :icon="PageIcon">
      {{ $t('manual.manualTrackingView.subtitleMileage') }}{{ vehicleStore.activeVehicle ? ` · ${vehicleStore.activeVehicle.name}` : '' }}{{ $t('manual.manualTrackingView.independent') }}
    </PageHeader>

    <template v-if="vehicleStore.activeVehicle">
      <div class="flex justify-end max-sm:hidden">
        <div class="flex items-center gap-2">
          <button
            v-if="vehicleStore.canEdit"
            type="button"
            class="btn btn-secondary tap-text"
            @click="showImport = true"
          >
            <UploadCloud class="w-3.5 h-3.5 text-sky-400" />
            <span>{{ $t('expenses.expensesView.importCsv') }}</span>
          </button>
          <button
            type="button"
            class="btn btn-secondary tap-text"
            @click="showExport = true"
          >
            <Download class="w-3.5 h-3.5 text-sky-400" />
            <span>{{ $t('import.export') }}</span>
          </button>
        </div>
      </div>

      <OdometerReadingsPanel :key="reloadKey" :vehicle="vehicleStore.activeVehicle" :can-edit="vehicleStore.canEdit">
        <!-- Import and export are rare: a menu next to the Add button on a phone, two buttons above on a wide screen -->
        <template #tools>
          <RowMenu phone class="sm:hidden" :label="$t('common.moreActions')" :heading="$t('shell.nav.odometer')">
            <button v-if="vehicleStore.canEdit" type="button" role="menuitem" class="menu-item" @click="showImport = true">
              <UploadCloud class="w-3.5 h-3.5 text-slate-400" />{{ $t('expenses.expensesView.importCsv') }}
            </button>
            <button type="button" role="menuitem" class="menu-item" @click="showExport = true">
              <Download class="w-3.5 h-3.5 text-slate-400" />{{ $t('import.export') }}
            </button>
          </RowMenu>
        </template>
      </OdometerReadingsPanel>
      <ExportDataModal v-model:open="showExport" :vehicle-id="vehicleStore.activeVehicle.id" />
      <CSVImportModal
        v-model:open="showImport"
        :vehicle-id="vehicleStore.activeVehicle.id"
        default-type="ODOMETER"
        @imported="onImported"
      />
    </template>
  </div>
</template>
