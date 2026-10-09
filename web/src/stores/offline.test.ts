import { afterEach, beforeEach, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

const mocks = vi.hoisted(() => ({ list: vi.fn(), remove: vi.fn(), enqueue: vi.fn(), auth: { user: { id: 'account-A' } as { id: string } | null } }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => mocks.auth }))
vi.mock('@/services/offlineQueue', () => ({ listQueuedMutations: mocks.list, removeQueuedMutation: mocks.remove, enqueueMutation: mocks.enqueue }))
vi.mock('@/stores/vehicle', () => ({ useVehicleStore: () => ({ lastSyncTimestamp: 0 }) }))
import { useOfflineStore } from './offline'

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  vi.stubGlobal('navigator', { onLine: true })
  mocks.auth.user = { id: 'account-A' }
  mocks.list.mockResolvedValue([{ id: 'queued-1', accountId: 'account-A', method: 'POST', endpoint: '/vehicles/v1/fuel', body: '{}', label: 'Fill-up', createdAt: 1 }])
})
afterEach(() => vi.unstubAllGlobals())

it('renews the session and removes a queued record only after a successful retry', async () => {
  const fetchMock = vi.fn()
    .mockResolvedValueOnce(new Response('{}', { status: 401 }))
    .mockResolvedValueOnce(new Response('{}', { status: 200 }))
    .mockResolvedValueOnce(new Response('{}', { status: 201 }))
  mocks.remove.mockImplementation(async () => mocks.list.mockResolvedValue([]))
  vi.stubGlobal('fetch', fetchMock)
  const store = useOfflineStore()
  await store.flush()
  expect(fetchMock.mock.calls.map(([url]) => url)).toEqual(['/api/vehicles/v1/fuel', '/api/auth/refresh', '/api/vehicles/v1/fuel'])
  expect(fetchMock.mock.calls[2]).toEqual(fetchMock.mock.calls[0])
  expect(mocks.remove).toHaveBeenCalledWith('queued-1')
  expect(store.pendingCount).toBe(0)
})

it('retains the queued record if the refresh session is no longer valid', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => new Response('{}', { status: 401 })))
  const store = useOfflineStore()
  await store.flush()
  expect(mocks.remove).not.toHaveBeenCalled()
  expect(store.pendingCount).toBe(1)
  expect(store.failures).toEqual([])
})


it('does not replay another account or legacy entries and resumes the owner queue', async () => {
  mocks.auth.user = { id: 'account-B' }
  const fetchMock = vi.fn(async () => new Response('{}', { status: 201 }))
  vi.stubGlobal('fetch', fetchMock)
  const store = useOfflineStore()
  await store.flush()
  expect(fetchMock).not.toHaveBeenCalled()
  expect(mocks.remove).not.toHaveBeenCalled()
  expect(store.pendingCount).toBe(0)
  mocks.auth.user = { id: 'account-A' }
  mocks.remove.mockImplementation(async () => mocks.list.mockResolvedValue([]))
  await store.flush()
  expect(fetchMock).toHaveBeenCalledTimes(1)
  expect(mocks.remove).toHaveBeenCalledWith('queued-1')
  mocks.list.mockResolvedValue([{ id: 'legacy', endpoint: '/vehicles/v1/fuel', label: 'Old entry' }])
  await store.flush()
  expect(fetchMock).toHaveBeenCalledTimes(1)
  expect(mocks.remove).not.toHaveBeenCalledWith('legacy')
  expect(store.failures).toHaveLength(1)
})

it.each([403, 404])('retains offline work on recoverable access response %s', async (status) => {
  vi.stubGlobal('fetch', vi.fn(async () => new Response('{"error":"Access denied"}', { status })))
  const store = useOfflineStore()
  await store.flush()
  expect(mocks.remove).not.toHaveBeenCalled()
  expect(store.pendingCount).toBe(1)
  expect(store.failures).toEqual([{ label: 'Fill-up', error: 'Access denied' }])
})

it('does not remove or continue replay when the account changes during a request', async () => {
  vi.stubGlobal('fetch', vi.fn(async () => {
    mocks.auth.user = { id: 'account-B' }
    return new Response('{}', { status: 201 })
  }))
  await useOfflineStore().flush()
  expect(mocks.remove).not.toHaveBeenCalled()
})

it('records the account when queueing and refuses an anonymous entry', async () => {
  vi.useFakeTimers()
  try {
    const store = useOfflineStore()
    const entry = { id: 'new', method: 'POST', endpoint: '/vehicles/v1/fuel', label: 'Fill-up', createdAt: 1 }
    await store.queue(entry)
    expect(mocks.enqueue).toHaveBeenCalledWith({ ...entry, accountId: 'account-A' })
    mocks.auth.user = null
    await expect(store.queue(entry)).rejects.toThrow()
    expect(mocks.enqueue).toHaveBeenCalledTimes(1)
  } finally { vi.useRealTimers() }
})
