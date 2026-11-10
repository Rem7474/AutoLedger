import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { Plus, Receipt } from './index'
import { filledIcon } from './filled'

const draw = (icon: unknown) => renderToString(createSSRApp({ render: () => h(icon as any) }))

describe('filledIcon', () => {
  it('draws a destination in its filled version, which is not the outline one', async () => {
    const filled = filledIcon(Receipt)
    expect(filled).not.toBe(Receipt)
    expect(await draw(filled)).toContain('data-icon="receipt_long-fill"')
    expect(await draw(filled)).not.toBe(await draw(Receipt))
  })

  it('keeps an icon that has no filled version', () => {
    expect(filledIcon(Plus)).toBe(Plus)
  })
})
