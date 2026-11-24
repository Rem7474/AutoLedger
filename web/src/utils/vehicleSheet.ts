// Which entries the vehicle sheet shows, in its three groups. The view renders from this list so the grouping and the
// capability and role conditions can be tested without mounting the page.
export type VehicleSheetGroup = 'follow' | 'ownership' | 'settings'

export type VehicleSheetEntry =
  | 'maintenance' | 'consumption' | 'tires' | 'readings'
  | 'battery' | 'hybridShare' | 'resale' | 'lease' | 'totals' | 'contract' | 'comparison'
  | 'homeCharging' | 'dataSources' | 'sharing' | 'edit' | 'manage'

export interface VehicleSheetInput {
  charges: boolean
  refuels: boolean
  isOwner: boolean
  hasTco: boolean
  vehicleCount: number
}

export function vehicleSheetGroups(i: VehicleSheetInput): Record<VehicleSheetGroup, VehicleSheetEntry[]> {
  const follow: VehicleSheetEntry[] = ['maintenance']
  if (i.charges || i.refuels) follow.push('consumption')
  follow.push('tires', 'readings')

  const ownership: VehicleSheetEntry[] = []
  if (i.charges) ownership.push('battery')
  if (i.charges && i.refuels) ownership.push('hybridShare')
  ownership.push('resale', 'lease')
  if (i.hasTco) ownership.push('totals')
  if (i.isOwner) ownership.push('contract')
  ownership.push('comparison')

  const settings: VehicleSheetEntry[] = []
  if (i.charges) settings.push('homeCharging')
  settings.push('dataSources', 'sharing')
  if (i.isOwner) settings.push('edit')
  if (i.vehicleCount > 1) settings.push('manage')

  return { follow, ownership, settings }
}
