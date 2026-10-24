import { describe, expect, it } from 'vitest'
import type { APITokenCall } from '@/services/api'
import { describeCall, formatCallDuration } from './tokenCalls'

const call = (over: Partial<APITokenCall> = {}): APITokenCall => ({ at: '2026-10-24T10:00:00Z', tool: 'get_vehicle_costs', vehicle_id: null, outcome: 'ok', duration_ms: 42, ...over })

describe('formatCallDuration', () => {
  it('shows milliseconds under a second and seconds with one decimal above', () => {
    expect(formatCallDuration(0)).toBe('0 ms')
    expect(formatCallDuration(41.6)).toBe('42 ms')
    expect(formatCallDuration(999)).toBe('999 ms')
    expect(formatCallDuration(1000)).toBe('1.0 s')
    expect(formatCallDuration(2349)).toBe('2.3 s')
  })
})

describe('describeCall', () => {
  const vehicles = [{ id: 'v1', name: 'Family car' }]

  it('names the vehicle the call named by id', () => {
    expect(describeCall(call({ vehicle_id: 'v1' }), vehicles)).toEqual({ tool: 'get_vehicle_costs', vehicle: 'Family car', ok: true, duration: '42 ms' })
  })

  it('has no vehicle for a call that named none, or one the account no longer shows', () => {
    expect(describeCall(call(), vehicles).vehicle).toBeNull()
    expect(describeCall(call({ vehicle_id: 'gone' }), vehicles).vehicle).toBeNull()
  })

  it('tells a refused call from a served one', () => {
    expect(describeCall(call({ outcome: 'error' }), vehicles).ok).toBe(false)
  })
})
