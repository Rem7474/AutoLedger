/** What the monthly cost chart does with a month far above the others, which would flatten every other bar. */
export interface ScaleCut {
  /** Top of the value axis: a little above the second highest month. */
  ceiling: number
  /** Indexes of the months that rise above it; their bars are cut at the top. */
  cutIndexes: number[]
}

const OUTLIER_RATIO = 2.5
const HEADROOM = 1.25

/** A cut scale when the highest month is more than 2.5 times the next one, otherwise null (the chart scales to the data). */
export function scaleCut(totals: number[]): ScaleCut | null {
  const sorted = totals.filter((v) => v > 0).sort((a, b) => b - a)
  if (sorted.length < 3 || sorted[0] <= sorted[1] * OUTLIER_RATIO) return null
  const ceiling = sorted[1] * HEADROOM
  return { ceiling, cutIndexes: totals.flatMap((v, i) => (v > ceiling ? [i] : [])) }
}
