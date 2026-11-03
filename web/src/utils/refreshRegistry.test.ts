import { describe, expect, it } from 'vitest'
import { currentRefresh, registerRefresh } from './refreshRegistry'

describe('refresh registry', () => {
  it('has nothing to refresh until a page registers', () => {
    expect(currentRefresh()).toBeNull()
  })

  it('serves the page that registered, and forgets it once it withdraws', () => {
    const refresh = () => 'reloaded'
    const withdraw = registerRefresh(refresh)
    expect(currentRefresh()).toBe(refresh)
    withdraw()
    expect(currentRefresh()).toBeNull()
  })

  it('keeps the next page when the previous one withdraws after it registered', () => {
    // Moving between pages mounts the new one before the old one unmounts.
    const withdrawOld = registerRefresh(() => 'old')
    const next = () => 'new'
    const withdrawNext = registerRefresh(next)
    withdrawOld()
    expect(currentRefresh()).toBe(next)
    withdrawNext()
    expect(currentRefresh()).toBeNull()
  })

  it('gives the pull back to the page when a panel inside it goes', () => {
    const page = () => 'page'
    const panel = () => 'panel'
    const withdrawPage = registerRefresh(page)
    const withdrawPanel = registerRefresh(panel)
    expect(currentRefresh()).toBe(panel)
    withdrawPanel()
    expect(currentRefresh()).toBe(page)
    withdrawPage()
    // Withdrawing twice changes nothing
    withdrawPage()
    expect(currentRefresh()).toBeNull()
  })
})
