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

export type NavItemKind = 'builtin' | 'custom'

export type HeaderNavItemConfig = {
  id: string
  label: string
  href: string
  enabled: boolean
  requireAuth?: boolean
  external?: boolean
  order: number
  kind: NavItemKind
}

export type HeaderNavModulesConfig = {
  items: HeaderNavItemConfig[]
}

export type SidebarItemConfig = {
  id: string
  label: string
  url?: string
  enabled: boolean
  external?: boolean
  icon?: string
  order: number
  kind: NavItemKind
}

export type SidebarSectionConfig = {
  id: string
  label: string
  enabled: boolean
  order: number
  kind: NavItemKind
  items: SidebarItemConfig[]
}

export type SidebarModulesAdminConfig = {
  sections: SidebarSectionConfig[]
}

export type SidebarVisibilityConfig = Record<
  string,
  { enabled: boolean; [moduleKey: string]: boolean }
>

const BUILTIN_HEADER_ITEMS: HeaderNavItemConfig[] = [
  {
    id: 'home',
    label: 'Home',
    href: '/',
    enabled: true,
    order: 0,
    kind: 'builtin',
  },
  {
    id: 'console',
    label: 'Console',
    href: '/dashboard',
    enabled: true,
    order: 1,
    kind: 'builtin',
  },
  {
    id: 'pricing',
    label: 'Model Square',
    href: '/pricing',
    enabled: true,
    requireAuth: false,
    order: 2,
    kind: 'builtin',
  },
  {
    id: 'rankings',
    label: 'Rankings',
    href: '/rankings',
    enabled: true,
    requireAuth: false,
    order: 3,
    kind: 'builtin',
  },
  {
    id: 'docs',
    label: 'Docs',
    href: '/docs',
    enabled: true,
    order: 4,
    kind: 'builtin',
  },
  {
    id: 'about',
    label: 'About',
    href: '/about',
    enabled: true,
    order: 5,
    kind: 'builtin',
  },
]

const BUILTIN_SIDEBAR_SECTIONS: SidebarSectionConfig[] = [
  {
    id: 'chat',
    label: 'Chat',
    enabled: true,
    order: 0,
    kind: 'builtin',
    items: [
      {
        id: 'playground',
        label: 'Playground',
        url: '/playground',
        enabled: true,
        order: 0,
        kind: 'builtin',
      },
      {
        id: 'chat',
        label: 'Chat',
        enabled: true,
        order: 1,
        kind: 'builtin',
      },
    ],
  },
  {
    id: 'console',
    label: 'General',
    enabled: true,
    order: 1,
    kind: 'builtin',
    items: [
      {
        id: 'overview',
        label: 'Overview',
        url: '/dashboard/overview',
        enabled: true,
        order: 0,
        kind: 'builtin',
      },
      {
        id: 'detail',
        label: 'Dashboard',
        url: '/dashboard/models',
        enabled: true,
        order: 1,
        kind: 'builtin',
      },
      {
        id: 'token',
        label: 'API Keys',
        url: '/keys',
        enabled: true,
        order: 2,
        kind: 'builtin',
      },
      {
        id: 'log',
        label: 'Usage Logs',
        url: '/usage-logs/common',
        enabled: true,
        order: 3,
        kind: 'builtin',
      },
      {
        id: 'midjourney',
        label: 'Drawing Logs',
        url: '/usage-logs/drawing',
        enabled: true,
        order: 4,
        kind: 'builtin',
      },
      {
        id: 'task',
        label: 'Task Logs',
        url: '/usage-logs/task',
        enabled: true,
        order: 5,
        kind: 'builtin',
      },
    ],
  },
  {
    id: 'personal',
    label: 'Personal',
    enabled: true,
    order: 2,
    kind: 'builtin',
    items: [
      {
        id: 'topup',
        label: 'Wallet',
        url: '/wallet',
        enabled: true,
        order: 0,
        kind: 'builtin',
      },
      {
        id: 'personal',
        label: 'Profile',
        url: '/profile',
        enabled: true,
        order: 1,
        kind: 'builtin',
      },
    ],
  },
  {
    id: 'admin',
    label: 'Admin',
    enabled: true,
    order: 3,
    kind: 'builtin',
    items: [
      {
        id: 'channel',
        label: 'Channels',
        url: '/channels',
        enabled: true,
        order: 0,
        kind: 'builtin',
      },
      {
        id: 'models',
        label: 'Models',
        url: '/models/metadata',
        enabled: true,
        order: 1,
        kind: 'builtin',
      },
      {
        id: 'user',
        label: 'Users',
        url: '/users',
        enabled: true,
        order: 2,
        kind: 'builtin',
      },
      {
        id: 'redemption',
        label: 'Redemption Codes',
        url: '/redemption-codes',
        enabled: true,
        order: 3,
        kind: 'builtin',
      },
      {
        id: 'subscription',
        label: 'Subscription Management',
        url: '/subscriptions',
        enabled: true,
        order: 4,
        kind: 'builtin',
      },
      {
        id: 'setting',
        label: 'System Settings',
        url: '/system-settings/site',
        enabled: true,
        order: 5,
        kind: 'builtin',
      },
    ],
  },
]

