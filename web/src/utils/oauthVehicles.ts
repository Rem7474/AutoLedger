/** A vehicle the consent page offers: the account's own vehicles, with the role the user holds on each. */
export interface GrantVehicleChoice {
  id: string
  name: string
  role: string
}

export type GrantVehicleMode = 'all' | 'some'

/**
 * The list sent with the decision: null grants every vehicle of the account, including the ones added later; otherwise
 * the chosen ones, in the order the page lists them.
 */
export function grantedVehicleIds(mode: GrantVehicleMode, selected: ReadonlySet<string>, choices: readonly GrantVehicleChoice[]): string[] | null {
  if (mode === 'all') return null
  return choices.filter((c) => selected.has(c.id)).map((c) => c.id)
}

/** Approving needs at least one vehicle when the user chose to limit the access. */
export function canApproveGrant(mode: GrantVehicleMode, selected: ReadonlySet<string>, choices: readonly GrantVehicleChoice[]): boolean {
  return grantedVehicleIds(mode, selected, choices)?.length !== 0
}

/** The selection after a vehicle is ticked or unticked; the previous set is left untouched. */
export function toggleVehicle(selected: ReadonlySet<string>, id: string): Set<string> {
  const next = new Set(selected)
  if (!next.delete(id)) next.add(id)
  return next
}

/**
 * What the token list shows for a limited grant: the names of the vehicles it covers, and how many of its vehicles the
 * account no longer shows (deleted, or no longer shared).
 */
export function coveredVehicles(ids: readonly string[], vehicles: readonly { id: string; name: string }[]): { names: string[]; missing: number } {
  const byId = new Map(vehicles.map((v) => [v.id, v.name]))
  const names: string[] = []
  let missing = 0
  for (const id of ids) {
    const name = byId.get(id)
    if (name === undefined) missing++
    else names.push(name)
  }
  return { names, missing }
}
