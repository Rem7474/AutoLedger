import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'

// A number field without `inputmode` opens a full keyboard on some phones; an integer wants the digit pad, an amount the decimal one
function vueFiles(dir: string): string[] {
  return readdirSync(dir).flatMap((name) => {
    const path = join(dir, name)
    if (statSync(path).isDirectory()) return vueFiles(path)
    return path.endsWith('.vue') ? [path] : []
  })
}

describe('number fields', () => {
  it('all say which keyboard they want', () => {
    const missing = vueFiles(__dirname).flatMap((file) => {
      const inputs = readFileSync(file, 'utf8').match(/<input\b[^>]*?type="number"[^>]*?\/?>/gs) ?? []
      return inputs.filter((tag) => !/\binputmode=/.test(tag)).map(() => file)
    })
    expect(missing).toEqual([])
  })
})
