import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import DriveFilterGroups from './DriveFilterGroups.vue'

const render = (props: Record<string, unknown> = {}) =>
  renderToString(
    createSSRApp({
      render: () =>
        h(DriveFilterGroups as any, {
          unqualifiedCount: 0,
          proPersoEnabled: false,
          tollDetectionActive: false,
          unqualifiedOnly: false,
          hasTollOnly: false,
          tollSource: '',
          carpoolFilter: '',
          selectedTag: '',
          ...props,
        }),
    }).use(i18n),
  )

describe('DriveFilterGroups', () => {
  it('has no label and no "all" chip: the filters are toggles that a second press clears', async () => {
    const html = await render({ proPersoEnabled: true })
    expect(html).not.toContain('Filtres rapides')
    expect(html).not.toContain('>Tous<')
    expect(html).toContain('Pro')
    expect(html).toContain('Perso')
    expect(html.match(/aria-pressed/g)).toHaveLength(4)
  })

  it('shows the work and personal toggles only when the classification is on', async () => {
    const html = await render()
    expect(html).not.toContain('Perso')
    expect(html.match(/aria-pressed/g)).toHaveLength(2)
  })

  it('marks the active tag as selected', async () => {
    const html = await render({ proPersoEnabled: true, selectedTag: 'Pro' })
    expect(html.match(/chip-selected/g)).toHaveLength(1)
    expect(html).toContain('aria-pressed="true"')
  })
})
