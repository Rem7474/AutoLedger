import { readFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { THEMES, THEME_HEX, THEME_STYLES, themeOfExpense, type Theme } from './themes'

const read = (path: string) => readFileSync(join(__dirname, path), 'utf8')
const mainCss = read('../assets/main.css')
const tailwindCss = readFileSync(join(__dirname, '../../node_modules/tailwindcss/theme.css'), 'utf8')

// --- colour maths: Tailwind's oklch tokens to sRGB, WCAG contrast, CIEDE2000 and colour-blindness simulation
function oklchToHex(name: string): string {
  const m = tailwindCss.match(new RegExp(`--color-${name}: oklch\\(([\\d.]+)% ([\\d.]+) ([\\d.]+)\\)`))
  if (!m) throw new Error(`no token ${name}`)
  const L = Number(m[1]) / 100
  const a = Number(m[2]) * Math.cos((Number(m[3]) * Math.PI) / 180)
  const b = Number(m[2]) * Math.sin((Number(m[3]) * Math.PI) / 180)
  const l = (L + 0.3963377774 * a + 0.2158037573 * b) ** 3
  const mm = (L - 0.1055613458 * a - 0.0638541728 * b) ** 3
  const s = (L - 0.0894841775 * a - 1.291485548 * b) ** 3
  const lin = [4.0767416621 * l - 3.3077115913 * mm + 0.2309699292 * s, -1.2684380046 * l + 2.6097574011 * mm - 0.3413193965 * s, -0.0041960863 * l - 0.7034186147 * mm + 1.707614701 * s]
  const enc = (x: number) => (x <= 0.0031308 ? 12.92 * x : 1.055 * x ** (1 / 2.4) - 0.055)
  return `#${lin.map((x) => Math.round(Math.min(1, Math.max(0, enc(x))) * 255).toString(16).padStart(2, '0')).join('')}`
}
const toLinear = (c: number) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4)
const linearOf = (hex: string) => [1, 3, 5].map((i) => toLinear(parseInt(hex.slice(i, i + 2), 16) / 255))
const luminance = (hex: string) => {
  const [r, g, b] = linearOf(hex)
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}
const contrast = (a: string, b: string) => (Math.max(luminance(a), luminance(b)) + 0.05) / (Math.min(luminance(a), luminance(b)) + 0.05)

type Vision = 'normal' | 'deuteranopia' | 'protanopia'
const CVD = {
  deuteranopia: [[0.367322, 0.860646, -0.228041], [0.280085, 0.672501, 0.047413], [-0.01182, 0.04294, 0.968881]],
  protanopia: [[0.152286, 1.052583, -0.204868], [0.114503, 0.786281, 0.099216], [-0.003882, -0.048116, 1.051998]],
}
const lab = (hex: string, vision: Vision) => {
  const l = linearOf(hex)
  const [r, g, b] = vision === 'normal' ? l : CVD[vision].map((row) => Math.min(1, Math.max(0, row[0] * l[0] + row[1] * l[1] + row[2] * l[2])))
  const X = (0.4124 * r + 0.3576 * g + 0.1805 * b) / 0.95047
  const Y = 0.2126 * r + 0.7152 * g + 0.0722 * b
  const Z = (0.0193 * r + 0.1192 * g + 0.9505 * b) / 1.08883
  const f = (t: number) => (t > 216 / 24389 ? Math.cbrt(t) : ((24389 / 27) * t + 16) / 116)
  return [116 * f(Y) - 16, 500 * (f(X) - f(Y)), 200 * (f(Y) - f(Z))]
}
function deltaE(hexA: string, hexB: string, vision: Vision = 'normal'): number {
  const [L1, a1, b1] = lab(hexA, vision)
  const [L2, a2, b2] = lab(hexB, vision)
  const rad = Math.PI / 180
  const Cb = (Math.hypot(a1, b1) + Math.hypot(a2, b2)) / 2
  const G = 0.5 * (1 - Math.sqrt(Cb ** 7 / (Cb ** 7 + 25 ** 7)))
  const ap1 = (1 + G) * a1
  const ap2 = (1 + G) * a2
  const Cp1 = Math.hypot(ap1, b1)
  const Cp2 = Math.hypot(ap2, b2)
  const hue = (b: number, a: number) => ((Math.atan2(b, a) / rad) + 360) % 360
  const h1 = Cp1 ? hue(b1, ap1) : 0
  const h2 = Cp2 ? hue(b2, ap2) : 0
  let dh = 0
  if (Cp1 * Cp2) dh = Math.abs(h2 - h1) <= 180 ? h2 - h1 : h2 - h1 + (h2 > h1 ? -360 : 360)
  const dH = 2 * Math.sqrt(Cp1 * Cp2) * Math.sin((dh * rad) / 2)
  const Lb = (L1 + L2) / 2
  const Cpb = (Cp1 + Cp2) / 2
  let hb = h1 + h2
  if (Cp1 * Cp2) hb = Math.abs(h1 - h2) <= 180 ? (h1 + h2) / 2 : h1 + h2 < 360 ? (h1 + h2 + 360) / 2 : (h1 + h2 - 360) / 2
  const T = 1 - 0.17 * Math.cos((hb - 30) * rad) + 0.24 * Math.cos(2 * hb * rad) + 0.32 * Math.cos((3 * hb + 6) * rad) - 0.2 * Math.cos((4 * hb - 63) * rad)
  const Sl = 1 + (0.015 * (Lb - 50) ** 2) / Math.sqrt(20 + (Lb - 50) ** 2)
  const Sc = 1 + 0.045 * Cpb
  const Sh = 1 + 0.015 * Cpb * T
  const Rt = -Math.sin(2 * 30 * Math.exp(-(((hb - 275) / 25) ** 2)) * rad) * 2 * Math.sqrt(Cpb ** 7 / (Cpb ** 7 + 25 ** 7))
  const dL = (L2 - L1) / Sl
  const dC = (Cp2 - Cp1) / Sc
  const dHs = dH / Sh
  return Math.sqrt(dL ** 2 + dC ** 2 + dHs ** 2 + Rt * dC * dHs)
}

