import { afterEach, describe, expect, it } from 'vitest'
import { currentRefresh, registerRefresh } from './refreshRegistry'

describe('refresh registry', () => {
  afterEach(() => {
    // Whatever a test left registered
    registerRefresh(() => {})()
  })

  it('has nothing to refresh until a page registers', () => {
    expect(currentRefresh()).toBeNull()
  })

  it('serves the page that registered', () => {
    const refresh = () => 'reloaded'
    registerRefresh(refresh)
    expect(currentRefresh()).toBe(refresh)
  })

  it('forgets the page once it withdraws', () => {
    const withdraw = registerRefresh(() => {})
    withdraw()
    expect(currentRefresh()).toBeNull()
  })

  it('keeps the next page when the previous one withdraws after it registered', () => {
    // Moving between pages mounts the new one before the old one unmounts.
    const withdrawOld = registerRefresh(() => 'old')
    const next = () => 'new'
    registerRefresh(next)
    withdrawOld()
    expect(currentRefresh()).toBe(next)
  })
})
