import type { APITokenCall } from '@/services/api'

/** How long a call took, as a person reads it: milliseconds under a second, seconds with one decimal above. */
export function formatCallDuration(ms: number): string {
  if (ms < 1000) return `${Math.max(0, Math.round(ms))} ms`
  return `${(ms / 1000).toFixed(1)} s`
}

export interface DescribedCall {
  tool: string
  /** The name of the vehicle the call named by id, null when it named none or the account no longer shows it. */
  vehicle: string | null
  ok: boolean
  duration: string
}

export function describeCall(call: APITokenCall, vehicles: readonly { id: string; name: string }[]): DescribedCall {
  const vehicle = call.vehicle_id ? (vehicles.find((v) => v.id === call.vehicle_id)?.name ?? null) : null
  return { tool: call.tool, vehicle, ok: call.outcome === 'ok', duration: formatCallDuration(call.duration_ms) }
}