export const HEADER_NAV_DEFAULT: HeaderNavModulesConfig = {
  items: BUILTIN_HEADER_ITEMS.map((item) => ({ ...item })),
}

export const SIDEBAR_MODULES_DEFAULT: SidebarModulesAdminConfig = {
  sections: BUILTIN_SIDEBAR_SECTIONS.map((section) => ({
    ...section,
    items: section.items.map((item) => ({ ...item })),
  })),
}

const HEADER_BUILTIN_BY_ID = new Map(
  BUILTIN_HEADER_ITEMS.map((item) => [item.id, item])
)
const SIDEBAR_SECTION_BY_ID = new Map(
  BUILTIN_SIDEBAR_SECTIONS.map((section) => [section.id, section])
)

const toBoolean = (value: unknown, fallback: boolean): boolean => {
  if (typeof value === 'boolean') return value
  if (typeof value === 'number') {
    if (value === 1) return true
    if (value === 0) return false
    return fallback
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true' || normalized === '1') return true
    if (normalized === 'false' || normalized === '0') return false
  }
  return fallback
}

const parseRecord = (value: unknown): Record<string, unknown> | null => {
  if (!value || String(value).trim() === '') return null
  if (typeof value === 'object') return value as Record<string, unknown>
  try {
    const parsed = JSON.parse(String(value))
    return parsed && typeof parsed === 'object'
      ? (parsed as Record<string, unknown>)
      : null
  } catch {
    return null
  }
}

const normalizeId = (value: string) =>
  value
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9_-]+/g, '-')
    .replace(/^-+|-+$/g, '')

const toOrder = (value: unknown, fallback: number) =>
  typeof value === 'number' && Number.isFinite(value) ? value : fallback

const sortByOrder = <T extends { order: number }>(items: T[]): T[] =>
  [...items].sort((a, b) => a.order - b.order)

function parseHeaderItem(
  id: string,
  raw: unknown,
  fallback: HeaderNavItemConfig
): HeaderNavItemConfig {
  if (
    typeof raw === 'boolean' ||
    typeof raw === 'string' ||
    typeof raw === 'number'
  ) {
    return {
      ...fallback,
      enabled: toBoolean(raw, fallback.enabled),
    }
  }

  if (!raw || typeof raw !== 'object') return { ...fallback }

  const record = raw as Record<string, unknown>
  return {
    ...fallback,
    id: String(record.id ?? id),
    label:
      typeof record.label === 'string' && record.label.trim()
        ? record.label.trim()
        : fallback.label,
    href:
      typeof record.href === 'string' && record.href.trim()
        ? record.href.trim()
        : fallback.href,
    enabled: toBoolean(record.enabled, fallback.enabled),
    requireAuth: toBoolean(record.requireAuth, fallback.requireAuth ?? false),
    external: toBoolean(record.external, fallback.external ?? false),
    order: toOrder(record.order, fallback.order),
    kind: record.kind === 'custom' ? 'custom' : fallback.kind,
  }
}

