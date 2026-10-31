import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import BulkSelectionBar from './BulkSelectionBar.vue'

const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(BulkSelectionBar as any, props, { default: () => h('button', { class: 'btn' }, 'Act') }) }).use(i18n))

describe('BulkSelectionBar', () => {
  it('renders nothing while nothing is selected', async () => {
    expect((await render({ count: 0 })).replace(/<!--.*?-->/g, '')).toBe('')
  })

  it('docks at the bottom edge on a phone and sticks to the top from the sm breakpoint', async () => {
    const html = await render({ count: 2, noun: 'drive' })
    expect(html).toContain('fixed inset-x-0 bottom-0')
    expect(html).toContain('sm:sticky')
    expect(html).toContain('Act')
  })
})
