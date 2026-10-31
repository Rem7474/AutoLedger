import { afterEach, describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n, setLocale } from '@/i18n'
import KpiCard from './KpiCard.vue'

const render = (props: Record<string, unknown>, slot?: string, badge?: string) =>
  renderToString(createSSRApp({
    render: () => h(KpiCard as any, props, {
      ...(slot ? { default: () => h('p', slot) } : {}),
      ...(badge ? { badge: () => h('span', badge) } : {}),
    }),
  }).use(i18n))

afterEach(() => setLocale('fr'))

describe('KpiCard', () => {
  it('shows the label, the value with its unit and the period', async () => {
    const html = await render({ label: 'Economic cost', value: '0,142', unit: '/km', period: 'last 12 months' })
    expect(html).toContain('Economic cost')
    expect(html).toContain('0,142')
    expect(html).toContain('/km')
    expect(html).toContain('last 12 months')
  })

  it('reads an unknown value as a dash, without its unit', async () => {
    const html = await render({ label: 'Cost per km', value: null, unit: '/km' })
    expect(html).toContain('—')
    expect(html).not.toContain('/km<')
    expect(html).toContain('text-slate-400')
  })

  it('reads the formatters\' own unknown marker as unknown', async () => {
    const html = await render({ label: 'Efficiency', value: '—', unit: ' kWh' })
    expect(html).not.toContain('kWh')
    expect(html).toContain('text-slate-400')
  })

  it('tints a tile with the danger tone', async () => {
    const html = await render({ label: 'Most worn', value: '85 %', tone: 'danger', tinted: true })
    expect(html).toContain('bg-danger-500/5')
    expect(html).toContain('text-danger-400')
  })

  it('has a dense variant for the panels inside a disclosure', async () => {
    const html = await render({ label: 'Consumption', value: '15,2', unit: ' kWh/100 km', compact: true })
    expect(html).toContain('bg-slate-950/60')
    expect(html).toContain('text-xl')
    expect(html).not.toContain('uppercase')
  })

  it('keeps the uppercase label on the full-size card', async () => {
    expect(await render({ label: 'Economic cost', value: '0,142' })).toContain('uppercase')
  })

  it('tints a status tile with its tone and keeps the plain tile otherwise', async () => {
    const good = await render({ label: 'Saving', value: '5 €', tone: 'success', tinted: true, compact: true })
    expect(good).toContain('border-success-500/30')
    expect(good).toContain('bg-success-500/5')
    expect(good).not.toContain('bg-slate-950/60')
    const worse = await render({ label: 'Extra', value: '5 €', tone: 'warning', tinted: true })
    expect(worse).toContain('border-warning-500/30')
    // A tone alone colours the value without tinting the tile
    const plain = await render({ label: 'Net', value: '5 €', tone: 'success' })
    expect(plain).toContain('text-success-400')
    expect(plain).not.toContain('bg-success-500/5')
  })

  it('shows the icon of a compact tile in front of its label', async () => {
    const icon = { render: () => h('svg', { class: 'piggy' }) }
    expect(await render({ label: 'Saving', value: '5 €', compact: true, icon })).toContain('class="piggy mr-1.5')
    expect(await render({ label: 'Saving', value: '5 €', icon })).not.toContain('mr-1.5')
  })

  it('takes its rows from the grid only when asked to line up with its neighbours', async () => {
    const lined = await render({ label: 'Economic cost', value: '0,142', aligned: true }, 'secondary line')
    expect(lined).toContain('grid-rows-subgrid')
    expect(lined).toContain('row-span-3')
    // The secondary lines stay together in the last row
    expect(lined.indexOf('secondary line')).toBeGreaterThan(lined.indexOf('0,142'))
    const alone = await render({ label: 'Economic cost', value: '0,142' })
    expect(alone).not.toContain('subgrid')
    expect(alone).toContain('flex flex-col')
  })

  it('adds the definition tip when a glossary term is given', async () => {
    setLocale('en')
    const html = await render({ label: 'Cash paid', value: '1 200 €', term: 'cashPaid' })
    expect(html).toContain('role="note"')
  })

  it('renders the delta with its tone and the secondary lines from the slot', async () => {
    const html = await render({ label: 'Month', value: '90 €', delta: { text: '+12 %', tone: 'bad' } }, 'of which tolls')
    expect(html).toContain('+12 %')
    expect(html).toContain('text-danger-400')
    expect(html).toContain('of which tolls')
  })

  it('colours the value with a warning tone and shows a badge next to the label', async () => {
    const html = await render({ label: 'Extra cost', value: '480 €', tone: 'warning' }, undefined, 'estimated')
    expect(html).toContain('text-warning-400')
    expect(html).toContain('estimated')
  })

  it('opens the detail through a button when there is no target', async () => {
    const html = await render({ label: 'Energy', value: '300 €', actionLabel: 'Open panel' })
    expect(html).toContain('<button')
    expect(html).toContain('Open panel')
  })
})