function parseHeaderItemsArray(raw: unknown): HeaderNavItemConfig[] {
  if (!Array.isArray(raw)) return []

  return raw
    .map((entry, index) => {
      if (!entry || typeof entry !== 'object') return null
      const record = entry as Record<string, unknown>
      const id = normalizeId(String(record.id ?? ''))
      if (!id) return null
      const fallback = HEADER_BUILTIN_BY_ID.get(id) ?? {
        id,
        label: id,
        href: '/',
        enabled: true,
        requireAuth: false,
        external: false,
        order: BUILTIN_HEADER_ITEMS.length + index,
        kind: 'custom' as const,
      }
      return parseHeaderItem(id, record, fallback)
    })
    .filter((item): item is HeaderNavItemConfig => Boolean(item))
}

export function parseHeaderNavModules(raw: unknown): HeaderNavModulesConfig {
  const parsed = parseRecord(raw)
  if (!parsed) return cloneHeaderNavDefault()

  const byId = new Map<string, HeaderNavItemConfig>()

  parseHeaderItemsArray(parsed.items).forEach((item) => {
    byId.set(item.id, item)
  })

  BUILTIN_HEADER_ITEMS.forEach((fallback) => {
    const rawItem = parsed[fallback.id]
    const parsedItem = parseHeaderItem(
      fallback.id,
      rawItem ?? byId.get(fallback.id) ?? fallback,
      fallback
    )
    byId.set(fallback.id, parsedItem)
  })

  parseHeaderItemsArray(parsed.customItems).forEach((item) => {
    byId.set(item.id, { ...item, kind: 'custom' })
  })

  return {
    items: sortByOrder([...byId.values()]),
  }
}

export function parseHeaderNavModulesFromStatus(
  status: Record<string, unknown> | null
): HeaderNavModulesConfig {
  return parseHeaderNavModules(status?.HeaderNavModules)
}

export function serializeHeaderNavModules(
  config: HeaderNavModulesConfig
): string {
  const items = sortByOrder(config.items).map((item, index) => ({
    ...item,
    order: index,
    id: normalizeId(item.id) || `nav-${index + 1}`,
  }))

  const payload: Record<string, unknown> = { items }
  items.forEach((item) => {
    if (item.kind === 'builtin') {
      payload[item.id] = item
    }
  })
  payload.customItems = items.filter((item) => item.kind === 'custom')

  return JSON.stringify(payload)
}

export function cloneHeaderNavDefault(): HeaderNavModulesConfig {
  return {
    items: HEADER_NAV_DEFAULT.items.map((item) => ({ ...item })),
  }
}

function parseSidebarItem(
  id: string,
  raw: unknown,
  fallback: SidebarItemConfig
): SidebarItemConfig {
  if (
    typeof raw === 'boolean' ||
    typeof raw === 'string' ||
    typeof raw === 'number'
  ) {
    return {
      ...fallback,
      enabled: toBoolean(raw, fallback.enabled),
    }
  }

  if (!raw || typeof raw !== 'object') return { ...fallback }

  const record = raw as Record<string, unknown>
  return {
    ...fallback,
    id: String(record.id ?? id),
    label:
      typeof record.label === 'string' && record.label.trim()
        ? record.label.trim()
        : fallback.label,
    url:
      typeof record.url === 'string' && record.url.trim()
        ? record.url.trim()
        : fallback.url,
    enabled: toBoolean(record.enabled, fallback.enabled),
    external: toBoolean(record.external, fallback.external ?? false),
    icon:
      typeof record.icon === 'string' && record.icon.trim()
        ? record.icon.trim()
        : fallback.icon,
    order: toOrder(record.order, fallback.order),
    kind: record.kind === 'custom' ? 'custom' : fallback.kind,
  }
}

