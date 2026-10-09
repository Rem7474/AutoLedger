import { ref } from 'vue'
import { intlLocale } from '@/i18n'

// Distances are always stored and sent to the API in kilometers (TeslaMate, toll data and the
// existing history are all metric); this module only converts for display and form input,
// driven by the signed-in account's stored preference (see stores/auth.ts).
export const SUPPORTED_DISTANCE_UNITS = ['km', 'mi'] as const
export type DistanceUnit = (typeof SUPPORTED_DISTANCE_UNITS)[number]
export const DEFAULT_DISTANCE_UNIT: DistanceUnit = 'km'

const KM_PER_MILE = 1.609344

const isSupported = (value: string | null | undefined): value is DistanceUnit =>
  !!value && (SUPPORTED_DISTANCE_UNITS as readonly string[]).includes(value)

const unit = ref<DistanceUnit>(DEFAULT_DISTANCE_UNIT)

/** Applies the account's stored choice; an unknown or missing value falls back to km. */
export function setDistanceUnit(value: string | null | undefined) {
  unit.value = isSupported(value) ? value : DEFAULT_DISTANCE_UNIT
}

export const currentDistanceUnit = (): DistanceUnit => unit.value

/** A stored km value, converted to the account's unit. */
export function kmToDisplayDistance(km: number): number {
  return unit.value === 'mi' ? km / KM_PER_MILE : km
}

/** The reverse: a value typed in the account's unit, converted back to km for the API. */
export function displayDistanceToKm(value: number): number {
  return unit.value === 'mi' ? value * KM_PER_MILE : value
}

/** A stored km value in the account's unit, locale-formatted without the unit ("42 000", "26,097.3"). */
export function formatDistanceValue(km: number, digits = 0): string {
  return kmToDisplayDistance(km).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

/** Locale-formatted distance with its unit suffix, e.g. "42 000 km" / "26,097 mi". */
export function formatDistance(km: number, digits = 0): string {
  return `${formatDistanceValue(km, digits)} ${unit.value}`
}

/**
 * A figure expressed per km (a cost per km, kWh per 100 km, a price per extra km), rescaled to the
 * account's unit: per mile it is 1.609 times larger. Pair it with "/{unit}" in the label.
 */
export function perDistance(valuePerKm: number): number {
  return unit.value === 'mi' ? valuePerKm * KM_PER_MILE : valuePerKm
}

/** A figure per km (or per 100 km) rescaled to the account's unit, locale-formatted without its unit. */
export function formatPerDistanceValue(valuePerKm: number, digits = 1): string {
  return perDistance(valuePerKm).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

/** The reverse of perDistance, for a per-distance figure typed in a form. */
export function perDistanceToPerKm(value: number): number {
  return unit.value === 'mi' ? value / KM_PER_MILE : value
}

/** Speed unit that goes with the distance unit. */
export const speedUnit = (): string => (unit.value === 'mi' ? 'mph' : 'km/h')

/** A speed stored in km/h, in the account's unit. */
export function formatSpeed(kmh: number): string {
  return `${Math.round(kmToDisplayDistance(kmh)).toLocaleString(intlLocale())} ${speedUnit()}`
}

/** The account's distance unit label, for a catalog's {unit} placeholder ("Distance ({unit})"). */
export const distanceUnit = (): DistanceUnit => unit.value

// Fuel volumes are always stored and sent to the API in litres; like distances, they are only
// converted for display and form input, driven by the account's volume unit.
export const SUPPORTED_VOLUME_UNITS = ['l', 'gal_us', 'gal_uk'] as const
export type VolumeUnit = (typeof SUPPORTED_VOLUME_UNITS)[number]
export const DEFAULT_VOLUME_UNIT: VolumeUnit = 'l'

const LITRES_PER_UNIT: Record<VolumeUnit, number> = { l: 1, gal_us: 3.785411784, gal_uk: 4.54609 }
const VOLUME_LABELS: Record<VolumeUnit, string> = { l: 'L', gal_us: 'gal US', gal_uk: 'gal UK' }

const isSupportedVolume = (value: string | null | undefined): value is VolumeUnit =>
  !!value && (SUPPORTED_VOLUME_UNITS as readonly string[]).includes(value)

const volume = ref<VolumeUnit>(DEFAULT_VOLUME_UNIT)

/** Applies the account's stored choice; an unknown or missing value falls back to litres. */
export function setVolumeUnit(value: string | null | undefined) {
  volume.value = isSupportedVolume(value) ? value : DEFAULT_VOLUME_UNIT
}

export const currentVolumeUnit = (): VolumeUnit => volume.value

/** The unit label for a quantity ("L", "gal US", "gal UK"). */
export const volumeUnitLabel = (): string => VOLUME_LABELS[volume.value]

/** A stored litre quantity, converted to the account's unit. */
export function litresToDisplayVolume(litres: number): number {
  return litres / LITRES_PER_UNIT[volume.value]
}

/** The reverse: a quantity typed in the account's unit, converted back to litres for the API. */
export function displayVolumeToLitres(value: number): number {
  return value * LITRES_PER_UNIT[volume.value]
}

/** A stored litre quantity in the account's unit, locale-formatted without the unit. */
export function formatVolumeValue(litres: number, digits = 1): string {
  return litresToDisplayVolume(litres).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

/** Locale-formatted quantity with its unit suffix, e.g. "41,2 L" / "10,9 gal US". */
export function formatVolume(litres: number, digits = 1): string {
  return `${formatVolumeValue(litres, digits)} ${volumeUnitLabel()}`
}

/** A price per litre, rescaled to a price per unit of the account's volume. */
export function perVolume(valuePerLitre: number): number {
  return valuePerLitre * LITRES_PER_UNIT[volume.value]
}

/** The reverse of perVolume, for a price per unit typed in a form. */
export function perVolumeToPerLitre(value: number): number {
  return value / LITRES_PER_UNIT[volume.value]
}

/** A price per litre in the account's unit, locale-formatted without currency or unit. */
export function formatPerVolumeValue(valuePerLitre: number, digits = 3): string {
  return perVolume(valuePerLitre).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}

// Fuel consumption is stored per 100 km in litres. With miles and gallons it reads as miles per
// gallon (the usual figure there), otherwise as a volume per 100 distance units.
const isMpg = (): boolean => unit.value === 'mi' && volume.value !== 'l'

/** The consumption unit label: "L/100 km", "gal US/100 km", "L/100 mi" or "mpg US". */
export function fuelConsumptionUnit(): string {
  if (isMpg()) return volume.value === 'gal_uk' ? 'mpg UK' : 'mpg US'
  return `${volumeUnitLabel()}/100 ${unit.value}`
}

/** A stored L/100 km figure, in the account's consumption unit. */
export function l100kmToDisplayConsumption(l100: number): number {
  if (isMpg()) return l100 > 0 ? (100 * LITRES_PER_UNIT[volume.value]) / (KM_PER_MILE * l100) : 0
  return perDistance(l100) / LITRES_PER_UNIT[volume.value]
}

/** The reverse, for a consumption typed in a form. */
export function displayConsumptionToL100km(value: number): number {
  if (isMpg()) return value > 0 ? (100 * LITRES_PER_UNIT[volume.value]) / (KM_PER_MILE * value) : 0
  return perDistanceToPerKm(value * LITRES_PER_UNIT[volume.value])
}

/** A stored L/100 km figure, locale-formatted without its unit. */
export function formatFuelConsumptionValue(l100: number, digits = 1): string {
  return l100kmToDisplayConsumption(l100).toLocaleString(intlLocale(), { minimumFractionDigits: digits, maximumFractionDigits: digits })
}
