import { Navigation, Users, Briefcase, User } from '@/icons'

/** The glyph of a drive row: what is particular to the drive (carpooled, work, personal); an ordinary drive keeps the plain trip glyph. */
export function driveGlyph(d: any, carpooled: boolean, proPersoEnabled: boolean) {
  if (carpooled) return Users
  if (proPersoEnabled && d.tags?.includes('Pro')) return Briefcase
  if (proPersoEnabled && d.tags?.includes('Perso')) return User
  return Navigation
}
