import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import CountBadge from './CountBadge.vue'

const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(CountBadge, props as never) }))

describe('CountBadge', () => {
  it('affiche le nombre', async () => {
    expect(await render({ count: 3 })).toContain('>3<')
  })

  it('se masque à 0, sauf avec showZero', async () => {
    expect(await render({ count: 0 })).not.toContain('<span')
    expect(await render({ count: 0, showZero: true })).toContain('>0<')
  })

  it('plafonne l’affichage', async () => {
    expect(await render({ count: 150 })).toContain('>99+<')
    expect(await render({ count: 12, max: 9 })).toContain('>9+<')
    expect(await render({ count: 99 })).toContain('>99<')
  })

  it.each([
    ['attention', 'text-warning-300'],
    ['neutral', 'text-slate-200'],
    ['danger', 'text-critical-300'],
  ])('le ton %s prend %s', async (tone, cls) => {
    expect(await render({ count: 2, tone })).toContain(cls)
  })

  it('expose un libellé accessible quand il est fourni', async () => {
    const html = await render({ count: 2, label: '2 coûts à compléter' })
    expect(html).toContain('aria-label="2 coûts à compléter"')
    expect(html).toContain('role="img"')
    expect(await render({ count: 2 })).not.toContain('aria-label')
  })

  it('n’est pas interactif', async () => {
    const html = await render({ count: 2 })
    expect(html).not.toContain('<button')
    expect(html).not.toContain('tabindex')
  })
})
