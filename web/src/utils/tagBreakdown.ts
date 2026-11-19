import { t } from '@/i18n'
import { distanceUnit, formatDistanceValue, formatVolume } from '@/units'
import { formatNumber } from '@/utils/numbers'

/** Distance and energy of a tag row: the kWh of a vehicle that charges, the litres of one that burns fuel only (none while unknown). */
export function tagEnergyLine(item: { distance_km: number; energy_kwh?: number; fuel_only?: boolean; fuel_liters?: number }): string {
  const distance_km = formatDistanceValue(item.distance_km, 1)
  const unit = distanceUnit()
  if (item.fuel_only) {
    return item.fuel_liters && item.fuel_liters > 0
      ? t('dashboard.tagBreakdown.kmFuel', { distance_km, unit, volume: formatVolume(item.fuel_liters) })
      : t('dashboard.tagBreakdown.kmOnly', { distance_km, unit })
  }
  return t('dashboard.tagBreakdown.kmKwh', { distance_km, unit, energy_kwh: formatNumber(item.energy_kwh ?? 0, 1) })
}
