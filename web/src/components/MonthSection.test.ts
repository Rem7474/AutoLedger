import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import MonthSection from './MonthSection.vue'

const render = (props: Record<string, unknown>, slots: Record<string, () => unknown> = { default: () => h('p', 'rows') }) =>
  renderToString(createSSRApp({ render: () => h(MonthSection as any, props, slots) }).use(i18n))

describe('MonthSection', () => {
  it('shows the month, its sub-total and the rows', async () => {
    const html = await render({ label: 'novembre', subtotal: '+700 km' })
    expect(html).toContain('novembre')
    expect(html).toContain('+700 km')
    expect(html).toContain('rows')
  })

  it('omits the figure of a month that has none', async () => {
    const html = await render({ label: 'octobre', subtotal: null })
    expect(html).toContain('octobre')
    expect(html).not.toContain('tabular-nums')
  })

  it('lets a month give its own summary', async () => {
    const html = await render({ label: 'octobre' }, { default: () => h('p', 'rows'), summary: () => h('span', 'trois trajets') })
    expect(html).toContain('trois trajets')
  })
})
