import { describe, expect, it } from 'vitest'
import { driveSteps, findDriveChain, groupDrivesByDay, recapFigures, splitShortDrives, stopFromRatio, tapStop, withAlight, withBoard } from './carpoolEditor'

describe('stops of a passenger', () => {
  it('turns a position along the route into a stop', () => {
    expect(stopFromRatio(0, 3)).toBe(0)
    expect(stopFromRatio(0.4, 3)).toBe(1)
    expect(stopFromRatio(0.9, 3)).toBe(3)
    expect(stopFromRatio(1.5, 3)).toBe(3)
    expect(stopFromRatio(-1, 3)).toBe(0)
  })

  it('keeps at least one leg when a handle moves', () => {
    expect(withBoard({ board: 0, alight: 2 }, 5)).toEqual({ board: 1, alight: 2 })
    expect(withBoard({ board: 1, alight: 2 }, -3)).toEqual({ board: 0, alight: 2 })
    expect(withAlight({ board: 1, alight: 3 }, 0, 3)).toEqual({ board: 1, alight: 2 })
    expect(withAlight({ board: 0, alight: 1 }, 9, 3)).toEqual({ board: 0, alight: 3 })
  })

  it('extends the ride on a tap outside it and moves the nearer end inside it', () => {
    expect(tapStop({ board: 1, alight: 2 }, 0, 3)).toEqual({ board: 0, alight: 2 })
    expect(tapStop({ board: 0, alight: 1 }, 3, 3)).toEqual({ board: 0, alight: 3 })
    expect(tapStop({ board: 0, alight: 4 }, 1, 4)).toEqual({ board: 1, alight: 4 })
    expect(tapStop({ board: 0, alight: 4 }, 3, 4)).toEqual({ board: 0, alight: 3 })
    expect(tapStop({ board: 0, alight: 4 }, 2, 4)).toEqual({ board: 0, alight: 2 })
  })
})

describe('recap figures', () => {
  it('splits the cost and measures what was received', () => {
    const r = recapFigures({ total: 4000, passengersShare: 1600, driverShare: 2400 }, 3000)
    expect(r.passengersPct).toBe(40)
    expect(r.driverPct).toBe(60)
    expect(r.receivedPct).toBe(75)
    expect(r.left).toBe(1000)
  })

  it('has no percentage while nothing costs anything', () => {
    expect(recapFigures({ total: 0, passengersShare: 0, driverShare: 0 }, 0).receivedPct).toBe(0)
  })
})

describe('drive picker', () => {
  const d = (id: string, start: string, end: string, from: string, to: string, km = 50, extra = {}) => ({
    id, start_time: start, end_time: end, start_address: from, end_address: to, distance_km: km, ...extra,
  })

  it('finds the most recent chain of drives that start where the previous ended', () => {
    const drives = [
      d('a', '2026-10-18T18:00:00Z', '2026-10-18T18:50:00Z', 'Annecy, FR', 'Grain, FR'),
      d('b', '2026-10-18T19:00:00Z', '2026-10-18T19:50:00Z', 'Grain, FR', 'Valence, FR'),
      d('c', '2026-10-19T09:00:00Z', '2026-10-19T10:00:00Z', 'Valence, FR', 'Annecy, FR'),
      d('x', '2026-10-10T09:00:00Z', '2026-10-10T10:00:00Z', 'Paris', 'Lyon'),
    ]
    expect(findDriveChain(drives).map((x) => x.id)).toEqual(['a', 'b'])
  })

  it('does not chain drives hours apart or in different places', () => {
    const drives = [
      d('a', '2026-10-18T08:00:00Z', '2026-10-18T09:00:00Z', 'A', 'B'),
      d('b', '2026-10-18T20:00:00Z', '2026-10-18T21:00:00Z', 'B', 'C'),
      d('c', '2026-10-18T21:10:00Z', '2026-10-18T22:00:00Z', 'Z', 'D'),
    ]
    expect(findDriveChain(drives)).toEqual([])
    expect(findDriveChain([])).toEqual([])
  })

  it('hides drives under a kilometre unless selected or already in a carpool', () => {
    const drives = [
      d('a', '2026-10-18T08:00:00Z', '', 'A', 'B', 0.4),
      d('b', '2026-10-18T09:00:00Z', '', 'A', 'B', 0.4),
      d('c', '2026-10-18T10:00:00Z', '', 'A', 'B', 0.4, { carpools: [{ id: 'x' }] }),
      d('e', '2026-10-18T11:00:00Z', '', 'A', 'B', 30),
    ]
    const { shown, hidden } = splitShortDrives(drives, ['b'])
    expect(shown.map((x) => x.id)).toEqual(['b', 'c', 'e'])
    expect(hidden.map((x) => x.id)).toEqual(['a'])
  })

  it('groups the drives by day, oldest first', () => {
    const groups = groupDrivesByDay([
      d('b', '2026-10-19T09:00:00', '', 'A', 'B'),
      d('a', '2026-10-18T19:00:00', '', 'A', 'B'),
      d('c', '2026-10-18T08:00:00', '', 'A', 'B'),
    ])
    expect(groups.map((g) => g.day)).toEqual(['2026-10-18', '2026-10-19'])
    expect(groups[0].drives.map((x) => x.id)).toEqual(['c', 'a'])
  })
})

describe('leg numbers of the selected drives', () => {
  it('numbers them in driving order, whatever the order of the list', () => {
    const steps = driveSteps([
      { id: 'b', start_time: '2026-10-19T09:00:00Z' },
      { id: 'a', start_time: '2026-10-18T19:00:00Z' },
    ])
    expect(steps.get('a')).toBe(1)
    expect(steps.get('b')).toBe(2)
    expect(steps.has('c')).toBe(false)
  })
})
