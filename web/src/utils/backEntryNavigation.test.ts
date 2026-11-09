import { describe, expect, it } from 'vitest'
import { replaceOverHeldEntry } from './backEntryNavigation'

const to = { path: '/drives', fullPath: '/drives?x=1' }

describe('replaceOverHeldEntry', () => {
  it('replaces the held entry when the page changes', () => {
    expect(replaceOverHeldEntry(true, to, { path: '/' }, null)).toBe(true)
  })

  it('leaves the entry alone when none is held', () => {
    expect(replaceOverHeldEntry(false, to, { path: '/' }, null)).toBe(false)
  })

  it('leaves the entry alone when only the query or the hash changes', () => {
    expect(replaceOverHeldEntry(true, to, { path: '/drives' }, null)).toBe(false)
  })

  it('does not replace again the navigation it has already turned into a replace', () => {
    expect(replaceOverHeldEntry(true, to, { path: '/' }, '/drives?x=1')).toBe(false)
  })
})
