// The page on screen says how it reloads its data; the pull of the shell asks for it when the finger is released.

export type Refresh = () => unknown | Promise<unknown>

// Newest last: a panel inside a page (the fill-ups of the expenses page) takes over while it is shown, and the
// page's own reload comes back when the panel goes.
const registered: Refresh[] = []

/** Makes `refresh` what a pull down reloads; the returned function withdraws it. */
export function registerRefresh(refresh: Refresh): () => void {
  registered.push(refresh)
  return () => {
    const index = registered.lastIndexOf(refresh)
    if (index !== -1) registered.splice(index, 1)
  }
}

export function currentRefresh(): Refresh | null {
  return registered[registered.length - 1] ?? null
}
