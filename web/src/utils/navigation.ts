export interface NavSection<T> {
  /** Translation key of the heading; absent for the section that stands alone at the top. */
  headingKey?: string
  items: T[]
}

/** The sections of the desktop sidebar, each listing the names of its pages in order. */
export const NAV_SECTIONS: { headingKey?: string; names: string[] }[] = [
  { names: ['dashboard'] },
  { headingKey: 'shell.navSections.tracking', names: ['drives', 'odometer', 'consumption'] },
  { headingKey: 'shell.navSections.costs', names: ['costs', 'maintenance', 'tires', 'comparison'] },
  { headingKey: 'shell.navSections.management', names: ['vehicle', 'vehicles', 'account'] },
]

/** A tab of the phone's bottom bar, and the pages that keep it lit. */
export interface PhoneTab {
  name: string
  pages: string[]
}

/**
 * The four tabs of the phone's bottom bar. The second one is the trips of a vehicle that has some (electric, hybrid; the
 * carpools are a segment of that page) and the consumption of one that has none (combustion); the other pages of the app are reached from these four, from the vehicle
 * sheet or from the account menu.
 */
export function phoneTabs(hasDrives: boolean): PhoneTab[] {
  return [
    { name: 'dashboard', pages: ['dashboard'] },
    hasDrives ? { name: 'drives', pages: ['drives'] } : { name: 'consumption', pages: ['consumption'] },
    { name: 'costs', pages: ['costs', 'expenses', 'maintenance'] },
    { name: 'vehicle', pages: ['vehicle', 'odometer', 'tires', 'comparison', 'vehicles', 'consumption'] },
  ]
}

/** The tab a page belongs to, or undefined for a page outside the four (the account page). */
export function activePhoneTab(tabs: PhoneTab[], routeName: unknown): string | undefined {
  const exact = tabs.find((tab) => tab.name === routeName)
  return (exact ?? tabs.find((tab) => tab.pages.includes(String(routeName))))?.name
}

/**
 * Spreads the pages the account can open over the sections, in the order the sections list them. A section left without
 * a page is dropped, and a page no section names is kept in a last section without a heading, so no page disappears.
 */
export function groupNavItems<T extends { name: string }>(items: T[], sections = NAV_SECTIONS): NavSection<T>[] {
  const placed = new Set<string>()
  const grouped: NavSection<T>[] = []
  for (const section of sections) {
    const inSection = section.names.map((name) => items.find((item) => item.name === name)).filter((item): item is T => !!item)
    inSection.forEach((item) => placed.add(item.name))
    if (inSection.length > 0) grouped.push({ headingKey: section.headingKey, items: inSection })
  }
  const rest = items.filter((item) => !placed.has(item.name))
  if (rest.length > 0) grouped.push({ items: rest })
  return grouped
}
