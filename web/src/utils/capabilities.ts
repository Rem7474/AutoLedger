import { canCharge, canRefuel, canLinkTeslaMate } from '@/utils/vehicles'

/**
 * What a vehicle records and offers, computed by the server and returned with the vehicle (`vehicle.capabilities`).
 * Components read a capability; they never compare the powertrain or test for a TeslaMate connection.
 */
export interface VehicleCapabilities {
  /** A battery to plug in: charging sessions, tariffs, home charging. */
  charges: boolean
  /** A tank: fill-ups. */
  refuels: boolean
  /** Trips apply: recorded by hand or written by a source. */
  drives: boolean
  /** A connected source (TeslaMate) synchronizes the vehicle's data. */
  sync: boolean
  /** A source provides the odometer. */
  odometer_auto: boolean
  /** The battery state of health can be read or recorded. */
  battery_health: boolean
  /** Trips are checked for tolls automatically. */
  tolls_auto: boolean
  /** The member may add and change records. */
  write: boolean
}

type VehicleLike =
  | {
      capabilities?: Partial<VehicleCapabilities> | null
      powertrain?: string | null
      role?: string | null
      teslamate_api_url?: string | null
      toll_detection_active?: boolean | null
    }
  | null
  | undefined

/**
 * The capabilities of a vehicle. A response without them (a list cached by the previous version of the app) is read
 * from the powertrain, the role and the connection with the same rules as the server; no vehicle reads as an
 * electric vehicle the user owns, the state before the first one is added.
 */
export function capabilitiesOf(vehicle: VehicleLike): VehicleCapabilities {
  const sent = vehicle?.capabilities
  if (sent && typeof sent.charges === 'boolean') {
    return {
      charges: sent.charges,
      refuels: !!sent.refuels,
      drives: !!sent.drives,
      sync: !!sent.sync,
      odometer_auto: !!sent.odometer_auto,
      battery_health: !!sent.battery_health,
      tolls_auto: !!sent.tolls_auto,
      write: !!sent.write,
    }
  }
  const powertrain = vehicle?.powertrain
  const charges = canCharge(powertrain)
  const sync = canLinkTeslaMate(powertrain) && !!vehicle?.teslamate_api_url?.trim()
  const role = vehicle?.role ?? 'OWNER'
  return {
    charges,
    refuels: canRefuel(powertrain),
    drives: charges,
    sync,
    odometer_auto: sync,
    battery_health: charges,
    tolls_auto: charges && vehicle?.toll_detection_active !== false,
    write: role === 'OWNER' || role === 'EDITOR',
  }
}
