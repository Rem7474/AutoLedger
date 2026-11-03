import { describe, expect, it } from 'vitest'
import { visibleSessions } from './sessionList'

const s = (id: string, last_used_at: string, current = false) => ({ id, last_used_at, current })
const list = [s('old', '2026-10-01T10:00:00Z'), s('new', '2026-11-02T10:00:00Z'), s('me', '2026-10-15T10:00:00Z', true), s('mid', '2026-10-20T10:00:00Z')]

describe('visibleSessions', () => {
  it('puts this device first, then the most recently used, cut to the limit', () => {
    expect(visibleSessions(list, false, 3).map((x) => x.id)).toEqual(['me', 'new', 'mid'])
  })

  it('lists every device once expanded, in the same order', () => {
    expect(visibleSessions(list, true).map((x) => x.id)).toEqual(['me', 'new', 'mid', 'old'])
  })

  it('does not reorder the list it is given', () => {
    const before = list.map((x) => x.id)
    visibleSessions(list, true)
    expect(list.map((x) => x.id)).toEqual(before)
  })
})
