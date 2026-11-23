import { describe, expect, it } from 'vitest'
import { scenarioSide, sideKey, trackedSide } from './comparisonSide'

describe('trackedSide', () => {
  it('compares plug-in and range-extender hybrids as hybrids', () => {
    expect(trackedSide({ powertrain: 'PHEV' })).toBe('hybrid')
    expect(trackedSide({ powertrain: 'REEV' })).toBe('hybrid')
  })
  it('keeps the electric wording for other vehicles and for projections', () => {
    expect(trackedSide({ powertrain: 'EV' })).toBe('electric')
    expect(trackedSide({ powertrain: 'ICE' })).toBe('electric')
    expect(trackedSide(undefined)).toBe('electric')
  })
  it('follows the capabilities the server sent', () => {
    expect(trackedSide({ capabilities: { charges: true, refuels: true } })).toBe('hybrid')
    expect(trackedSide({ capabilities: { charges: true, refuels: false } })).toBe('electric')
  })
})

describe('sideKey', () => {
  it('moves a key under comparison.hybrid only for hybrids', () => {
    expect(sideKey('comparison.verdict.less', 'hybrid')).toBe('comparison.hybrid.verdict.less')
    expect(sideKey('comparison.verdict.less', 'electric')).toBe('comparison.verdict.less')
  })
})

describe('scenarioSide', () => {
  const vehicles = [{ id: 'a', powertrain: 'PHEV' }, { id: 'b', powertrain: 'EV' }]
  it('follows the powertrain of a tracked comparison only', () => {
    expect(scenarioSide({ mode: 'RETROSPECTIVE', vehicle_id: 'a' }, vehicles)).toBe('hybrid')
    expect(scenarioSide({ mode: 'RETROSPECTIVE', vehicle_id: 'b' }, vehicles)).toBe('electric')
    expect(scenarioSide({ mode: 'PROJECTION', vehicle_id: 'a' }, vehicles)).toBe('electric')
    expect(scenarioSide({ mode: 'RETROSPECTIVE', vehicle_id: 'gone' }, vehicles)).toBe('electric')
    expect(scenarioSide(null, vehicles)).toBe('electric')
  })
})
