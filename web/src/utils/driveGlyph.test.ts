import { describe, expect, it } from 'vitest'
import { Navigation, Users, Briefcase, User } from '@/icons'
import { driveGlyph } from './driveGlyph'

describe('driveGlyph', () => {
  it('says carpooled first, then work or personal when the split is on, else the plain trip glyph', () => {
    expect(driveGlyph({ tags: ['Pro'] }, true, true)).toBe(Users)
    expect(driveGlyph({ tags: ['Pro'] }, false, true)).toBe(Briefcase)
    expect(driveGlyph({ tags: ['Perso'] }, false, true)).toBe(User)
    expect(driveGlyph({ tags: ['Pro'] }, false, false)).toBe(Navigation)
    expect(driveGlyph({}, false, true)).toBe(Navigation)
  })
})
