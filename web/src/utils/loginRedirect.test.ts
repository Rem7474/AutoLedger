import { beforeEach, describe, expect, it, vi } from 'vitest'
import { isResumablePath, rememberLoginRedirect, takeLoginRedirect } from './loginRedirect'

function fakeStorage() {
  const data = new Map<string, string>()
  return {
    getItem: (key: string) => data.get(key) ?? null,
    setItem: (key: string, value: string) => void data.set(key, value),
    removeItem: (key: string) => void data.delete(key),
  }
}

describe('loginRedirect', () => {
  beforeEach(() => vi.stubGlobal('sessionStorage', fakeStorage()))

  it('gives back the remembered path once', () => {
    rememberLoginRedirect('/oauth/authorize?client_id=abc&state=x%2Fy', 1000)
    expect(takeLoginRedirect(2000)).toBe('/oauth/authorize?client_id=abc&state=x%2Fy')
    expect(takeLoginRedirect(2000)).toBeNull()
  })

  it('forgets a path after ten minutes', () => {
    rememberLoginRedirect('/oauth/authorize?client_id=abc', 0)
    expect(takeLoginRedirect(10 * 60 * 1000 + 1)).toBeNull()
  })

  it('keeps only paths of this application', () => {
    for (const path of ['https://evil.example/', '//evil.example/x', '/\\evil.example', 'javascript:alert(1)', '', '/login', '/register?x=1', '/onboarding']) {
      expect(isResumablePath(path), path).toBe(false)
      rememberLoginRedirect(path)
      expect(takeLoginRedirect(), path).toBeNull()
    }
    expect(isResumablePath('/oauth/authorize?a=b')).toBe(true)
  })

  it('refuses a stored value somebody else wrote', () => {
    sessionStorage.setItem('al-login-resume', JSON.stringify({ path: '//evil.example', at: Date.now() }))
    expect(takeLoginRedirect()).toBeNull()
    sessionStorage.setItem('al-login-resume', 'not json')
    expect(takeLoginRedirect()).toBeNull()
  })
})
