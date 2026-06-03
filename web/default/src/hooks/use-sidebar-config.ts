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
import { useMemo } from 'react'
import { useAuthStore } from '@/stores/auth-store'
import {
  getSidebarVisibilityMap,
  parseSidebarModulesAdmin,
  type SidebarVisibilityConfig,
} from '@/lib/navigation-config'
import { useStatus } from '@/hooks/use-status'
import type { NavGroup, NavItem } from '@/components/layout/types'

type SidebarModulesUserConfig = SidebarVisibilityConfig | null

const URL_TO_CONFIG_MAP: Record<string, { section: string; module: string }> = {
  '/playground': { section: 'chat', module: 'playground' },
  '/dashboard': { section: 'console', module: 'detail' },
  '/dashboard/overview': { section: 'console', module: 'overview' },
  '/dashboard/models': { section: 'console', module: 'detail' },
  '/dashboard/users': { section: 'console', module: 'detail' },
  '/keys': { section: 'console', module: 'token' },
  '/usage-logs': { section: 'console', module: 'log' },
  '/usage-logs/common': { section: 'console', module: 'log' },
  '/usage-logs/drawing': { section: 'console', module: 'midjourney' },
  '/usage-logs/task': { section: 'console', module: 'task' },
  '/wallet': { section: 'personal', module: 'topup' },
  '/profile': { section: 'personal', module: 'personal' },
  '/channels': { section: 'admin', module: 'channel' },
  '/models': { section: 'admin', module: 'models' },
  '/models/metadata': { section: 'admin', module: 'models' },
  '/models/deployments': { section: 'admin', module: 'models' },
  '/users': { section: 'admin', module: 'user' },
  '/redemption-codes': { section: 'admin', module: 'redemption' },
  '/subscriptions': { section: 'admin', module: 'subscription' },
  '/system-settings': { section: 'admin', module: 'setting' },
  '/system-settings/site': { section: 'admin', module: 'setting' },
}

function parseUserSidebarConfig(
  value: string | null | undefined
): SidebarModulesUserConfig {
  if (!value || value.trim() === '') return null

  try {
    const parsed = JSON.parse(value)
    if (!parsed || typeof parsed !== 'object') return null

    if (Array.isArray((parsed as Record<string, unknown>).sections)) {
      return getSidebarVisibilityMap(parseSidebarModulesAdmin(parsed))
    }

    const legacy = parsed as SidebarVisibilityConfig
    if (
      legacy.console &&
      legacy.console.overview === undefined &&
      legacy.console.detail !== undefined
    ) {
      legacy.console = {
        ...legacy.console,
        overview: legacy.console.detail,
      }
    }
    return legacy
  } catch {
    return null
  }
}

function isModuleEnabledByKey(
  section: string,
  module: string,
  adminConfig: SidebarVisibilityConfig,
  userConfig: SidebarModulesUserConfig
): boolean {
  const adminSection = adminConfig[section]
  const adminAllowed = Boolean(
    adminSection && adminSection.enabled && adminSection[module] === true
  )
  if (!adminAllowed) return false

  if (!userConfig) return true

  const userSection = userConfig[section]
  if (!userSection) return true
  if (userSection.enabled === false) return false
  return userSection[module] !== false
}

function isModuleEnabled(
  url: string,
  adminConfig: SidebarVisibilityConfig,
  userConfig: SidebarModulesUserConfig
): boolean {
  const mapping = URL_TO_CONFIG_MAP[url]
  if (!mapping) return true

  return isModuleEnabledByKey(
    mapping.section,
    mapping.module,
    adminConfig,
    userConfig
  )
}

function isNavItemVisible(
  item: NavItem,
  adminConfig: SidebarVisibilityConfig,
  userConfig: SidebarModulesUserConfig
): boolean {
  if (item.configSection && item.configModule) {
    return isModuleEnabledByKey(
      item.configSection,
      item.configModule,
      adminConfig,
      userConfig
    )
  }

  if ('type' in item && item.type === 'chat-presets') {
    return isModuleEnabledByKey('chat', 'chat', adminConfig, userConfig)
  }

  if ('url' in item && item.url) {
    const configUrls = item.configUrls ?? [item.url]
    return configUrls.some((url) =>
      isModuleEnabled(url as string, adminConfig, userConfig)
    )
  }

  if ('items' in item && item.items) {
    return item.items.some((subItem) =>
      isModuleEnabled(subItem.url as string, adminConfig, userConfig)
    )
  }

  return true
}

function filterNavItems(
  items: NavItem[],
  adminConfig: SidebarVisibilityConfig,
  userConfig: SidebarModulesUserConfig
): NavItem[] {
  return items
    .map((item) => {
      if ('items' in item && item.items) {
        return {
          ...item,
          items: item.items.filter((subItem) => {
            if (subItem.configSection && subItem.configModule) {
              return isModuleEnabledByKey(
                subItem.configSection,
                subItem.configModule,
                adminConfig,
                userConfig
              )
            }
            return isModuleEnabled(
              subItem.url as string,
              adminConfig,
              userConfig
            )
          }),
        }
      }
      return item
    })
    .filter((item) => isNavItemVisible(item, adminConfig, userConfig))
}

export function useSidebarConfig(navGroups: NavGroup[]): NavGroup[] {
  const { status } = useStatus()
  const { auth } = useAuthStore()

  const adminConfig = useMemo(
    () =>
      getSidebarVisibilityMap(
        parseSidebarModulesAdmin(
          status?.SidebarModulesAdmin as string | null | undefined
        )
      ),
    [status?.SidebarModulesAdmin]
  )

  const userConfig = useMemo(() => {
    if (auth?.user?.permissions?.sidebar_settings === false) {
      return null
    }
    return parseUserSidebarConfig(auth?.user?.sidebar_modules)
  }, [auth?.user?.permissions?.sidebar_settings, auth?.user?.sidebar_modules])

  return useMemo(
    () =>
      navGroups
        .map((group) => ({
          ...group,
          items: filterNavItems(group.items, adminConfig, userConfig),
        }))
        .filter((group) => group.items.length > 0),
    [navGroups, adminConfig, userConfig]
  )
}
