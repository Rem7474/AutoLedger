import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import KpiRow from './KpiRow.vue'

const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(KpiRow as any, props) }).use(i18n).component('RouterLink', { props: ['to'], render() { return h('a', { href: (this as any).to }, (this as any).$slots.default?.()) } }))

describe('KpiRow', () => {
  it('is a link to a page when it has a target, with its label, hint and figure', async () => {
    const html = await render({ label: 'Cash paid', value: '12 385 €', hint: 'last 10 months', to: '/expenses' })
    expect(html).toContain('href="/expenses"')
    expect(html).toContain('Cash paid')
    expect(html).toContain('last 10 months')
    expect(html).toContain('12 385 €')
  })

  it('is a button when it opens something on the page', async () => {
    const html = await render({ label: 'November 2026', value: '57,48 €' })
    expect(html).toContain('<button')
    expect(html).toContain('type="button"')
  })

  it('reads an unknown figure as a dash, never as zero', async () => {
    const html = await render({ label: 'Energy', value: null })
    expect(html).toContain('—')
    expect(html).toContain('text-slate-400')
  })
})
