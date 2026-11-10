import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DetailDeleteButton from './DetailDeleteButton.vue'

const render = (props: Record<string, unknown>) => renderToString(createSSRApp({ render: () => h(DetailDeleteButton, props) }).use(i18n))

describe('DetailDeleteButton', () => {
  it('affiche « Supprimer » en texte discret, sans rose', async () => {
    const html = await render({})
    expect(html).toContain('Supprimer')
    expect(html).toContain('text-critical-400')
    expect(html).not.toContain('rose')
  })

  it('est absent si non supprimable', async () => {
    expect(await render({ deletable: false })).not.toContain('<button')
  })

  it('est absent en lecture seule', async () => {
    expect(await render({ canEdit: false })).not.toContain('<button')
  })
})
