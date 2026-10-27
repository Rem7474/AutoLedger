interface DetectedSegment {
  entry?: string
  exit?: string | null
  spans?: number
  full_price?: number | null
}

type SharedSegment<T> = T & { spans: number; full_price: number }

/** The tickets of a drive that run over several drives: its estimate holds only this drive's part of each. */
export function sharedTollSegments<T extends DetectedSegment>(segments: T[] | null | undefined): SharedSegment<T>[] {
  return (segments ?? []).filter((seg): seg is SharedSegment<T> => (seg.spans ?? 1) > 1 && seg.full_price != null)
}
