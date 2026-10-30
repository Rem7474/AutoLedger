import { chargeOrigin } from './chargeProvenance'

/** The synchronisation re-reads the last 30 days (`resyncOverlap`): a cost set in TeslaMate on an older charge is not picked up. */
const RESYNC_WINDOW_MS = 30 * 24 * 3600 * 1000

const GRAFANA_MARGIN_MS = 60_000

function isTeslaMateCharge(c: any): boolean {
  return !!c?.teslamate_charge_id && chargeOrigin(c) === 'TESLAMATE'
}

/** Link to the charge in the TeslaMate Grafana "Charge Details" dashboard (standard TeslaMate dashboard uid),
 * when the vehicle has a Grafana URL configured. The time range is padded so the whole charge is visible. */
export function teslamateChargeUrl(vehicle: any, c: any): string | null {
  if (!vehicle?.teslamate_grafana_url || !isTeslaMateCharge(c)) return null
  const start = new Date(c.date).getTime()
  const end = c.end_date ? new Date(c.end_date).getTime() : start
  const params = new URLSearchParams({
    orgId: '1',
    from: String(start - GRAFANA_MARGIN_MS),
    to: String(end + GRAFANA_MARGIN_MS),
    'var-car_id': String(vehicle.teslamate_car_id || 1),
    'var-charging_process_id': String(c.teslamate_charge_id),
  })
  if (vehicle.teslamate_url) params.set('var-base_url', vehicle.teslamate_url)
  return `${vehicle.teslamate_grafana_url}/d/BHhxFeZRz/charge-details?${params.toString()}`
}

/** Link to the page of TeslaMate where the cost of a charge is set. Only for a charge whose cost is unknown and
 * that the next synchronisation will read again: the cost typed there comes back on its own. */
export function teslamateChargeCostUrl(vehicle: any, c: any, now: number = Date.now()): string | null {
  if (!vehicle?.teslamate_url || !isTeslaMateCharge(c)) return null
  if (c.cost !== null && c.cost !== undefined) return null
  if (c.cost_source === 'MANUAL') return null
  if (now - new Date(c.date).getTime() > RESYNC_WINDOW_MS) return null
  return `${vehicle.teslamate_url}/charge-cost/${c.teslamate_charge_id}`
}
