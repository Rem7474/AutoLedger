import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import FuelFillUpModal from './FuelFillUpModal.vue'

const log = { id: 'f1', version: 3, date: '2026-03-10', amount: 40, liters: 25, is_full_tank: true }
const render = (props: Record<string, unknown>) => {
  const app = createSSRApp({ render: () => h(FuelFillUpModal as any, { open: true, vehicleId: 'v1', editing: null, canEdit: true, ...props }) })
  app.directive('dialog', { getSSRProps: () => ({}) })
  app.use(createPinia()).use(i18n)
  return renderToString(app)
}

describe('FuelFillUpModal', () => {
  beforeEach(() => {
    vi.stubGlobal('localStorage', { getItem: () => null, setItem: () => undefined })
    setActivePinia(createPinia())
  })

  it("ne rend rien tant qu'elle n'est pas ouverte", async () => {
    expect(await render({ open: false })).not.toContain('plein')
  })
  it('propose un nouveau plein à la création, sans Supprimer', async () => {
    const html = await render({})
    expect(html).toContain('Nouveau plein')
    expect(html).not.toContain('Supprimer')
  })
  it('modifie un plein existant et propose Supprimer', async () => {
    const html = await render({ editing: log })
    expect(html).toContain('Modifier le plein')
    expect(html).toContain('Supprimer')
  })
  it('ne propose pas Supprimer en lecture seule', async () => {
    expect(await render({ editing: log, canEdit: false })).not.toContain('Supprimer')
  })
  it("garde l'annulation et l'enregistrement dans le pied", async () => {
    const html = await render({ editing: log })
    expect(html).toContain('Annuler')
    expect(html).toContain('Enregistrer')
  })
})
