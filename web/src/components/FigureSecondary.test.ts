import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import FigureSecondary from './FigureSecondary.vue'

const render = (props: Record<string, unknown>, slot?: string) =>
  renderToString(createSSRApp({ render: () => h(FigureSecondary as any, props, slot ? { default: () => h('p', slot) } : {}) }).use(i18n))

describe('FigureSecondary', () => {
  it('shows the label, the figure and its unit, with no frame', async () => {
    const html = await render({ label: 'Dernier relevé', value: '12 400', unit: ' km' }, '3 nov.')
    expect(html).toContain('Dernier relevé')
    expect(html).toContain('12 400')
    expect(html).toContain(' km')
    expect(html).toContain('3 nov.')
    expect(html).not.toContain('border')
    expect(html).not.toContain('bg-')
  })

  it('draws nothing for a figure with no data', async () => {
    expect(await render({ label: 'Depuis', value: null }, 'detail')).not.toContain('Depuis')
    expect(await render({ label: 'Depuis', value: '—' })).not.toContain('Depuis')
  })

  it('draws a dash for an empty figure that is kept', async () => {
    const html = await render({ label: 'Rendement', value: null, keepEmpty: true }, 'Complétez les recharges')
    expect(html).toContain('Rendement')
    expect(html).toContain('—')
    expect(html).toContain('Complétez les recharges')
  })
})
