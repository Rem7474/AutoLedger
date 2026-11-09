import { defineStore } from 'pinia'
import { ref } from 'vue'

const PRO_PERSO_KEY = 'teslacost_pro_perso'

const storage = () => (typeof localStorage === 'undefined' ? null : localStorage)

// Display preferences of the person using this browser (like the language), not data of the vehicle.
export const usePreferencesStore = defineStore('preferences', () => {
  // Work / personal classification of drives; on by default
  const proPersoEnabled = ref(storage()?.getItem(PRO_PERSO_KEY) !== 'false')

  function setProPersoEnabled(enabled: boolean) {
    proPersoEnabled.value = enabled
    storage()?.setItem(PRO_PERSO_KEY, String(enabled))
  }

  return { proPersoEnabled, setProPersoEnabled }
})
