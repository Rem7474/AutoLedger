export type RotationMode = 'FRONT_BACK' | 'CROSS'

// The wheels each mode swaps (same pairs as the server, `QuickRotateTires`).
const SWAPS: Record<RotationMode, Record<string, string>> = {
  FRONT_BACK: { FL: 'RL', RL: 'FL', FR: 'RR', RR: 'FR' },
  CROSS: { FL: 'RR', RR: 'FL', FR: 'RL', RL: 'FR' },
}

interface TireEntry {
  tire: { current_position: string }
}

/** The tire list as it reads once `mode` is applied: each mounted tire takes the position its mode sends it to. */
export function rotatedTires<T extends TireEntry>(list: T[], mode: RotationMode): T[] {
  const swaps = SWAPS[mode]
  return list.map((entry) => {
    const to = swaps[entry.tire.current_position]
    return to ? { ...entry, tire: { ...entry.tire, current_position: to } } : entry
  })
}
