export type CursorStyle = 'default' | 'banana'

export const CURSOR_STORAGE_KEY = 'newapi:cursor:v1:style'

const CURSOR_STYLES = new Set<CursorStyle>(['default', 'banana'])

export function readCursorPreference(): CursorStyle {
  if (typeof window === 'undefined') return 'default'

  try {
    const value = window.localStorage.getItem(CURSOR_STORAGE_KEY)
    return value && CURSOR_STYLES.has(value as CursorStyle)
      ? (value as CursorStyle)
      : 'default'
  } catch {
    return 'default'
  }
}

export function writeCursorPreference(style: CursorStyle): void {
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(CURSOR_STORAGE_KEY, style)
  } catch {
    // Keep the toggle usable when browser storage is blocked or full.
  }
}