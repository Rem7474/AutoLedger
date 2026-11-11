import { describe, expect, it, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import ComparisonScenarioList from './ComparisonScenarioList.vue'

// The bar renders its actions only in the open "More" menu: the stub lists every action with its state
vi.mock('@/components/BulkSelectionBar.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      props: ['count', 'noun', 'actions'],
      render() {
        return h('ul', { 'data-noun': this.noun }, (this.actions as any[]).map((a) => h('li', { 'data-key': a.key, 'data-disabled': String(!!a.disabled), title: a.disabled ? a.reason : '' }, a.label)))
      },
    }),
  }
})

const retro = { id: 's1', name: 'Mon EV contre un SUV', mode: 'RETROSPECTIVE', annual_km: 12000, years: 5 }
const projection = { id: 's2', name: 'Projection citadine', mode: 'PROJECTION', annual_km: 8000, years: 3 }
const render = (scenarios: any[], selectedIds: string[] = []) =>
  renderToString(createSSRApp({ render: () => h(ComparisonScenarioList as any, { scenarios, selectedIds }) }).use(i18n))

describe('ComparisonScenarioList', () => {
  it('opens a comparison from the whole row, with no row menu and no icon buttons', async () => {
    const html = await render([retro, projection])
    expect(html).toContain('inset-0')
    expect(html).toContain('Ouvrir Mon EV contre un SUV')
    expect(html).not.toContain('aria-haspopup="menu"')
    expect(html).not.toContain('data-icon="edit"')
    expect(html).not.toContain('data-icon="delete"')
  })

  it('writes the mode, the distance per year and the duration as separate parts of the quiet line', async () => {
    const html = await render([retro, projection])
    expect(html).toContain('Véhicule suivi')
    expect(html).toContain('Projection')
    expect(html).toContain('5 an(s)')
    expect(html).toContain('3 an(s)')
  })

  it('shows the tick hint only when there are two comparisons to choose from', async () => {
    expect(await render([retro])).not.toContain('Cochez 2 ou 3')
    expect(await render([retro, projection])).toContain('Cochez 2 ou 3')
  })

  it('compares from two rows, edits one row and deletes several', async () => {
    const one = await render([retro, projection], ['s1'])
    expect(one).toMatch(/data-key="compare" data-disabled="true"/)
    expect(one).toMatch(/data-key="edit" data-disabled="false"/)
    const two = await render([retro, projection], ['s1', 's2'])
    expect(two).toMatch(/data-key="compare" data-disabled="false"/)
    expect(two).toMatch(/data-key="edit" data-disabled="true"/)
    expect(two).toMatch(/data-key="delete" data-disabled="false"/)
    expect(two).toContain('data-noun="scenario"')
  })

  it('stops offering a new tick once three comparisons are selected', async () => {
    const four = [retro, projection, { ...retro, id: 's3' }, { ...retro, id: 's4' }]
    const html = await render(four, ['s1', 's2', 's3'])
    expect(html).toMatch(/id="cmp-select-s4"[^>]*disabled/)
    expect(html).not.toMatch(/id="cmp-select-s1"[^>]*disabled/)
  })
})
