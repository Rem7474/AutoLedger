// The devices signed in to the account: this one first, then the most recently used, cut to `limit` unless the list is
// expanded. An account that never signs out of anything can hold dozens of them.
export const COLLAPSED_SESSIONS = 3

interface ListedSession {
  current: boolean
  last_used_at: string
}

export function visibleSessions<T extends ListedSession>(sessions: T[], expanded: boolean, limit = COLLAPSED_SESSIONS): T[] {
  const ordered = [...sessions].sort((a, b) => Number(b.current) - Number(a.current) || Date.parse(b.last_used_at) - Date.parse(a.last_used_at))
  return expanded ? ordered : ordered.slice(0, limit)
}
