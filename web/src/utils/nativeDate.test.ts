import { beforeEach, describe, expect, it, vi } from 'vitest'
import { enGB, fr } from 'date-fns/locale'
import { monthLabel } from './nativeDate'

async function load() {
  vi.resetModules()
  return import('./nativeDate')
}

describe('nativeKind', () => {
  it('picks the input type from the picker options, the month winning over the time', async () => {
    const { nativeKind } = await load()
    expect(nativeKind({})).toBe('date')
    expect(nativeKind({ enableTimePicker: true })).toBe('datetime-local')
    expect(nativeKind({ monthPicker: true })).toBe('month')
    expect(nativeKind({ monthPicker: true, enableTimePicker: true })).toBe('month')
  })
})

describe('toNativeValue', () => {
  it('leaves an empty value empty', async () => {
    const { toNativeValue } = await load()
    for (const empty of [null, undefined, '']) expect(toNativeValue(empty, 'date')).toBe('')
  })

  it('cuts a stored value down to what the input of that kind accepts', async () => {
    const { toNativeValue } = await load()
    expect(toNativeValue('2026-03-04', 'date')).toBe('2026-03-04')
    expect(toNativeValue('2026-03-04T18:30:00', 'date')).toBe('2026-03-04')
    expect(toNativeValue('2026-03-04T18:30:00', 'datetime-local')).toBe('2026-03-04T18:30')
    expect(toNativeValue('2026-03-04', 'datetime-local')).toBe('2026-03-04T00:00')
    expect(toNativeValue('2026-03-04', 'month')).toBe('2026-03')
  })

  it('drops what the input would reject instead of showing a blank field with a stale value', async () => {
    const { toNativeValue } = await load()
    expect(toNativeValue('04/03/2026', 'date')).toBe('')
    expect(toNativeValue('2026-03', 'date')).toBe('')
    expect(toNativeValue('soon', 'month')).toBe('')
    expect(toNativeValue('18:30', 'datetime-local')).toBe('')
    expect(toNativeValue(new Date('nope'), 'date')).toBe('')
  })

  it('reads a Date in local time and the month object of the desktop picker', async () => {
    const { toNativeValue } = await load()
    const d = new Date(2026, 2, 4, 7, 5)
    expect(toNativeValue(d, 'date')).toBe('2026-03-04')
    expect(toNativeValue(d, 'datetime-local')).toBe('2026-03-04T07:05')
    expect(toNativeValue(d, 'month')).toBe('2026-03')
    expect(toNativeValue({ year: 2026, month: 0 }, 'month')).toBe('2026-01')
  })
})

describe('supportsNativeInput', () => {
  beforeEach(() => vi.resetModules())

  // A browser that does not know a type reports a text box
  const doc = (known: string[]) => ({
    createElement: vi.fn(() => {
      let type = 'text'
      return {
        setAttribute: (_: string, v: string) => {
          if (known.includes(v)) type = v
        },
        get type() {
          return type
        },
      }
    }),
  })

  it('is false without a document, as when rendered on the server', async () => {
    const { supportsNativeInput } = await load()
    expect(supportsNativeInput('date', undefined)).toBe(false)
  })

  it('tells the kinds the browser knows from the ones it turns into text', async () => {
    const { supportsNativeInput } = await load()
    const d = doc(['date', 'datetime-local'])
    expect(supportsNativeInput('date', d as never)).toBe(true)
    expect(supportsNativeInput('datetime-local', d as never)).toBe(true)
    expect(supportsNativeInput('month', d as never)).toBe(false)
  })

  it('probes a kind once', async () => {
    const { supportsNativeInput } = await load()
    const d = doc(['date'])
    supportsNativeInput('date', d as never)
    supportsNativeInput('date', d as never)
    expect(d.createElement).toHaveBeenCalledTimes(1)
  })
})

describe('toRange', () => {
  it('waits for both days', async () => {
    const { toRange } = await load()
    expect(toRange('2026-01-01', '')).toBeNull()
    expect(toRange('', '2026-01-31')).toBeNull()
    expect(toRange('', '')).toBeNull()
  })

  it('puts the two days in chronological order whichever was picked first', async () => {
    const { toRange } = await load()
    expect(toRange('2026-01-01', '2026-01-31')).toEqual(['2026-01-01', '2026-01-31'])
    expect(toRange('2026-01-31', '2026-01-01')).toEqual(['2026-01-01', '2026-01-31'])
    expect(toRange('2026-01-05', '2026-01-05')).toEqual(['2026-01-05', '2026-01-05'])
  })
})

describe('monthLabel', () => {
  it('writes the month in the language asked, with a capital', () => {
    expect(monthLabel('2026-11', fr)).toBe('Novembre 2026')
    expect(monthLabel('2026-11', enGB)).toBe('November 2026')
  })

  it('is empty when there is no month', () => {
    expect(monthLabel('', fr)).toBe('')
    expect(monthLabel('soon', fr)).toBe('')
  })
})
