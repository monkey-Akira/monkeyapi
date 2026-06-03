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
import { useEffect, useMemo, useState } from 'react'
import { ArrowDown, ArrowUp, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  HEADER_NAV_DEFAULT,
  createCustomHeaderItem,
  isValidNavigationUrl,
  normalizeNavigationId,
  serializeHeaderNavModules,
  type HeaderNavItemConfig,
  type HeaderNavModulesConfig,
} from './config'

type HeaderNavigationSectionProps = {
  config: HeaderNavModulesConfig
  initialSerialized: string
}

function normalizeItems(items: HeaderNavItemConfig[]): HeaderNavItemConfig[] {
  return items.map((item, index) => ({
    ...item,
    id: item.kind === 'builtin' ? item.id : normalizeNavigationId(item.id),
    order: index,
  }))
}

export function HeaderNavigationSection({
  config,
  initialSerialized,
}: HeaderNavigationSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const user = useAuthStore((state) => state.auth.user)
  const canEdit = user?.role === ROLE.SUPER_ADMIN
  const formDefaults = useMemo(() => config.items, [config])
  const [items, setItems] = useState<HeaderNavItemConfig[]>(formDefaults)

  useEffect(() => {
    setItems(formDefaults)
  }, [formDefaults])

  const updateItem = (
    index: number,
    patch: Partial<HeaderNavItemConfig>
  ) => {
    setItems((prev) =>
      prev.map((item, itemIndex) =>
        itemIndex === index ? { ...item, ...patch } : item
      )
    )
  }

  const moveItem = (index: number, direction: -1 | 1) => {
    setItems((prev) => {
      const target = index + direction
      if (target < 0 || target >= prev.length) return prev
      const next = [...prev]
      const [item] = next.splice(index, 1)
      next.splice(target, 0, item)
      return normalizeItems(next)
    })
  }

  const addItem = () => {
    if (!canEdit) return
    setItems((prev) => [...prev, createCustomHeaderItem(prev.length)])
  }

  const deleteItem = (index: number) => {
    if (!canEdit) return
    setItems((prev) => normalizeItems(prev.filter((_, i) => i !== index)))
  }

  const validateItems = (nextItems: HeaderNavItemConfig[]) => {
    const ids = new Set<string>()

    for (const item of nextItems) {
      if (!item.label.trim()) {
        toast.error(t('Name cannot be empty'))
        return false
      }
      if (!item.href.trim() || !isValidNavigationUrl(item.href)) {
        toast.error(t('Please enter a valid URL'))
        return false
      }

      const id = item.kind === 'builtin' ? item.id : normalizeNavigationId(item.id)
      if (!id) {
        toast.error(t('ID cannot be empty'))
        return false
      }
      if (ids.has(id)) {
        toast.error(t('ID already exists'))
        return false
      }
      ids.add(id)
    }

    return true
  }

  const onSubmit = async () => {
    if (!canEdit) return

    const nextItems = normalizeItems(items)
    if (!validateItems(nextItems)) return

    const serialized = serializeHeaderNavModules({ items: nextItems })
    if (serialized === initialSerialized) return

    await updateOption.mutateAsync({
      key: 'HeaderNavModules',
      value: serialized,
    })
  }

  const resetToDefault = () => {
    if (!canEdit) return
    setItems(HEADER_NAV_DEFAULT.items)
  }

  return (
    <SettingsSection title={t('Header navigation')}>
      <SettingsForm
        onSubmit={(event) => {
          event.preventDefault()
          void onSubmit()
        }}
      >
        <SettingsPageFormActions
          onSave={() => void onSubmit()}
          onReset={resetToDefault}
          isSaving={updateOption.isPending}
          isSaveDisabled={!canEdit}
          isResetDisabled={!canEdit}
          resetLabel='Reset to default'
          saveLabel='Save navigation'
        />

        {!canEdit && (
          <div className='text-muted-foreground rounded-lg border px-3 py-2 text-sm'>
            {t('Only super administrators can modify this setting.')}
          </div>
        )}

        <div className='space-y-3'>
          {items.map((item, index) => {
            const isBuiltin = item.kind === 'builtin'
            return (
              <div
                key={`${item.id}-${index}`}
                className='bg-muted/20 rounded-xl border p-3'
              >
                <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                  <div className='flex min-w-0 items-center gap-2'>
                    <span className='truncate text-sm font-medium'>
                      {item.label || item.id}
                    </span>
                    <Badge variant='secondary'>
                      {isBuiltin ? t('Built-in') : t('Custom')}
                    </Badge>
                  </div>
                  <div className='flex items-center gap-1'>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      onClick={() => moveItem(index, -1)}
                      disabled={!canEdit || index === 0}
                      title={t('Move up')}
                    >
                      <ArrowUp />
                    </Button>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      onClick={() => moveItem(index, 1)}
                      disabled={!canEdit || index === items.length - 1}
                      title={t('Move down')}
                    >
                      <ArrowDown />
                    </Button>
                    {!isBuiltin && (
                      <Button
                        type='button'
                        size='icon-sm'
                        variant='destructive'
                        onClick={() => deleteItem(index)}
                        disabled={!canEdit}
                        title={t('Delete')}
                      >
                        <Trash2 />
                      </Button>
                    )}
                  </div>
                </div>

                <div className='grid gap-3 md:grid-cols-3'>
                  <label className='space-y-1.5 text-sm'>
                    <span className='text-muted-foreground'>{t('ID')}</span>
                    <Input
                      value={item.id}
                      disabled={!canEdit || isBuiltin}
                      onChange={(event) =>
                        updateItem(index, { id: event.target.value })
                      }
                    />
                  </label>
                  <label className='space-y-1.5 text-sm'>
                    <span className='text-muted-foreground'>{t('Name')}</span>
                    <Input
                      value={item.label}
                      disabled={!canEdit}
                      onChange={(event) =>
                        updateItem(index, { label: event.target.value })
                      }
                    />
                  </label>
                  <label className='space-y-1.5 text-sm md:col-span-1'>
                    <span className='text-muted-foreground'>{t('URL')}</span>
                    <Input
                      value={item.href}
                      disabled={!canEdit}
                      onChange={(event) =>
                        updateItem(index, { href: event.target.value })
                      }
                    />
                  </label>
                </div>

                <div className='mt-3 grid gap-2 sm:grid-cols-3'>
                  <ToggleRow
                    label={t('Enabled')}
                    checked={item.enabled}
                    disabled={!canEdit}
                    onCheckedChange={(enabled) => updateItem(index, { enabled })}
                  />
                  <ToggleRow
                    label={t('Require login')}
                    checked={Boolean(item.requireAuth)}
                    disabled={!canEdit}
                    onCheckedChange={(requireAuth) =>
                      updateItem(index, { requireAuth })
                    }
                  />
                  <ToggleRow
                    label={t('External link')}
                    checked={Boolean(item.external)}
                    disabled={!canEdit}
                    onCheckedChange={(external) =>
                      updateItem(index, { external })
                    }
                  />
                </div>
              </div>
            )
          })}
        </div>

        <Button
          type='button'
          variant='outline'
          className='w-fit'
          onClick={addItem}
          disabled={!canEdit}
        >
          <Plus data-icon='inline-start' />
          {t('Add module')}
        </Button>
      </SettingsForm>
    </SettingsSection>
  )
}

function ToggleRow(props: {
  label: string
  checked: boolean
  disabled?: boolean
  onCheckedChange: (checked: boolean) => void
}) {
  return (
    <div className='flex min-h-9 items-center justify-between gap-3 rounded-lg border px-3 py-2'>
      <span className='truncate text-sm'>{props.label}</span>
      <Switch
        checked={props.checked}
        disabled={props.disabled}
        onCheckedChange={props.onCheckedChange}
      />
    </div>
  )
}