function parseSidebarSection(
  id: string,
  raw: unknown,
  fallback: SidebarSectionConfig
): SidebarSectionConfig {
  const record =
    raw && typeof raw === 'object' ? (raw as Record<string, unknown>) : {}
  const byId = new Map<string, SidebarItemConfig>()

  if (Array.isArray(record.items)) {
    record.items.forEach((entry, index) => {
      if (!entry || typeof entry !== 'object') return
      const entryRecord = entry as Record<string, unknown>
      const itemId = normalizeId(String(entryRecord.id ?? ''))
      if (!itemId) return
      const itemFallback =
        fallback.items.find((item) => item.id === itemId) ??
        ({
          id: itemId,
          label: itemId,
          url: '/',
          enabled: true,
          order: fallback.items.length + index,
          kind: 'custom',
        } satisfies SidebarItemConfig)
      byId.set(itemId, parseSidebarItem(itemId, entryRecord, itemFallback))
    })
  }

  fallback.items.forEach((item) => {
    const legacyRaw =
      record[item.id] ?? (item.id === 'overview' ? record.detail : undefined)

    const parsedItem = byId.get(item.id)
    if (parsedItem) {
      byId.set(item.id, {
        ...parsedItem,
        enabled:
          legacyRaw === undefined
            ? parsedItem.enabled
            : toBoolean(legacyRaw, parsedItem.enabled),
      })
      return
    }

    byId.set(
      item.id,
      parseSidebarItem(item.id, legacyRaw ?? item, item)
    )
  })

  Object.entries(record).forEach(([key, value], index) => {
    if (key === 'enabled' || key === 'items' || key === 'label') return
    if (fallback.items.some((item) => item.id === key)) return
    if (!value || typeof value !== 'object') return
    const itemId = normalizeId(key)
    if (!itemId || byId.has(itemId)) return
    byId.set(
      itemId,
      parseSidebarItem(itemId, value, {
        id: itemId,
        label: itemId,
        url: '/',
        enabled: true,
        order: fallback.items.length + index,
        kind: 'custom',
      })
    )
  })

  return {
    ...fallback,
    id,
    label:
      typeof record.label === 'string' && record.label.trim()
        ? record.label.trim()
        : fallback.label,
    enabled: toBoolean(record.enabled, fallback.enabled),
    order: toOrder(record.order, fallback.order),
    kind: record.kind === 'custom' ? 'custom' : fallback.kind,
    items: sortByOrder([...byId.values()]),
  }
}

function parseSidebarSectionsArray(raw: unknown): SidebarSectionConfig[] {
  if (!Array.isArray(raw)) return []

  return raw
    .map((entry, index) => {
      if (!entry || typeof entry !== 'object') return null
      const record = entry as Record<string, unknown>
      const id = normalizeId(String(record.id ?? ''))
      if (!id) return null
      const fallback =
        SIDEBAR_SECTION_BY_ID.get(id) ??
        ({
          id,
          label: id,
          enabled: true,
          order: BUILTIN_SIDEBAR_SECTIONS.length + index,
          kind: 'custom',
          items: [],
        } satisfies SidebarSectionConfig)
      return parseSidebarSection(id, record, fallback)
    })
    .filter((section): section is SidebarSectionConfig => Boolean(section))
}

