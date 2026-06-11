/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

const DEFAULT_AUTH_REDIRECT = '/dashboard'

const DISALLOWED_AUTH_REDIRECTS = new Set([
  '/',
  '/404',
  '/sign-in',
  '/sign-up',
  '/register',
  '/forgot-password',
  '/reset',
  '/oauth',
  '/otp',
])

const LEGACY_AUTH_REDIRECTS: Array<[RegExp, string]> = [
  [/^\/console(?:\/.*)?$/, '/dashboard'],
  [/^\/token(?:\/.*)?$/, '/keys'],
  [/^\/channel(?:\/.*)?$/, '/channels'],
  [/^\/redemption(?:\/.*)?$/, '/redemption-codes'],
  [/^\/topup(?:\/.*)?$/, '/wallet'],
  [/^\/user(?:\/.*)?$/, '/users'],
  [/^\/log(?:\/.*)?$/, '/usage-logs'],
  [/^\/setting(?:\/.*)?$/, '/system-settings'],
]

const ALLOWED_AUTH_REDIRECT_PREFIXES = [
  '/dashboard',
  '/keys',
  '/channels',
  '/models',
  '/playground',
  '/profile',
  '/redemption-codes',
  '/subscriptions',
  '/system-settings',
  '/usage-logs',
  '/users',
  '/wallet',
  '/chat2link',
  '/chat',
]

function toSameOriginPath(rawRedirect: string): string | null {
  const value = rawRedirect.trim()
  if (!value) return null

  try {
    const url = new URL(value, window.location.origin)
    if (url.origin !== window.location.origin) return null
    return `${url.pathname}${url.search}${url.hash}`
  } catch {
    return null
  }
}

function stripSuffix(path: string): string {
  if (path.length <= 1) return path
  return path.replace(/\/+$/, '')
}

export function normalizeAuthRedirect(redirectTo?: string | null): string {
  const sameOriginPath = redirectTo ? toSameOriginPath(redirectTo) : null
  if (!sameOriginPath) return DEFAULT_AUTH_REDIRECT

  const path = stripSuffix(sameOriginPath.split(/[?#]/, 1)[0] || '/')
  const legacyMatch = LEGACY_AUTH_REDIRECTS.find(([pattern]) =>
    pattern.test(path)
  )
  if (legacyMatch) return legacyMatch[1]

  if (DISALLOWED_AUTH_REDIRECTS.has(path)) {
    return DEFAULT_AUTH_REDIRECT
  }

  const isAllowed = ALLOWED_AUTH_REDIRECT_PREFIXES.some(
    (prefix) => path === prefix || path.startsWith(`${prefix}/`)
  )

  return isAllowed ? sameOriginPath : DEFAULT_AUTH_REDIRECT
}
