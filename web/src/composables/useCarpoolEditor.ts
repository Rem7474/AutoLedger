import { ref } from 'vue'
import { api } from '@/services/api'
import { useVehicleStore } from '@/stores/vehicle'
import { recentPassengerNames } from '@/utils/carpool'

export type CarpoolCreateOptions = { driveIds?: string[]; tripGroupId?: string }

// State for opening the carpool editor (`CarpoolTripModal`) from a page other than the carpool list, so the page keeps its
// filters and selection. The editor offers the names used on the vehicle's latest carpools, loaded when it opens.
export function useCarpoolEditor() {
  const vehicleStore = useVehicleStore()
  const showCarpoolEditor = ref(false)
  const carpoolCreateOptions = ref<CarpoolCreateOptions>({})
  const carpoolOpenToken = ref(0)
  const recentCarpoolNames = ref<string[]>([])

  async function openCarpoolEditor(options: CarpoolCreateOptions) {
    carpoolCreateOptions.value = options
    carpoolOpenToken.value++
    showCarpoolEditor.value = true
    const vehicleId = vehicleStore.activeVehicle?.id
    if (!vehicleId) return
    try {
      recentCarpoolNames.value = recentPassengerNames((await api.getCarpools(vehicleId, { limit: 30 })).trips)
    } catch {
      // The names are a convenience: the editor works without them
      recentCarpoolNames.value = []
    }
  }

  return { showCarpoolEditor, carpoolCreateOptions, carpoolOpenToken, recentCarpoolNames, openCarpoolEditor }
}
