<script setup lang="ts">
import { Zap, Disc, Wrench, Shield } from '@/icons'
import { formatAmount } from '@/currency'
import { formatQuantity } from '@/utils/numbers'
import { t } from '@/i18n'
import { distanceUnit, formatDistance, formatVolumeValue, perDistance, perVolume, volumeUnitLabel } from '@/units'
import CostItemRow from '@/components/costs/CostItemRow.vue'
import { computed } from 'vue'
import { hybridDriveEnergy } from '@/utils/hybridShare'

// The energy, tires, maintenance and insurance lines of the cost breakdown of a drive or a trip
const props = defineProps<{ drive: any; breakdown: any; currency: string }>()

// The fuel line of a fuel-only vehicle: litres burnt at the average price of its fill-ups, nothing while they are unknown
function fuelSub(): string {
  const c = props.drive.costs
  if (c?.fuel_liters == null) return ''
  return t('drives.driveCostModal.fuelVolume', {
    volume: formatVolumeValue(Number(c.fuel_liters)),
    unit: volumeUnitLabel(),
    rate: formatAmount(perVolume(Number(c.fuel_rate_per_liter) || 0), props.currency, 3),
  })
}

// A plug-in hybrid's energy line carries both parts: the electric one from the kWh, the fuel one from the litres
const hybrid = computed(() => hybridDriveEnergy(props.drive))
const hybridSub = computed(() => {
  const h = hybrid.value
  if (!h) return ''
  if (h.kind === 'unknown') return t('drives.driveCostModal.hybridUnknownSub')
  const electric = t('drives.driveCostModal.hybridSplit', {
    percent: h.electricPercent,
    kwh: formatQuantity(h.kwh),
    rate: formatAmount(props.drive.costs?.electricity_rate ?? 0, props.currency, 3),
  })
  if (h.electricPercent >= 100) return electric
  const fuel =
    h.litres === null
      ? t('drives.driveCostModal.hybridFuelPartUnknown')
      : t('drives.driveCostModal.hybridFuelPart', {
          volume: formatVolumeValue(h.litres),
          unit: volumeUnitLabel(),
          rate: formatAmount(perVolume(Number(props.drive.costs?.fuel_rate_per_liter) || 0), props.currency, 3),
        })
  return `${electric} + ${fuel}`
})
</script>

