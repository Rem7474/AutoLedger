import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import TireWearBar from './TireWearBar.vue'

const render = (props: Record<string, unknown>) => renderToString(createSSRApp({ render: () => h(TireWearBar as any, props) }).use(i18n))

describe('TireWearBar', () => {
  it('fills with the wear by default', async () => {
    const html = await render({ pct: 22 })
    expect(html).toContain('width:22%')
    expect(html).toContain('aria-valuenow="22"')
  })

  it('fills with what is left when asked, and says so', async () => {
    const html = await render({ pct: 22, remaining: true })
    expect(html).toContain('width:78%')
    expect(html).toContain('aria-valuenow="78"')
    expect(html).toContain('Gomme restante')
  })

  it('stays inside the track for a tire past its lifespan', async () => {
    expect(await render({ pct: 130 })).toContain('width:100%')
    expect(await render({ pct: 130, remaining: true })).toContain('width:0%')
  })
})
