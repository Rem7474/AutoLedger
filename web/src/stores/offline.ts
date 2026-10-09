import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { t } from '@/i18n'
import { enqueueMutation, listQueuedMutations, removeQueuedMutation, type QueuedMutation } from '@/services/offlineQueue'
import { useVehicleStore } from '@/stores/vehicle'
import { useAuthStore } from '@/stores/auth'
import { fetchWithSessionRefresh } from '@/services/sessionFetch'

const RETRY_INTERVAL_MS = 30_000

export const useOfflineStore = defineStore('offline', () => {
  const auth = useAuthStore()
  const isOnline = ref(typeof navigator === 'undefined' ? true : navigator.onLine)
  const pendingCount = ref(0)
  const isFlushing = ref(false)
  const lastQueuedLabel = ref<string | null>(null)
  const failures = ref<{ label: string; error: string }[]>([])
  let started = false

  async function refreshCount() {
    try {
      pendingCount.value = (await listQueuedMutations()).filter((m) => m.accountId === auth.user?.id && !!m.accountId).length
    } catch (err) {
      console.error('Failed to read the offline queue', err)
      pendingCount.value = 0
    }
  }

  async function queue(mutation: QueuedMutation) {
    const accountId = mutation.accountId || auth.user?.id
    if (!accountId) throw new Error(t('shell.offline.signInRequired'))
    await enqueueMutation({ ...mutation, accountId })
    lastQueuedLabel.value = mutation.label
    setTimeout(() => {
      if (lastQueuedLabel.value === mutation.label) lastQueuedLabel.value = null
    }, 4000)
    await refreshCount()
  }

  // Replays queued mutations in order; stops at the first network or server error to keep ordering
  async function flush() {
    const accountId = auth.user?.id
    if (isFlushing.value || !navigator.onLine || !accountId) return
    isFlushing.value = true
    let sent = 0
    try {
      const entries = await listQueuedMutations()
      if (entries.some((m) => !m.accountId) && !failures.value.some((f) => f.label === t('shell.offline.legacyLabel'))) {
        failures.value.push({ label: t('shell.offline.legacyLabel'), error: t('shell.offline.legacyRetained') })
      }
      for (const m of entries) {
        if (auth.user?.id !== accountId) break
        if (m.accountId !== accountId) continue
        let res: Response
        try {
          res = await fetchWithSessionRefresh(`/api${m.endpoint}`, {
            method: m.method,
            credentials: 'include',
            headers: {
              'Content-Type': 'application/json',
              'Idempotency-Key': m.id,
            },
            body: m.body,
          }, () => auth.user?.id === accountId)
        } catch {
          break
        }
        if (auth.user?.id !== accountId || res.status === 401 || res.status >= 500) break
        if (!res.ok) {
          const data = await res.json().catch(() => ({}))
          const failure = { label: m.label, error: data.error || t('shell.offline.httpError', { status: res.status }) }
          if (!failures.value.some((f) => f.label === failure.label && f.error === failure.error)) failures.value.push(failure)
          // An access change is recoverable; never discard the user's record for it.
          if (res.status === 403 || res.status === 404) break
        } else {
          sent++
        }
        await removeQueuedMutation(m.id)
      }
    } finally {
      isFlushing.value = false
      await refreshCount()
      if (sent > 0) {
        useVehicleStore().lastSyncTimestamp = Date.now()
      }
    }
  }

  // Background replay: an IndexedDB failure must not surface as an unhandled rejection
  function flushInBackground() {
    flush().catch((err) => console.error('Failed to replay the offline queue', err))
  }

  function start() {
    if (started) return
    started = true
    window.addEventListener('online', () => {
      isOnline.value = true
      flushInBackground()
    })
    window.addEventListener('offline', () => {
      isOnline.value = false
    })
    setInterval(() => {
      if (pendingCount.value > 0) flushInBackground()
    }, RETRY_INTERVAL_MS)
    void refreshCount().then(flushInBackground)
  }

  watch(() => auth.user?.id, () => {
    lastQueuedLabel.value = null
    failures.value = []
    void refreshCount()
  })

  function dismissFailures() {
    failures.value = []
  }

  return { isOnline, pendingCount, isFlushing, lastQueuedLabel, failures, queue, flush, start, dismissFailures }
})