export function parseSidebarModulesAdmin(
  raw: unknown
): SidebarModulesAdminConfig {
  const parsed = parseRecord(raw)
  if (!parsed) return cloneSidebarDefault()

  const byId = new Map<string, SidebarSectionConfig>()

  parseSidebarSectionsArray(parsed.sections).forEach((section) => {
    byId.set(section.id, section)
  })

  BUILTIN_SIDEBAR_SECTIONS.forEach((fallback) => {
    byId.set(
      fallback.id,
      parseSidebarSection(
        fallback.id,
        parsed[fallback.id] ?? byId.get(fallback.id) ?? fallback,
        fallback
      )
    )
  })

  parseSidebarSectionsArray(parsed.customSections).forEach((section) => {
    byId.set(section.id, { ...section, kind: 'custom' })
  })

  Object.entries(parsed).forEach(([key, value]) => {
    if (
      key === 'sections' ||
      key === 'customSections' ||
      SIDEBAR_SECTION_BY_ID.has(key) ||
      byId.has(key)
    ) {
      return
    }
    if (!value || typeof value !== 'object') return
    const sectionId = normalizeId(key)
    if (!sectionId) return
    byId.set(
      sectionId,
      parseSidebarSection(sectionId, value, {
        id: sectionId,
        label: sectionId,
        enabled: true,
        order: byId.size,
        kind: 'custom',
        items: [],
      })
    )
  })

  return {
    sections: sortByOrder([...byId.values()]),
  }
}

export function serializeSidebarModulesAdmin(
  config: SidebarModulesAdminConfig
): string {
  const sections = sortByOrder(config.sections).map((section, sectionIndex) => {
    const id = normalizeId(section.id) || `section-${sectionIndex + 1}`
    return {
      ...section,
      id,
      order: sectionIndex,
      items: sortByOrder(section.items).map((item, itemIndex) => ({
        ...item,
        id: normalizeId(item.id) || `module-${itemIndex + 1}`,
        order: itemIndex,
      })),
    }
  })

  const payload: Record<string, unknown> = { sections }
  sections.forEach((section) => {
    const legacySection: Record<string, unknown> = {
      ...section,
      enabled: section.enabled,
      items: section.items,
    }
    section.items.forEach((item) => {
      legacySection[item.id] = item.enabled
    })
    payload[section.id] = legacySection
  })
  payload.customSections = sections.filter((section) => section.kind === 'custom')

  return JSON.stringify(payload)
}

export function cloneSidebarDefault(): SidebarModulesAdminConfig {
  return {
    sections: SIDEBAR_MODULES_DEFAULT.sections.map((section) => ({
      ...section,
      items: section.items.map((item) => ({ ...item })),
    })),
  }
}

export function getSidebarVisibilityMap(
  config: SidebarModulesAdminConfig
): SidebarVisibilityConfig {
  return config.sections.reduce<SidebarVisibilityConfig>((acc, section) => {
    acc[section.id] = { enabled: section.enabled }
    section.items.forEach((item) => {
      acc[section.id][item.id] = item.enabled
    })
    return acc
  }, {})
}

export function createCustomHeaderItem(order: number): HeaderNavItemConfig {
  const id = `custom-${Date.now()}`
  return {
    id,
    label: 'New link',
    href: '/',
    enabled: true,
    requireAuth: false,
    external: false,
    order,
    kind: 'custom',
  }
}

export function createCustomSidebarSection(
  order: number
): SidebarSectionConfig {
  const id = `custom-section-${Date.now()}`
  return {
    id,
    label: 'New section',
    enabled: true,
    order,
    kind: 'custom',
    items: [],
  }
}

export function createCustomSidebarItem(order: number): SidebarItemConfig {
  const id = `custom-module-${Date.now()}`
  return {
    id,
    label: 'New module',
    url: '/',
    enabled: true,
    external: false,
    icon: 'Link',
    order,
    kind: 'custom',
  }
}

export function isValidNavigationUrl(value: string): boolean {
  const trimmed = value.trim()
  return (
    trimmed.startsWith('/') ||
    trimmed.startsWith('http://') ||
    trimmed.startsWith('https://')
  )
}

export function normalizeNavigationId(value: string): string {
  return normalizeId(value)
}
