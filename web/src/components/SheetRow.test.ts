import { afterEach, describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import SheetRow from './SheetRow.vue'

const phone = (matches: boolean) =>
  vi.stubGlobal('window', { matchMedia: () => ({ matches, addEventListener: () => {}, removeEventListener: () => {} }) })

const render = (props: Record<string, unknown> = {}) =>
  renderToString(createSSRApp({
    render: () => h(SheetRow as any, { title: 'Analysis', hint: 'Period and chart', ...props }, {
      default: ({ bare }: { bare: boolean }) => h('section', { 'data-bare': String(bare) }, 'the block'),
    }),
  }).use(i18n))

afterEach(() => vi.unstubAllGlobals())

describe('SheetRow', () => {
  it('leaves the block where it is on a wide screen, with its own card', async () => {
    phone(false)
    const html = await render()
    expect(html).toContain('data-bare="false"')
    expect(html).toContain('the block')
    expect(html).not.toContain('aria-haspopup')
  })

  it('shows one row with its title and hint on a phone, and keeps the block in the closed sheet', async () => {
    phone(true)
    const html = await render()
    expect(html).toContain('aria-haspopup="dialog"')
    expect(html).toContain('Analysis')
    expect(html).toContain('Period and chart')
    expect(html).not.toContain('the block')
  })

  it('shows nothing on a phone when the block has nothing to show', async () => {
    phone(true)
    const html = await render({ enabled: false })
    expect(html).not.toContain('aria-haspopup')
    expect(html).not.toContain('Analysis')
  })
})
