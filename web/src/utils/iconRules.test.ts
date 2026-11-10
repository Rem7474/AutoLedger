import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// One family of icons (Material Symbols Rounded), reached only through `@/icons`: a screen never imports a glyph file or
// another icon library, so changing the family or a glyph stays a change of the `icons/` folder.
function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !name.endsWith('.test.ts') ? [path] : []
  })
}

describe('icon rules', () => {
  const root = join(__dirname, '..')
  const files = sources(root).map((file) => relative(root, file))
  const importing = (pattern: RegExp, allowed: (rel: string) => boolean) =>
    files.filter((rel) => !allowed(rel) && pattern.test(readFileSync(join(root, rel), 'utf8')))

  it('imports no other icon library', () => {
    expect(importing(/from\s+['"](lucide-vue-next|@heroicons|@mdi|@iconify|vue-feather-icons)/, () => false)).toEqual([])
  })

  it('imports a Material Symbols file only from the icons folder', () => {
    expect(importing(/@material-symbols\//, (rel) => rel.startsWith('icons/'))).toEqual([])
  })
})
