import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import SegmentedControl from './SegmentedControl.vue'

const options = [
  { value: 'month', label: 'Mois' },
  { value: 'year', label: 'Année' },
  { value: 'all', label: 'Tout' },
]
const render = (modelValue: string) =>
  renderToString(createSSRApp({ render: () => h(SegmentedControl as never, { options, label: 'Période', modelValue }) }))

describe('SegmentedControl', () => {
  it('est un radiogroup nommé avec un radio par option', async () => {
    const html = await render('year')
    expect(html).toContain('role="radiogroup"')
    expect(html).toContain('aria-label="Période"')
    expect(html.match(/role="radio"/g)).toHaveLength(3)
  })

  it('marque seule l’option choisie, avec le style de sélection partagé', async () => {
    const html = await render('year')
    expect(html.match(/aria-checked="true"/g)).toHaveLength(1)
    expect(html.match(/chip-selected/g)).toHaveLength(1)
    expect(html.indexOf('chip-selected')).toBeGreaterThan(html.indexOf('Mois'))
    expect(html.indexOf('chip-selected')).toBeLessThan(html.indexOf('Année'))
  })

  it('garde une seule tabulation, sur l’option choisie', async () => {
    const html = await render('year')
    expect(html.match(/tabindex="0"/g)).toHaveLength(1)
    expect(html.match(/tabindex="-1"/g)).toHaveLength(2)
    expect(html.indexOf('tabindex="0"')).toBeGreaterThan(html.indexOf('Mois'))
  })

  it('retombe sur la première option si la valeur n’en désigne aucune', async () => {
    const html = await render('zzz')
    expect(html).not.toContain('aria-checked="true"')
    expect(html.indexOf('tabindex="0"')).toBeLessThan(html.indexOf('Mois'))
  })

  it('réserve 44 px au toucher', async () => {
    expect(await render('month')).toContain('min-h-11')
  })
})
