import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import GlossaryTip from './GlossaryTip.vue'

const render = () => renderToString(createSSRApp({ render: () => h(GlossaryTip as any, { term: 'cashPaid' }) }).use(i18n))

describe('GlossaryTip', () => {
  it('is the size of its glyph, and keeps a 44 px hit area through a pseudo-element, so it does not stretch the line it sits in', async () => {
    const html = await render()
    expect(html).toContain('no-tap')
    expect(html).toContain("after:-inset-3.5")
    expect(html).not.toContain('min-h-')
  })

  it('carries the definition as its accessible name and opens a dialog', async () => {
    const html = await render()
    expect(html).toContain('aria-haspopup="dialog"')
    expect(html).toMatch(/aria-label="[^"]+: [^"]+"/)
  })
})
