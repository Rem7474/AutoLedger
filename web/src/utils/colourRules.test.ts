import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A feature does not get an accent hue of its own: a cost's hue is its theme (`text-theme-*`, utils/themes.ts), anything
// else is slate. Green, amber and red are statuses, rose is the brand.
const FORBIDDEN = /\b(?:[a-z-]+:)*(?:text|bg|border|ring|from|to|via|stroke|fill|shadow|divide|outline|placeholder|accent|caret)-(?:violet|indigo|cyan|teal|blue|pink|purple|fuchsia|orange|lime|yellow)-\d{2,3}/g

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return /\.(vue|ts)$/.test(name) && !/\.test\.ts$/.test(name) ? [path] : []
  })
}

describe('colour rules', () => {
  it('uses no raw accent hue: the colour of a cost is a theme token', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const rel = relative(root, file)
      const hits = readFileSync(file, 'utf8').match(FORBIDDEN)
      if (hits) found.push(`${rel}: ${[...new Set(hits)].join(' ')}`)
    }
    expect(found).toEqual([])
  })

  // The primary action is rose in every dialog; a solid green button reads as a status, not as the main action.
  it('keeps solid green fills for the confirmation dialog of a success', () => {
    const root = join(__dirname, '..')
    const found: string[] = []
    for (const file of sources(root)) {
      const rel = relative(root, file)
      if (rel === 'components/ConfirmModal.vue') continue
      if (/<button[^>]*\bbg-success-[56]00\b(?!\/)/.test(readFileSync(file, 'utf8'))) found.push(rel)
    }
    expect(found).toEqual([])
  })
})
