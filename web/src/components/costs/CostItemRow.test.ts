import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { Zap } from 'lucide-vue-next'
import { i18n, setLocale } from '@/i18n'
import CostItemRow from './CostItemRow.vue'

const render = (props: Record<string, unknown>, slots: Record<string, () => unknown> = {}) =>
  renderToString(createSSRApp({ render: () => h(CostItemRow, { icon: Zap, label: 'Energy', amount: 42, currency: 'EUR', ...props }, slots) }).use(i18n))

beforeEach(() => {
  vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
  setLocale('en')
})

afterEach(() => {
  vi.unstubAllGlobals()
  setLocale('fr')
})

describe('cost item row', () => {
  it('tints the icon tile with the colour of the category', async () => {
    const html = await render({ color: '#38bdf8' })
    expect(html).toContain('data-testid="cost-row-icon"')
    expect(html).toContain('background-color:#38bdf826')
    expect(html).toContain('color:#38bdf8')
    expect(html).not.toContain('bg-slate-800 text-slate-300')
  })

  it('keeps the neutral tile without a colour', async () => {
    const html = await render({})
    expect(html).toContain('bg-slate-800 text-slate-300')
    expect(html).not.toContain('background-color')
  })

  it('shows the share and the cost per distance only when both are known', async () => {
    expect(await render({ sharePct: 25, costPerKm: 0.05 })).toContain('(25')
    expect(await render({ sharePct: 25 })).not.toContain('(25')
    expect(await render({ costPerKm: 0.05 })).not.toContain('/km')
  })

  it('shows a credit in green with its minus sign', async () => {
    const html = await render({ amount: -12, credit: true })
    expect(html).toContain('text-success-400')
    expect(html).toMatch(/-.?12/)
    expect(await render({ amount: 12 })).not.toContain('text-success-400')
  })

  it('renders the note, and the slots it offers', async () => {
    const html = await render(
      { sub: 'smoothed over the year' },
      { badge: () => h('i', 'badge-slot'), aside: () => h('i', 'aside-slot'), actions: () => h('i', 'actions-slot'), below: () => h('i', 'below-slot') },
    )
    for (const part of ['smoothed over the year', 'badge-slot', 'aside-slot', 'actions-slot', 'below-slot']) expect(html).toContain(part)
  })
})
