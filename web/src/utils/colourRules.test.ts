import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// Features do not get an accent hue of their own: a chip or an icon is slate, or sky when it links somewhere.
// Green, amber and red are statuses, rose is the brand. The expense categories keep their own palette.
const FORBIDDEN = /\b(?:[a-z-]+:)*(?:text|bg|border|ring|from|to|via|stroke|fill|shadow|divide|outline|placeholder|accent|caret)-(?:violet|indigo|cyan|teal|blue|pink|purple|fuchsia|orange|lime|yellow)-\d{2,3}/g
const ALLOWED = new Set(['utils/expenses.ts'])

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !/\.test\.ts$/.test(name) ? [path] : []
  })
}

describe('colour rules', () => {
  it('uses no accent hue outside the expense category palette', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const rel = relative(root, file)
      if (ALLOWED.has(rel)) continue
      const hits = readFileSync(file, 'utf8').match(FORBIDDEN)
      if (hits) found.push(`${rel}: ${[...new Set(hits)].join(' ')}`)
    }
    expect(found).toEqual([])
  })
})
