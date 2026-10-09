// Offline mutation queue persisted in IndexedDB.
// Entries are replayed in order with their Idempotency-Key, so a request whose response was lost is applied once.

export interface QueuedMutation {
  id: string
  // Optional only for entries created before account-scoped queues were introduced.
  accountId?: string
  method: string
  endpoint: string
  body?: string
  label: string
  createdAt: number
}

const DB_NAME = 'teslacost-offline'
const STORE = 'mutations'

function openDb(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, 1)
    req.onupgradeneeded = () => {
      req.result.createObjectStore(STORE, { keyPath: 'id' })
    }
    req.onsuccess = () => resolve(req.result)
    req.onerror = () => reject(req.error instanceof Error ? req.error : new Error(String(req.error || 'IndexedDB request failed')))
  })
}

async function withStore<T>(mode: IDBTransactionMode, fn: (store: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  const db = await openDb()
  try {
    return await new Promise<T>((resolve, reject) => {
      const tx = db.transaction(STORE, mode)
      const req = fn(tx.objectStore(STORE))
      tx.oncomplete = () => resolve(req.result)
      tx.onerror = () => reject(tx.error instanceof Error ? tx.error : new Error(String(tx.error || 'IndexedDB transaction failed')))
      tx.onabort = () => reject(tx.error instanceof Error ? tx.error : new Error(String(tx.error || 'IndexedDB transaction aborted')))
    })
  } finally {
    db.close()
  }
}

// crypto.randomUUID is only available in secure contexts (self-hosted instances are often served over plain HTTP)
export function newIdempotencyKey(): string {
  const c = typeof window !== 'undefined' ? (window.crypto || (window as any).msCrypto) : (typeof crypto !== 'undefined' ? crypto : null)
  if (c && typeof c.randomUUID === 'function') {
    return c.randomUUID()
  }
  const bytes = new Uint8Array(16)
  if (c && typeof c.getRandomValues === 'function') {
    c.getRandomValues(bytes)
  }
  bytes[6] = (bytes[6] & 0x0f) | 0x40
  bytes[8] = (bytes[8] & 0x3f) | 0x80
  const hex = [...bytes].map((b) => b.toString(16).padStart(2, '0')).join('')
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`
}

export async function enqueueMutation(mutation: QueuedMutation): Promise<void> {
  await withStore('readwrite', (store) => store.put(mutation))
}

export async function listQueuedMutations(): Promise<QueuedMutation[]> {
  const all = await withStore<QueuedMutation[]>('readonly', (store) => store.getAll())
  return all.sort((a, b) => a.createdAt - b.createdAt)
}

export async function removeQueuedMutation(id: string): Promise<void> {
  await withStore('readwrite', (store) => store.delete(id))
}
