import { describe, expect, it } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from '@vue/server-renderer'
import { i18n } from '@/i18n'
import OdometerReadingForm from './OdometerReadingForm.vue'

const render = (props: Record<string, unknown>) =>
  renderToString(createSSRApp({ render: () => h(OdometerReadingForm as any, { date: '2026-10-15', odometer: '60500', notes: '', submitting: false, ...props }) }).use(i18n))

describe('OdometerReadingForm', () => {
  it('propose Supprimer en modification', async () => {
    expect(await render({ editing: true, canEdit: true })).toContain('Supprimer')
  })
  it('ne propose pas Supprimer à la création', async () => {
    expect(await render({ editing: false })).not.toContain('Supprimer')
  })
  it('ne propose pas Supprimer en lecture seule', async () => {
    expect(await render({ editing: true, canEdit: false })).not.toContain('Supprimer')
  })
})
