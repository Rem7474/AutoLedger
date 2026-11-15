export type Powertrain = 'EV' | 'ICE' | 'PHEV' | 'REEV'

export interface OnboardingVehicleForm {
  name: string
  make: string
  model: string
  vin: string
  powertrain: Powertrain
  odometer: number | string
  currency?: string
}

// The wizard only creates the vehicle: data sources are connected later from the vehicle's "Data sources" button.
// telemetry_mode is derived by the server from the connected sources, never sent by the client.
export function buildVehiclePayload(form: OnboardingVehicleForm): Record<string, unknown> {
  const payload: Record<string, unknown> = {
    name: form.name,
    powertrain: form.powertrain,
    make: form.make,
    model: form.model,
    vin: form.vin || undefined,
    current_odometer: Number(form.odometer) || 0,
    teslamate_auth_type: 'NONE',
  }
  if (form.currency) payload.currency = form.currency
  return payload
}
