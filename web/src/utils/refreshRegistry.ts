// The page on screen says how it reloads its data; the pull of the shell asks for it when the finger is released.

export type Refresh = () => unknown | Promise<unknown>

let current: Refresh | null = null

/** Makes `refresh` what a pull down reloads; the returned function withdraws it unless another page took over since. */
export function registerRefresh(refresh: Refresh): () => void {
  current = refresh
  return () => {
    if (current === refresh) current = null
  }
}

export function currentRefresh(): Refresh | null {
  return current
}
