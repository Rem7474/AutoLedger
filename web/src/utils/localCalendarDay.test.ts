import { afterEach, expect, it } from 'vitest'
import { toLocalDay } from './dates'

const previousTimezone = process.env.TZ
afterEach(() => { process.env.TZ = previousTimezone })

it('uses the local day when filling or editing a calendar date field', () => {
  process.env.TZ = 'America/Argentina/Buenos_Aires'
  expect(toLocalDay('2026-10-02T01:00:00Z')).toBe('2026-10-01')
  expect(toLocalDay('2026-10-01T03:00:00Z')).toBe('2026-10-01')
  process.env.TZ = 'Europe/Paris'
  expect(toLocalDay('2026-09-30T22:00:00Z')).toBe('2026-10-01')
})