// The token each theme reads in main.css, resolved to its hex
const tokenOf = (name: string): string => {
  const m = mainCss.match(new RegExp(`--color-theme-${name}: var\\(--color-([a-z]+-\\d+)\\)`))
  if (!m) throw new Error(`no theme token ${name}`)
  return oklchToHex(m[1])
}
const TOKEN_NAME: Partial<Record<Theme, string>> = {
  energy: 'energy', tolls: 'tolls', tires: 'tires', maintenance: 'maintenance', repairs: 'maintenance', insurance: 'insurance',
  financing: 'financing', subscriptions: 'neutral', taxes: 'taxes', accessories: 'neutral', other: 'neutral',
}
const HUES: Theme[] = ['energy', 'tolls', 'tires', 'maintenance', 'insurance', 'financing']
const RESERVED = { brand: 'rose-600', good: 'emerald-500', warning: 'amber-500', danger: 'red-500' }
const SURFACES = [oklchToHex('slate-900'), oklchToHex('slate-800')]

describe('themes', () => {
  it('gives each theme the colour of its token in main.css', () => {
    for (const theme of THEMES) {
      const token = TOKEN_NAME[theme]
      // the depreciation slice is a chart-only neutral, a step darker than the neutral text
      const expected = token ? tokenOf(token) : oklchToHex('slate-500')
      expect([theme, THEME_HEX[theme]]).toEqual([theme, expected])
    }
  })

  it('keeps every theme text readable on a card and on a field', () => {
    for (const theme of THEMES) {
      if (theme === 'depreciation') continue
      for (const surface of SURFACES) expect(contrast(THEME_HEX[theme], surface), `${theme} on ${surface}`).toBeGreaterThanOrEqual(4.5)
    }
  })

  it('keeps the hues apart from each other, in normal vision and under colour blindness', () => {
    for (const [i, a] of HUES.entries()) {
      for (const b of HUES.slice(i + 1)) {
        expect(deltaE(THEME_HEX[a], THEME_HEX[b]), `${a}/${b}`).toBeGreaterThanOrEqual(15)
        for (const vision of ['deuteranopia', 'protanopia'] as const) expect(deltaE(THEME_HEX[a], THEME_HEX[b], vision), `${a}/${b} ${vision}`).toBeGreaterThanOrEqual(8)
      }
    }
  })

  it('keeps the hues apart from the brand and the three statuses', () => {
    for (const theme of HUES) {
      for (const [role, token] of Object.entries(RESERVED)) {
        // the energy sky is not a status: it only has to stay clear of them like the others
        expect(deltaE(THEME_HEX[theme], oklchToHex(token)), `${theme}/${role}`).toBeGreaterThanOrEqual(15)
      }
    }
  })

  it('has a chip, an accent bar and a text class for every theme, all of its own token', () => {
    for (const theme of THEMES) {
      const style = THEME_STYLES[theme]
      const token = TOKEN_NAME[theme] ?? 'neutral'
      expect(style.badge).toContain(`text-theme-${token}`)
      expect(style.accent).toBe(`border-l-theme-${token}`)
      expect(style.text).toBe(`text-theme-${token}`)
    }
  })

  it('maps the expense categories to their theme, neutral for the others', () => {
    expect(themeOfExpense('INSURANCE')).toBe('insurance')
    expect(themeOfExpense('SUBSCRIPTION')).toBe('subscriptions')
    expect(themeOfExpense('TAX')).toBe('taxes')
    expect(themeOfExpense('REPAIR')).toBe('repairs')
    expect(themeOfExpense('OTHER')).toBe('other')
  })
})