<template>
  <!-- 1. Électricité -->
  <CostItemRow
    :icon="Zap"
    :label="drive.costs?.fuel_only ? $t('drives.driveCostModal.fuelEnergy') : hybrid ? $t('drives.driveCostModal.hybridEnergy') : $t('drives.driveCostModal.electricEnergy')"
    :sub="drive.costs?.fuel_only ? fuelSub() : hybrid ? hybridSub : $t('drives.driveCostModal.kwhKwh', { electricity_kwh: formatQuantity(Number(drive.costs?.electricity_kwh) || 0), electricity_rate: formatAmount(drive.costs?.electricity_rate ?? 0, currency, 3) })"
    :amount="drive.costs?.energy_cost || 0"
    :share-pct="breakdown.byKey.energy.sharePct"
    :cost-per-km="breakdown.byKey.energy.costPerKm"
    :color="breakdown.byKey.energy.color"
    :currency="currency"
  >
    <template #badge>
      <span v-if="drive.costs?.fuel_only && drive.costs?.energy_source === 'UNKNOWN'" class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium">{{ $t('drives.driveCostModal.fuelUnknown') }}</span>
      <span v-else-if="drive.costs?.fuel_only" />
      <span v-else-if="hybrid?.kind === 'unknown'" class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium">{{ $t('drives.driveCostModal.hybridUnknown') }}</span>
      <span v-else-if="drive.costs?.electricity_rate_source === 'UNKNOWN'" class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium">{{ $t('drives.driveCostModal.rateUnknown') }}</span>
      <span v-else-if="drive.costs?.energy_source === 'DEFAULT'" class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium">{{ $t('drives.driveCostModal.estimate') }}</span>
    </template>
  </CostItemRow>

  <!-- 2. Pneus -->
  <CostItemRow
    :icon="Disc"
    :label="$t('drives.driveCostModal.tireWear')"
    :sub="`${formatDistance(drive.distance_km, 1)} × ${formatAmount(perDistance(drive.costs?.tires_rate ?? 0), currency, 3)}/${distanceUnit()}`"
    :amount="drive.costs?.tires_cost || 0"
    :share-pct="breakdown.byKey.tires.sharePct"
    :cost-per-km="breakdown.byKey.tires.costPerKm"
    :color="breakdown.byKey.tires.color"
    :currency="currency"
  >
    <template #badge>
      <span v-if="drive.costs?.tires_rate_source === 'UNKNOWN'" class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium">{{ $t('drives.driveCostModal.rateUnknown') }}</span>
      <span v-else-if="drive.costs?.tires_rate_source === 'INCLUDED_IN_LEASE'" class="text-[9px] px-1.5 py-0.5 rounded bg-success-500/10 text-success-400 font-medium">{{ $t('drives.driveCostModal.includedInTheLease2') }}</span>
    </template>
  </CostItemRow>

  <!-- 3. Entretien -->
  <CostItemRow
    :icon="Wrench"
    :label="$t('drives.driveCostModal.maintenanceProvision')"
    :sub="`${formatDistance(drive.distance_km, 1)} × ${formatAmount(perDistance(drive.costs?.maintenance_rate ?? 0), currency, 3)}/${distanceUnit()}`"
    :amount="drive.costs?.maintenance_cost || 0"
    :share-pct="breakdown.byKey.maintenance.sharePct"
    :cost-per-km="breakdown.byKey.maintenance.costPerKm"
    :color="breakdown.byKey.maintenance.color"
    :currency="currency"
  >
    <template #badge>
      <span v-if="drive.costs?.maintenance_rate_source === 'UNKNOWN'" class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium">{{ $t('drives.driveCostModal.rateUnknown') }}</span>
      <span v-else-if="drive.costs?.maintenance_rate_source === 'INCLUDED_IN_LEASE'" class="text-[9px] px-1.5 py-0.5 rounded bg-success-500/10 text-success-400 font-medium">{{ $t('drives.driveCostModal.includedInTheLease2') }}</span>
    </template>
  </CostItemRow>

  <!-- 4. Assurance -->
  <CostItemRow
    :icon="Shield"
    :label="$t('drives.driveCostModal.insuranceShareFixedCost')"
    :sub="`${formatDistance(drive.distance_km, 1)} × ${formatAmount(perDistance(drive.costs?.insurance_rate || 0), currency, 3)}/${distanceUnit()}`"
    :amount="drive.costs?.insurance_cost || 0"
    :share-pct="breakdown.byKey.insurance.sharePct"
    :cost-per-km="breakdown.byKey.insurance.costPerKm"
    :color="breakdown.byKey.insurance.color"
    :currency="currency"
  >
    <template #badge>
      <span
        v-if="drive.costs?.insurance_source === 'RECORDED_EXPENSES'"
        class="text-[9px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-300 font-medium"
        :title="$t('drives.driveCostModal.premiumsPaidOverTheLast')"
      >
        {{ $t('drives.driveCostModal.actualPremiums') }}
      </span>
      <span v-else-if="drive.costs?.insurance_source === 'INCLUDED_IN_LEASE'" class="text-[9px] px-1.5 py-0.5 rounded bg-success-500/10 text-success-400 font-medium">
        {{ $t('drives.driveCostModal.includedInTheLease') }}
      </span>
      <span
        v-else-if="drive.costs?.insurance_source === 'INSUFFICIENT_DISTANCE'"
        class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium"
        :title="$t('drives.driveCostModal.lessThan500KmDriven', { min: formatDistance(500) })"
      >
        {{ $t('drives.driveCostModal.notEnoughKm') }}
      </span>
      <span v-else class="text-[9px] px-1.5 py-0.5 rounded bg-warning-500/10 text-warning-400 font-medium" :title="$t('drives.driveCostModal.noInsurancePremiumRecordedIn')">
        {{ $t('drives.driveCostModal.notEntered') }}
      </span>
    </template>
  </CostItemRow>
</template>
