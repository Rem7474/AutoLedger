import { describe, expect, it } from 'vitest'
import { renderToString } from 'vue/server-renderer'
import { createSSRApp, h } from 'vue'
import SheetGrabber from './SheetGrabber.vue'

describe('SheetGrabber', () => {
  it('is decorative and takes the caller classes', async () => {
    const html = await renderToString(createSSRApp({ render: () => h(SheetGrabber, { class: 'md:hidden' }) }))
    expect(html).toContain('aria-hidden="true"')
    expect(html).toContain('md:hidden')
  })
})
