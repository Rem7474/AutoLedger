// A page that cannot be shown to a signed-out visitor but must be reached once they sign in (the consent page an
// assistant sends the user to) is remembered for the length of the sign-in, here and not in the address, so the
// sign-in views stay as they are and a single sign-on round trip keeps it.
const KEY = 'al-login-resume'
const MAX_AGE_MS = 10 * 60 * 1000

/** Only a path of this application: never an address another site could use to take the user elsewhere. */
export function isResumablePath(path: string): boolean {
  return /^\/(?![/\\])/.test(path) && !/^\/(login|register|onboarding)(\/|\?|$)/.test(path)
}

export function rememberLoginRedirect(path: string, now = Date.now()): void {
  if (!isResumablePath(path)) return
  try {
    sessionStorage.setItem(KEY, JSON.stringify({ path, at: now }))
  } catch {
    // storage blocked: the user lands on the dashboard and starts again from the assistant
  }
}

/** The remembered path, once: null when none was kept, it is too old, or it is not a path of this application. */
export function takeLoginRedirect(now = Date.now()): string | null {
  try {
    const raw = sessionStorage.getItem(KEY)
    if (!raw) return null
    sessionStorage.removeItem(KEY)
    const { path, at } = JSON.parse(raw) as { path?: unknown; at?: unknown }
    if (typeof path !== 'string' || typeof at !== 'number' || now - at > MAX_AGE_MS) return null
    return isResumablePath(path) ? path : null
  } catch {
    return null
  }
}
