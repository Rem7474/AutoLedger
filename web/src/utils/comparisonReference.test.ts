import { expect, it } from 'vitest'
import { buildComparisonPayload, comparisonReferenceVehicle, emptyComparisonForm, scenarioToForm } from './comparisonForm'

it('preserves a saved reference after editing while another vehicle is active', () => {
  const saved = { ...emptyComparisonForm(true), vehicle_id: 'car-A', name: 'Original' }
  const form = scenarioToForm(saved, true)
  form.name = 'Renamed'
  expect(buildComparisonPayload(form, 'car-B').vehicle_id).toBe('car-A')
  const vehicles = [{ id: 'car-A', currency: 'USD', powertrain: 'PHEV' }, { id: 'car-B', currency: 'ARS', powertrain: 'ICE' }]
  expect(comparisonReferenceVehicle(form.vehicle_id, vehicles[1], vehicles)).toEqual(vehicles[0])
})

it('uses the active vehicle only for a new comparison and never replaces an unavailable saved reference', () => {
  const active = { id: 'car-B' }
  expect(buildComparisonPayload(emptyComparisonForm(true), active.id).vehicle_id).toBe(active.id)
  expect(comparisonReferenceVehicle(undefined, active, [active])).toBe(active)
  expect(comparisonReferenceVehicle('removed-car', active, [active])).toBeUndefined()
  const form = scenarioToForm({ ...emptyComparisonForm(true), vehicle_id: 'removed-car' }, true)
  expect(buildComparisonPayload(form, active.id).vehicle_id).toBe('removed-car')
  form.mode = 'PROJECTION'
  expect(buildComparisonPayload(form, active.id)).not.toHaveProperty('vehicle_id')
})
