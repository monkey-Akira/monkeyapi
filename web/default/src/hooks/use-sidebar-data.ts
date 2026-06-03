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
import {
  Activity,
  Box,
  Circle,
  CreditCard,
  FileText,
  FlaskConical,
  Key,
  LayoutDashboard,
  LinkIcon,
  ListTodo,
  MessageSquare,
  Radio,
  Settings,
  Ticket,
  User,
  Users,
  Wallet,
} from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  parseSidebarModulesAdmin,
  type SidebarItemConfig,
} from '@/lib/navigation-config'
import { type NavItem, type SidebarData } from '@/components/layout/types'
import { useStatus } from './use-status'

const ICONS = {
  Activity,
  Box,
  Circle,
  CreditCard,
  FileText,
  FlaskConical,
  Key,
  LayoutDashboard,
  Link: LinkIcon,
  ListTodo,
  MessageSquare,
  Radio,
  Settings,
  Ticket,
  User,
  Users,
  Wallet,
} as const

type BuiltinItemFactory = (item: SidebarItemConfig) => NavItem

const withConfigKey = (
  section: string,
  item: SidebarItemConfig,
  navItem: NavItem
): NavItem => ({
  ...navItem,
  title: item.label,
  external: item.external,
  configSection: section,
  configModule: item.id,
})

const getDisplayTitle = (
  item: Pick<SidebarItemConfig, 'kind' | 'label' | 'labelKey'>,
  t: (key: string) => string
) => (item.kind === 'builtin' ? t(item.labelKey ?? item.label) : item.label)

const builtinItemFactories: Record<string, BuiltinItemFactory> = {
  'chat.playground': (item) =>
    withConfigKey('chat', item, {
      title: item.label,
      url: '/playground',
      icon: FlaskConical,
    }),
  'chat.chat': (item) =>
    withConfigKey('chat', item, {
      title: item.label,
      icon: MessageSquare,
      type: 'chat-presets',
    }),
  'console.overview': (item) =>
    withConfigKey('console', item, {
      title: item.label,
      url: '/dashboard/overview',
      icon: Activity,
    }),
  'console.detail': (item) =>
    withConfigKey('console', item, {
      title: item.label,
      url: '/dashboard/models',
      icon: LayoutDashboard,
      configUrls: ['/dashboard', '/dashboard/overview', '/dashboard/models'],
    }),
  'console.token': (item) =>
    withConfigKey('console', item, {
      title: item.label,
      url: '/keys',
      icon: Key,
    }),
  'console.log': (item) =>
    withConfigKey('console', item, {
      title: item.label,
      url: '/usage-logs/common',
      icon: FileText,
    }),
  'console.midjourney': (item) =>
    withConfigKey('console', item, {
      title: item.label,
      url: '/usage-logs/drawing',
      icon: FileText,
    }),
  'console.task': (item) =>
    withConfigKey('console', item, {
      title: item.label,
      url: '/usage-logs/task',
      activeUrls: ['/usage-logs/drawing'],
      configUrls: ['/usage-logs/drawing', '/usage-logs/task'],
      icon: ListTodo,
    }),
  'personal.topup': (item) =>
    withConfigKey('personal', item, {
      title: item.label,
      url: '/wallet',
      icon: Wallet,
    }),
  'personal.personal': (item) =>
    withConfigKey('personal', item, {
      title: item.label,
      url: '/profile',
      icon: User,
    }),
  'admin.channel': (item) =>
    withConfigKey('admin', item, {
      title: item.label,
      url: '/channels',
      icon: Radio,
    }),
  'admin.models': (item) =>
    withConfigKey('admin', item, {
      title: item.label,
      url: '/models/metadata',
      icon: Box,
      configUrls: ['/models', '/models/metadata', '/models/deployments'],
    }),
  'admin.user': (item) =>
    withConfigKey('admin', item, {
      title: item.label,
      url: '/users',
      icon: Users,
    }),
  'admin.redemption': (item) =>
    withConfigKey('admin', item, {
      title: item.label,
      url: '/redemption-codes',
      icon: Ticket,
    }),
  'admin.subscription': (item) =>
    withConfigKey('admin', item, {
      title: item.label,
      url: '/subscriptions',
      icon: CreditCard,
    }),
  'admin.setting': (item) =>
    withConfigKey('admin', item, {
      title: item.label,
      url: '/system-settings/site',
      activeUrls: ['/system-settings'],
      icon: Settings,
    }),
}

function createCustomNavItem(
  sectionId: string,
  item: SidebarItemConfig
): NavItem | null {
  if (!item.url) return null
  const Icon = ICONS[item.icon as keyof typeof ICONS] ?? LinkIcon

  return withConfigKey(sectionId, item, {
    title: item.label,
    url: item.url,
    icon: Icon,
    external: item.external,
  })
}

export function useSidebarData(): SidebarData {
  const { t } = useTranslation()
  const { status } = useStatus()

  const config = useMemo(
    () => parseSidebarModulesAdmin(status?.SidebarModulesAdmin),
    [status?.SidebarModulesAdmin]
  )

  return useMemo(
    () => ({
      navGroups: config.sections
        .filter((section) => section.enabled)
        .map((section) => ({
          id: section.id,
          title: getDisplayTitle(section, t),
          items: section.items
            .filter((item) => item.enabled)
            .map((item) => {
              const factory = builtinItemFactories[`${section.id}.${item.id}`]
              const navItem = factory
                ? factory(item)
                : createCustomNavItem(section.id, item)
              if (!navItem) return null
              return {
                ...navItem,
                title: getDisplayTitle(item, t),
              }
            })
            .filter((item): item is NavItem => Boolean(item)),
        }))
        .filter((section) => section.items.length > 0),
    }),
    [config, t]
  )
}
