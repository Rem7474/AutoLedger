// While a layer holds a history entry (see useEscapeToClose), a navigation to another page would leave that entry
// behind it: Back from the new page would land on a copy of the page just left. It replaces the entry instead.
export function replaceOverHeldEntry(held: boolean, to: { path: string; fullPath: string }, from: { path: string }, alreadyReplaced: string | null): boolean {
  return held && to.path !== from.path && to.fullPath !== alreadyReplaced
}
