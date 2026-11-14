import type { VehicleCapabilities } from '@/utils/capabilities'

/** The capability a page needs; a page that is not listed is open to every vehicle. */
export const PAGE_CAPABILITY: Record<string, keyof VehicleCapabilities> = {
  drives: 'drives',
}

/** Where an address goes when the active vehicle lacks the capability of the page: the home of the vehicle. */
export const CAPABILITY_FALLBACK = '/vehicle'

/** The address to send the user to, or null when the vehicle can open the page. */
export function capabilityRedirect(routeName: unknown, capabilities: VehicleCapabilities): string | null {
  const needed = typeof routeName === 'string' ? PAGE_CAPABILITY[routeName] : undefined
  return needed && !capabilities[needed] ? CAPABILITY_FALLBACK : null
}
