import type { Component } from 'vue'
import { Activity, Car, Garage, LayoutDashboard, Navigation as NavIcon, Receipt } from '@/icons'

/**
 * A main destination of the app. The bottom bar of a phone and the sidebar of a desktop are both drawn from this list;
 * the other pages are reached from these destinations (the vehicle sheet, the garage, the account menu).
 */
export interface NavDestination {
  name: string
  labelKey: string
  /** A shorter label for the bottom bar, when the sidebar one is longer. */
  mobileLabelKey?: string
  path: string
  icon: Component
  /** The pages that keep this destination lit, besides the one of the same name. */
  pages: string[]
  /** Present only for a vehicle that has (`drives`) or lacks (`no-drives`) the `drives` capability. */
  when?: 'drives' | 'no-drives'
}

/**
 * The destinations, in order. The second one is the trips of a vehicle with the `drives` capability (every vehicle: the
 * carpools are a segment of that page) and the consumption of one without it. The vehicle sheet reaches the odometer,
 * the consumption, the tires, the maintenance and the comparison, so those pages keep it lit.
 */
export const NAV_DESTINATIONS: NavDestination[] = [
  { name: 'dashboard', labelKey: 'shell.nav.dashboard', mobileLabelKey: 'shell.nav.home', path: '/', icon: LayoutDashboard, pages: ['dashboard'] },
  { name: 'drives', labelKey: 'shell.nav.drives', path: '/drives', icon: NavIcon, pages: ['drives'], when: 'drives' },
  { name: 'consumption', labelKey: 'shell.nav.consumption', path: '/consumption', icon: Activity, pages: ['consumption'], when: 'no-drives' },
  { name: 'costs', labelKey: 'shell.nav.expenses', path: '/costs', icon: Receipt, pages: ['costs', 'expenses'] },
  {
    name: 'vehicle',
    labelKey: 'shell.nav.vehicle',
    path: '/vehicle',
    icon: Car,
    pages: ['vehicle', 'odometer', 'consumption', 'tires', 'comparison', 'maintenance', 'vehicles'],
  },
]

/** The garage (every vehicle, and where one is added): a secondary entry of the sidebar, always shown. */
export const GARAGE_DESTINATION: NavDestination = {
  name: 'vehicles',
  labelKey: 'shell.nav.vehicles',
  path: '/vehicles',
  icon: Garage,
  pages: ['vehicles'],
}

/** The four destinations of a vehicle, according to whether it has the `drives` capability. */
export function navDestinations(hasDrives: boolean): NavDestination[] {
  return NAV_DESTINATIONS.filter((d) => !d.when || (d.when === 'drives') === hasDrives)
}

/** The destination a page belongs to, or undefined for a page outside the four (the account page). */
export function activeDestination(destinations: NavDestination[], routeName: unknown): string | undefined {
  const exact = destinations.find((d) => d.name === routeName)
  return (exact ?? destinations.find((d) => d.pages.includes(String(routeName))))?.name
}

/**
 * The entry lit in the sidebar: the garage when its page is open (a page lights one entry only), otherwise the
 * destination the page belongs to.
 */
export function activeSidebarEntry(destinations: NavDestination[], routeName: unknown): string | undefined {
  return routeName === GARAGE_DESTINATION.name ? GARAGE_DESTINATION.name : activeDestination(destinations, routeName)
}
