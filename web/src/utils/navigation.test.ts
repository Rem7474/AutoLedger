import { describe, expect, it } from 'vitest'
import { activePhoneTab, groupNavItems, NAV_SECTIONS, phoneTabs } from './navigation'

const item = (name: string) => ({ name })
const all = NAV_SECTIONS.flatMap((s) => s.names).map(item)

describe('groupNavItems', () => {
  it('spreads every page over the sections, in the order of the sections', () => {
    const grouped = groupNavItems(all)
    expect(grouped.map((g) => g.headingKey)).toEqual([undefined, 'shell.navSections.tracking', 'shell.navSections.costs', 'shell.navSections.management'])
    expect(grouped.flatMap((g) => g.items.map((i) => i.name))).toEqual(all.map((i) => i.name))
  })

  it('drops the pages the account cannot open and the sections left empty', () => {
    const noDrives = all.filter((i) => !['drives', 'odometer', 'consumption'].includes(i.name))
    const grouped = groupNavItems(noDrives)
    expect(grouped.map((g) => g.headingKey)).not.toContain('shell.navSections.tracking')
    expect(grouped.flatMap((g) => g.items.map((i) => i.name))).not.toContain('drives')
  })

  it('keeps a page that no section names, after the others', () => {
    const grouped = groupNavItems([...all, item('reports')])
    const last = grouped[grouped.length - 1]
    expect(last.headingKey).toBeUndefined()
    expect(last.items.map((i) => i.name)).toEqual(['reports'])
  })

  it('returns nothing for no page', () => {
    expect(groupNavItems([])).toEqual([])
  })
})

describe('phoneTabs', () => {
  it('has four tabs, the second one the trips of a vehicle that has some and the consumption of one that has none', () => {
    expect(phoneTabs(true).map((tab) => tab.name)).toEqual(['dashboard', 'drives', 'costs', 'vehicle'])
    expect(phoneTabs(false).map((tab) => tab.name)).toEqual(['dashboard', 'consumption', 'costs', 'vehicle'])
  })

  it('names only pages of the sidebar, so no page is reachable on a phone alone', () => {
    const named = NAV_SECTIONS.flatMap((s) => s.names)
    for (const tab of phoneTabs(true)) expect(named).toContain(tab.name)
  })
})

describe('activePhoneTab', () => {
  it('lights the tab a page belongs to', () => {
    expect(activePhoneTab(phoneTabs(true), 'drives')).toBe('drives')
    expect(activePhoneTab(phoneTabs(true), 'maintenance')).toBe('costs')
    expect(activePhoneTab(phoneTabs(true), 'tires')).toBe('vehicle')
  })

  it('gives the consumption page to the consumption tab when it is one, and to the vehicle tab otherwise', () => {
    expect(activePhoneTab(phoneTabs(false), 'consumption')).toBe('consumption')
    expect(activePhoneTab(phoneTabs(true), 'consumption')).toBe('vehicle')
  })

  it('lights no tab for a page outside the four', () => {
    expect(activePhoneTab(phoneTabs(true), 'account')).toBeUndefined()
    expect(activePhoneTab(phoneTabs(true), undefined)).toBeUndefined()
  })
})
