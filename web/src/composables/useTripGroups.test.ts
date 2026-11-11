import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '@/services/api'
import { useTripGroups } from './useTripGroups'
import { flushPendingDeletes, UNDO_MS } from './useUndoableDelete'

vi.mock('@/services/api', () => ({
  api: {
    getTripGroups: vi.fn(),
    getTripSuggestions: vi.fn(),
    dismissTripSuggestion: vi.fn(),
    createTripGroup: vi.fn(),
    getDrives: vi.fn(),
    updateTripGroup: vi.fn(),
    deleteTripGroup: vi.fn(),
  },
}))

vi.mock('@/stores/vehicle', () => ({
  useVehicleStore: () => ({ activeVehicle: { id: 'v1' }, currency: 'EUR' }),
}))

const alert = vi.fn()
const choice = vi.fn()
vi.mock('@/composables/useConfirm', () => ({
  useConfirm: () => ({ showChoice: choice, showAlert: alert }),
}))

const suggestion = { drive_ids: ['d1', 'd2'], start_time: '2026-01-01T08:00:00Z', end_time: '2026-01-02T08:00:00Z', start_address: 'Paris', end_address: 'Lyon' }

describe('useTripGroups', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the trips and the suggestions, and tolerates failing suggestions', async () => {
    vi.mocked(api.getTripGroups).mockResolvedValue([{ id: 't1' }] as any)
    vi.mocked(api.getTripSuggestions).mockRejectedValue(new Error('nope'))
    const trips = useTripGroups({ loadDrives: vi.fn() })
    await trips.loadTripGroups()
    expect(trips.tripGroups.value).toEqual([{ id: 't1' }])
    expect(trips.tripSuggestions.value).toEqual([])
    expect(trips.loadingTrips.value).toBe(false)
    expect(trips.tripsLoadError.value).toBeNull()
  })

  it('exposes the load error message', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => {})
    vi.mocked(api.getTripGroups).mockRejectedValue(new Error('boom'))
    vi.mocked(api.getTripSuggestions).mockResolvedValue([] as any)
    const trips = useTripGroups({ loadDrives: vi.fn() })
    await trips.loadTripGroups()
    expect(trips.tripsLoadError.value).toBe('boom')
  })

  it('drops a dismissed suggestion and releases the busy key', async () => {
    vi.mocked(api.dismissTripSuggestion).mockResolvedValue(undefined as any)
    const trips = useTripGroups({ loadDrives: vi.fn() })
    trips.tripSuggestions.value = [suggestion, { ...suggestion, drive_ids: ['d9'] }]
    await trips.dismissTripSuggestion(suggestion)
    expect(api.dismissTripSuggestion).toHaveBeenCalledWith('v1', ['d1', 'd2'])
    expect(trips.tripSuggestions.value.map((s) => s.drive_ids[0])).toEqual(['d9'])
    expect(trips.suggestionBusyKey.value).toBeNull()
  })

  it('names a suggestion after its route and creates the trip', async () => {
    vi.mocked(api.createTripGroup).mockResolvedValue({} as any)
    vi.mocked(api.getTripGroups).mockResolvedValue([] as any)
    vi.mocked(api.getTripSuggestions).mockResolvedValue([] as any)
    const trips = useTripGroups({ loadDrives: vi.fn() })
    expect(trips.suggestionName(suggestion)).toBe('Paris → Lyon')
    await trips.createTripFromSuggestion(suggestion)
    expect(api.createTripGroup).toHaveBeenCalledWith('v1', { name: 'Paris → Lyon', drive_ids: ['d1', 'd2'] })
    expect(api.getTripGroups).toHaveBeenCalled()
  })

  it('shows the legs of a trip sorted by start, and collapses on a second toggle', async () => {
    vi.mocked(api.getDrives).mockResolvedValue({ drives: [{ id: 'b', start_time: '2026-01-02' }, { id: 'a', start_time: '2026-01-01' }] } as any)
    const trips = useTripGroups({ loadDrives: vi.fn() })
    await trips.toggleTripDetails({ id: 't1' })
    expect(trips.expandedTripId.value).toBe('t1')
    expect(trips.tripDrives.value.map((d) => d.id)).toEqual(['a', 'b'])
    await trips.toggleTripDetails({ id: 't1' })
    expect(trips.expandedTripId.value).toBeNull()
  })

  it('refuses to remove the last drive of a trip', async () => {
    const trips = useTripGroups({ loadDrives: vi.fn() })
    await trips.removeDriveFromTrip({ id: 't1', drive_ids: ['d1'] }, 'd1')
    expect(alert).toHaveBeenCalledTimes(1)
  })

  describe('removeDriveFromTrip with several legs', () => {
    const tg = { id: 't1', name: 'Trip', notes: '', version: 3, drive_ids: ['a', 'b'] }

    it('keeps the drawer open on the reloaded legs when it was open for that trip', async () => {
      vi.mocked(api.updateTripGroup).mockResolvedValue({} as any)
      vi.mocked(api.getTripGroups).mockResolvedValue([{ ...tg, drive_ids: ['b'] }] as any)
      vi.mocked(api.getTripSuggestions).mockResolvedValue([] as any)
      vi.mocked(api.getDrives).mockResolvedValue({ drives: [{ id: 'b', start_time: '2026-01-02' }] } as any)
      const loadDrives = vi.fn()
      const trips = useTripGroups({ loadDrives })
      trips.expandedTripId.value = 't1'
      await trips.removeDriveFromTrip(tg, 'a')
      expect(api.updateTripGroup).toHaveBeenCalledWith('v1', 't1', { name: 'Trip', notes: '', drive_ids: ['b'], expected_version: 3 })
      expect(trips.expandedTripId.value).toBe('t1')
      expect(trips.tripDrives.value.map((d) => d.id)).toEqual(['b'])
      expect(loadDrives).toHaveBeenCalled()
    })

    it('leaves the drawer closed when it was not open for that trip', async () => {
      vi.mocked(api.updateTripGroup).mockResolvedValue({} as any)
      vi.mocked(api.getTripGroups).mockResolvedValue([{ ...tg, drive_ids: ['b'] }] as any)
      vi.mocked(api.getTripSuggestions).mockResolvedValue([] as any)
      const trips = useTripGroups({ loadDrives: vi.fn() })
      await trips.removeDriveFromTrip(tg, 'a')
      expect(trips.expandedTripId.value).toBeNull()
      expect(api.getDrives).not.toHaveBeenCalled()
    })

    it('closes the drawer when the trip is gone after the reload', async () => {
      vi.mocked(api.updateTripGroup).mockResolvedValue({} as any)
      vi.mocked(api.getTripGroups).mockResolvedValue([] as any)
      vi.mocked(api.getTripSuggestions).mockResolvedValue([] as any)
      const trips = useTripGroups({ loadDrives: vi.fn() })
      trips.expandedTripId.value = 't1'
      await trips.removeDriveFromTrip(tg, 'a')
      expect(trips.expandedTripId.value).toBeNull()
      expect(api.getDrives).not.toHaveBeenCalled()
    })
  })

  describe('handleDeleteTrip', () => {
    async function listed(expenseCount: number) {
      const trip = { id: 't1', name: 'Alps', expense_count: expenseCount, expenses_total: '12.5' }
      vi.mocked(api.getTripGroups).mockResolvedValue([trip] as any)
      vi.mocked(api.getTripSuggestions).mockResolvedValue([] as any)
      const loadDrives = vi.fn()
      const trips = useTripGroups({ loadDrives })
      await trips.loadTripGroups()
      return { trips, trip, loadDrives }
    }

    it('hides a trip without costs at once and deletes it when the time to undo is over', async () => {
      vi.useFakeTimers()
      try {
        const { trips, trip, loadDrives } = await listed(0)
        vi.mocked(api.deleteTripGroup).mockResolvedValue({} as any)
        vi.mocked(api.getTripGroups).mockResolvedValue([] as any)
        await trips.handleDeleteTrip(trip)
        expect(choice).not.toHaveBeenCalled()
        expect(trips.tripGroups.value).toEqual([])
        expect(api.deleteTripGroup).not.toHaveBeenCalled()
        await vi.advanceTimersByTimeAsync(UNDO_MS)
        expect(api.deleteTripGroup).toHaveBeenCalledWith('v1', 't1', false)
        expect(loadDrives).toHaveBeenCalled()
      } finally {
        await flushPendingDeletes()
        vi.useRealTimers()
      }
    })

    it('asks what to do with the costs first: delete them too, keep them unlinked, or keep the trip', async () => {
      vi.useFakeTimers()
      try {
        const { trips, trip } = await listed(2)
        vi.mocked(api.deleteTripGroup).mockResolvedValue({} as any)
        choice.mockResolvedValueOnce('cancel')
        await trips.handleDeleteTrip(trip)
        expect(trips.tripGroups.value).toHaveLength(1)
        choice.mockResolvedValueOnce('confirm')
        await trips.handleDeleteTrip(trip)
        await vi.advanceTimersByTimeAsync(UNDO_MS)
        expect(api.deleteTripGroup).toHaveBeenLastCalledWith('v1', 't1', true)
        vi.mocked(api.getTripGroups).mockResolvedValue([trip] as any)
        await trips.loadTripGroups()
        choice.mockResolvedValueOnce('alternative')
        await trips.handleDeleteTrip(trip)
        await vi.advanceTimersByTimeAsync(UNDO_MS)
        expect(api.deleteTripGroup).toHaveBeenLastCalledWith('v1', 't1', false)
      } finally {
        await flushPendingDeletes()
        vi.useRealTimers()
      }
    })

    it('puts the trip back and alerts when the server refuses', async () => {
      vi.useFakeTimers()
      try {
        const { trips, trip } = await listed(0)
        vi.mocked(api.deleteTripGroup).mockRejectedValue(new Error('nope'))
        await trips.handleDeleteTrip(trip)
        await vi.advanceTimersByTimeAsync(UNDO_MS)
        expect(trips.tripGroups.value).toHaveLength(1)
        expect(alert).toHaveBeenCalled()
      } finally {
        await flushPendingDeletes()
        vi.useRealTimers()
      }
    })
  })
})
