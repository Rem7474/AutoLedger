import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import StatusLabel from './StatusLabel.vue'

const render = (props: Record<string, unknown>, text = 'Coût manquant') =>
  renderToString(createSSRApp({ render: () => h(StatusLabel, props as never, () => text) }))
const Icon = { render: () => h('svg', { class: 'ico' }) }

describe('StatusLabel', () => {
  it('écrit toujours le libellé', async () => {
    expect(await render({ tone: 'info' }, 'Estimé')).toContain('Estimé')
  })

  it.each([
    ['attention', 'text-warning-400'],
    ['info', 'text-info-300'],
    ['danger', 'text-critical-400'],
    ['neutral', 'text-slate-400'],
  ])('le ton %s prend la couleur %s', async (tone, cls) => {
    const html = await render({ tone })
    expect(html).toContain(cls)
    expect(html).toContain(`data-tone="${tone}"`)
  })

  it('ne porte ni fond, ni bordure, ni forme de pastille', async () => {
    for (const tone of ['attention', 'info', 'danger', 'neutral']) {
      const html = await render({ tone, icon: Icon })
      expect(html).not.toMatch(/\bbg-/)
      expect(html).not.toMatch(/\bborder/)
      expect(html).not.toContain('rounded')
      expect(html).not.toContain('<button')
    }
  })

  it('utilise le ton neutre et la taille sm par défaut', async () => {
    const html = await render({})
    expect(html).toContain('text-slate-400')
    expect(html).toContain('text-xs')
  })

  it('agrandit le texte et l’icône en taille md', async () => {
    const html = await render({ size: 'md', icon: Icon })
    expect(html).toContain('text-sm')
    expect(html).toContain('h-4 w-4')
  })

  it('masque l’icône aux lecteurs d’écran, et n’en dessine pas sans icon', async () => {
    expect(await render({ icon: Icon })).toContain('aria-hidden="true"')
    expect(await render({})).not.toContain('<svg')
  })
})
