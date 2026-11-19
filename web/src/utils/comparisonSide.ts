import { capabilitiesOf } from '@/utils/capabilities'

export type TrackedSide = 'electric' | 'hybrid'

// A vehicle that records both fill-ups and charging sessions is compared as a hybrid.
export function trackedSide(vehicle: Parameters<typeof capabilitiesOf>[0]): TrackedSide {
  const { charges, refuels } = capabilitiesOf(vehicle)
  return vehicle && charges && refuels ? 'hybrid' : 'electric'
}

// The hybrid wording lives under `comparison.hybrid.*`, mirroring the electric keys.
export function sideKey(key: string, side: TrackedSide): string {
  return side === 'hybrid' ? key.replace(/^comparison\./, 'comparison.hybrid.') : key
}

// Projections compare two entered vehicles; a tracked comparison follows its reference vehicle.
export function scenarioSide(scenario: { mode?: string; vehicle_id?: string | null } | null | undefined, vehicles: ({ id: string } & NonNullable<Parameters<typeof capabilitiesOf>[0]>)[]): TrackedSide {
  if (scenario?.mode !== 'RETROSPECTIVE') return 'electric'
  return trackedSide(vehicles.find((v) => v.id === scenario.vehicle_id))
}
