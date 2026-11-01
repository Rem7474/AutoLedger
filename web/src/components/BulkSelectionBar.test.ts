import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import BulkSelectionBar from './BulkSelectionBar.vue'
import { Pencil, Trash2 } from 'lucide-vue-next'

const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(BulkSelectionBar as any, props, { default: () => h('button', { class: 'btn' }, 'Act') }) }).use(i18n))

describe('BulkSelectionBar', () => {
  it('renders nothing while nothing is selected', async () => {
    const html = await render({ count: 0 })
    expect(html).not.toContain('Act')
    expect(html).not.toContain('<div')
  })

  it('is a contextual bar at the top edge on a phone and sticks under the header from the sm breakpoint', async () => {
    const html = await render({ count: 2, noun: 'drive' })
    expect(html).toContain('fixed inset-x-0 top-0')
    expect(html).toContain('sm:sticky')
    expect(html).toContain('Act')
  })

  it('lists the actions on a desktop bar and keeps an unavailable one disabled', async () => {
    const actions = [
      { key: 'edit', label: 'Edit row', icon: Pencil, run: () => {}, disabled: true, reason: 'Select a single row' },
      { key: 'delete', label: 'Delete rows', icon: Trash2, run: () => {}, danger: true },
    ]
    const html = await render({ count: 2, noun: 'drive', actions })
    expect(html).toContain('Edit row')
    expect(html).toContain('Delete rows')
    expect(html).toMatch(/disabled/)
  })
})
