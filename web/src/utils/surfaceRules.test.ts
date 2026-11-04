import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative } from 'node:path'
import { describe, expect, it } from 'vitest'

// A block of content is `.card` (slate-900 on the slate-950 page, outlined from `sm` up). Writing the three utilities
// by hand again would leave that block with an outline on a phone, where the tone alone separates it from the page.
const CARD_UTILITIES = ['bg-slate-900', 'border', 'border-slate-800', 'rounded-2xl']

// The classes of one element, as written between quotes (a class built by an expression is not looked at)
function handWrittenCards(source: string): number {
  let count = 0
  for (const [, classes] of source.matchAll(/(?<![:\w-])class="([^"{]*)"/g)) {
    const tokens = classes.split(/\s+/)
    if (CARD_UTILITIES.every((utility) => tokens.includes(utility))) count++
  }
  return count
}

// What sits inside a card and is not content (an inset row, a recap block) is slate-800. A translucent slate-950 well
// with a slate outline is the older way to write it and gives every panel a slightly different depth.
function outlinedWells(source: string): number {
  let count = 0
  for (const [, classes] of source.matchAll(/(?<![:\w-])class="([^"{]*)"/g)) {
    const tokens = classes.split(/\s+/)
    const well = tokens.some((t) => /^bg-slate-950\/(40|50|60|70)$/.test(t))
    const outlined = tokens.some((t) => /^border-slate-800(\/\d+)?$/.test(t))
    // A tappable row keeps its well: it lightens under the finger
    if (well && outlined && !tokens.some((t) => t.startsWith('hover:border'))) count++
  }
  return count
}

function sources(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return sources(path)
    return name.endsWith('.vue') ? [path] : []
  })
}

describe('surface rules', () => {
  it('writes an inset as slate-800, not as an outlined translucent slate-950 well', () => {
    const root = join(__dirname, '..')
    const found = sources(root).filter((file) => outlinedWells(readFileSync(file, 'utf8')) > 0).map((file) => relative(root, file))
    expect(found).toEqual([])
  })

  it('writes a card as .card, not as its three utilities', () => {
    const root = join(__dirname, '..')
    const found = sources(root).filter((file) => handWrittenCards(readFileSync(file, 'utf8')) > 0).map((file) => relative(root, file))
    expect(found).toEqual([])
  })
})
