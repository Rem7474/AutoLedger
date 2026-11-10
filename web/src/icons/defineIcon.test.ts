import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { defineIcon } from './defineIcon'
import * as icons from './index'

const svg = '<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 -960 960 960"><path d="M10 10h20Z"/></svg>'

describe('defineIcon', () => {
  it('draws the path in the current colour and leaves the size to the class', async () => {
    const Icon = defineIcon('test', svg)
    const html = await renderToString(createSSRApp({ render: () => h(Icon, { class: 'h-5 w-5', 'aria-hidden': 'true' }) }))
    expect(html).toContain('viewBox="0 -960 960 960"')
    expect(html).toContain('fill="currentColor"')
    expect(html).toContain('class="h-5 w-5"')
    expect(html).toContain('aria-hidden="true"')
    expect(html).toContain('d="M10 10h20Z"')
    expect(html).not.toContain('width="48"')
  })

  it('refuses a file that is not a single path', () => {
    expect(() => defineIcon('bad', '<svg viewBox="0 0 1 1"><g/></svg>')).toThrow(/bad/)
  })
})

describe('the icon set', () => {
  it('has a drawable glyph for every name it exports', async () => {
    for (const [name, Icon] of Object.entries(icons)) {
      const html = await renderToString(createSSRApp({ render: () => h(Icon) }))
      expect(html, name).toMatch(/<svg[^>]* viewBox="0 -960 960 960"[^>]*><path d="[^"]+"/)
    }
  })
})
