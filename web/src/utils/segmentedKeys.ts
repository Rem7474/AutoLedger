// Index of the segment a key moves the selection to, or -1 when the key does nothing. Arrows wrap around, Home and End
// jump to the ends (the keyboard pattern of a radiogroup).
export function nextSegment(key: string, current: number, count: number): number {
  if (count <= 0) return -1
  switch (key) {
    case 'ArrowRight':
    case 'ArrowDown':
      return (current + 1) % count
    case 'ArrowLeft':
    case 'ArrowUp':
      return (current - 1 + count) % count
    case 'Home':
      return 0
    case 'End':
      return count - 1
    default:
      return -1
  }
}
