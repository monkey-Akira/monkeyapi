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
  SIDEBAR_MODULES_DEFAULT,
  createCustomSidebarItem,
  createCustomSidebarSection,
  isValidNavigationUrl,
  normalizeNavigationId,
  serializeSidebarModulesAdmin,
  type SidebarItemConfig,
  type SidebarModulesAdminConfig,
  type SidebarSectionConfig,
} from './config'

type SidebarModulesSectionProps = {
  config: SidebarModulesAdminConfig
  initialSerialized: string
}

function normalizeSections(
  sections: SidebarSectionConfig[]
): SidebarSectionConfig[] {
  return sections.map((section, sectionIndex) => ({
    ...section,
    id:
      section.kind === 'builtin'
        ? section.id
        : normalizeNavigationId(section.id),
    order: sectionIndex,
    items: section.items.map((item, itemIndex) => ({
      ...item,
      id: item.kind === 'builtin' ? item.id : normalizeNavigationId(item.id),
      order: itemIndex,
    })),
  }))
}

export function SidebarModulesSection({
  config,
  initialSerialized,
}: SidebarModulesSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const user = useAuthStore((state) => state.auth.user)
  const canEdit = user?.role === ROLE.SUPER_ADMIN
  const formDefaults = useMemo(() => config.sections, [config])
  const [sections, setSections] =
    useState<SidebarSectionConfig[]>(formDefaults)

  useEffect(() => {
    setSections(formDefaults)
  }, [formDefaults])

  const updateSection = (
    sectionIndex: number,
    patch: Partial<SidebarSectionConfig>
  ) => {
    setSections((prev) =>
      prev.map((section, index) =>
        index === sectionIndex ? { ...section, ...patch } : section
      )
    )
  }

  const updateItem = (
    sectionIndex: number,
    itemIndex: number,
    patch: Partial<SidebarItemConfig>
  ) => {
    setSections((prev) =>
      prev.map((section, index) => {
        if (index !== sectionIndex) return section
        return {
          ...section,
          items: section.items.map((item, currentItemIndex) =>
            currentItemIndex === itemIndex ? { ...item, ...patch } : item
          ),
        }
      })
    )
  }

  const moveSection = (sectionIndex: number, direction: -1 | 1) => {
    setSections((prev) => {
      const target = sectionIndex + direction
      if (target < 0 || target >= prev.length) return prev
      const next = [...prev]
      const [section] = next.splice(sectionIndex, 1)
      next.splice(target, 0, section)
      return normalizeSections(next)
    })
  }

  const moveItem = (
    sectionIndex: number,
    itemIndex: number,
    direction: -1 | 1
  ) => {
    setSections((prev) =>
      normalizeSections(
        prev.map((section, index) => {
          if (index !== sectionIndex) return section
          const target = itemIndex + direction
          if (target < 0 || target >= section.items.length) return section
          const items = [...section.items]
          const [item] = items.splice(itemIndex, 1)
          items.splice(target, 0, item)
          return { ...section, items }
        })
      )
    )
  }

  const addSection = () => {
    if (!canEdit) return
    setSections((prev) => [...prev, createCustomSidebarSection(prev.length)])
  }

  const addItem = (sectionIndex: number) => {
    if (!canEdit) return
    setSections((prev) =>
      prev.map((section, index) =>
        index === sectionIndex
          ? {
              ...section,
              items: [
                ...section.items,
                createCustomSidebarItem(section.items.length),
              ],
            }
          : section
      )
    )
  }

  const deleteSection = (sectionIndex: number) => {
    if (!canEdit) return
    setSections((prev) => normalizeSections(prev.filter((_, i) => i !== sectionIndex)))
  }

  const deleteItem = (sectionIndex: number, itemIndex: number) => {
    if (!canEdit) return
    setSections((prev) =>
      normalizeSections(
        prev.map((section, index) =>
          index === sectionIndex
            ? {
                ...section,
                items: section.items.filter((_, i) => i !== itemIndex),
              }
            : section
        )
      )
    )
  }

  const validateSections = (nextSections: SidebarSectionConfig[]) => {
    const sectionIds = new Set<string>()

    for (const section of nextSections) {
      if (!section.label.trim()) {
        toast.error(t('Name cannot be empty'))
        return false
      }

      const sectionId =
        section.kind === 'builtin'
          ? section.id
          : normalizeNavigationId(section.id)
      if (!sectionId) {
        toast.error(t('ID cannot be empty'))
        return false
      }
      if (sectionIds.has(sectionId)) {
        toast.error(t('ID already exists'))
        return false
      }
      sectionIds.add(sectionId)

      const itemIds = new Set<string>()
      for (const item of section.items) {
        if (!item.label.trim()) {
          toast.error(t('Name cannot be empty'))
          return false
        }
        const itemId =
          item.kind === 'builtin' ? item.id : normalizeNavigationId(item.id)
        if (!itemId) {
          toast.error(t('ID cannot be empty'))
          return false
        }
        if (itemIds.has(itemId)) {
          toast.error(t('ID already exists'))
          return false
        }
        itemIds.add(itemId)

        if (item.url && !isValidNavigationUrl(item.url)) {
          toast.error(t('Please enter a valid URL'))
          return false
        }
        if (item.kind === 'custom' && !item.url?.trim()) {
          toast.error(t('Please enter a valid URL'))
          return false
        }
      }
    }

    return true
  }

  const onSubmit = async () => {
    if (!canEdit) return

    const nextSections = normalizeSections(sections)
    if (!validateSections(nextSections)) return

    const serialized = serializeSidebarModulesAdmin({
      sections: nextSections,
    })
    if (serialized === initialSerialized) return

    await updateOption.mutateAsync({
      key: 'SidebarModulesAdmin',
      value: serialized,
    })
  }

  const resetToDefault = () => {
    if (!canEdit) return
    setSections(SIDEBAR_MODULES_DEFAULT.sections)
  }

  return (
    <SettingsSection title={t('Sidebar modules')}>
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
          saveLabel='Save sidebar modules'
        />

        {!canEdit && (
          <div className='text-muted-foreground rounded-lg border px-3 py-2 text-sm'>
            {t('Only super administrators can modify this setting.')}
          </div>
        )}

        <div className='space-y-4'>
          {sections.map((section, sectionIndex) => {
            const isBuiltinSection = section.kind === 'builtin'
            return (
              <div
                key={`${section.id}-${sectionIndex}`}
                className='bg-muted/20 rounded-xl border p-3'
              >
                <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                  <div className='flex min-w-0 items-center gap-2'>
                    <span className='truncate text-sm font-semibold'>
                      {section.label || section.id}
                    </span>
                    <Badge variant='secondary'>
                      {isBuiltinSection ? t('Built-in') : t('Custom')}
                    </Badge>
                  </div>
                  <div className='flex items-center gap-1'>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      onClick={() => moveSection(sectionIndex, -1)}
                      disabled={!canEdit || sectionIndex === 0}
                      title={t('Move up')}
                    >
                      <ArrowUp />
                    </Button>
                    <Button
                      type='button'
                      size='icon-sm'
                      variant='ghost'
                      onClick={() => moveSection(sectionIndex, 1)}
                      disabled={!canEdit || sectionIndex === sections.length - 1}
                      title={t('Move down')}
                    >
                      <ArrowDown />
                    </Button>
                    {!isBuiltinSection && (
                      <Button
                        type='button'
                        size='icon-sm'
                        variant='destructive'
                        onClick={() => deleteSection(sectionIndex)}
                        disabled={!canEdit}
                        title={t('Delete')}
                      >
                        <Trash2 />
                      </Button>
                    )}
                  </div>
                </div>

                <div className='grid gap-3 md:grid-cols-[1fr_1fr_auto]'>
                  <label className='space-y-1.5 text-sm'>
                    <span className='text-muted-foreground'>{t('ID')}</span>
                    <Input
                      value={section.id}
                      disabled={!canEdit || isBuiltinSection}
                      onChange={(event) =>
                        updateSection(sectionIndex, { id: event.target.value })
                      }
                    />
                  </label>
                  <label className='space-y-1.5 text-sm'>
                    <span className='text-muted-foreground'>{t('Name')}</span>
                    <Input
                      value={section.label}
                      disabled={!canEdit}
                      onChange={(event) =>
                        updateSection(sectionIndex, {
                          label: event.target.value,
                        })
                      }
                    />
                  </label>
                  <ToggleRow
                    label={t('Enabled')}
                    checked={section.enabled}
                    disabled={!canEdit}
                    onCheckedChange={(enabled) =>
                      updateSection(sectionIndex, { enabled })
                    }
                  />
                </div>

                <div className='mt-4 space-y-3 border-l pl-3'>
                  {section.items.map((item, itemIndex) => {
                    const isBuiltinItem = item.kind === 'builtin'
                    return (
                      <div
                        key={`${section.id}.${item.id}-${itemIndex}`}
                        className='rounded-lg border bg-background/60 p-3'
                      >
                        <div className='mb-3 flex flex-wrap items-center justify-between gap-2'>
                          <div className='flex min-w-0 items-center gap-2'>
                            <span className='truncate text-sm font-medium'>
                              {item.label || item.id}
                            </span>
                            <Badge variant='outline'>
                              {isBuiltinItem ? t('Built-in') : t('Custom')}
                            </Badge>
                          </div>
                          <div className='flex items-center gap-1'>
                            <Button
                              type='button'
                              size='icon-sm'
                              variant='ghost'
                              onClick={() => moveItem(sectionIndex, itemIndex, -1)}
                              disabled={!canEdit || itemIndex === 0}
                              title={t('Move up')}
                            >
                              <ArrowUp />
                            </Button>
                            <Button
                              type='button'
                              size='icon-sm'
                              variant='ghost'
                              onClick={() => moveItem(sectionIndex, itemIndex, 1)}
                              disabled={
                                !canEdit ||
                                itemIndex === section.items.length - 1
                              }
                              title={t('Move down')}
                            >
                              <ArrowDown />
                            </Button>
                            {!isBuiltinItem && (
                              <Button
                                type='button'
                                size='icon-sm'
                                variant='destructive'
                                onClick={() =>
                                  deleteItem(sectionIndex, itemIndex)
                                }
                                disabled={!canEdit}
                                title={t('Delete')}
                              >
                                <Trash2 />
                              </Button>
                            )}
                          </div>
                        </div>

                        <div className='grid gap-3 md:grid-cols-4'>
                          <label className='space-y-1.5 text-sm'>
                            <span className='text-muted-foreground'>
                              {t('ID')}
                            </span>
                            <Input
                              value={item.id}
                              disabled={!canEdit || isBuiltinItem}
                              onChange={(event) =>
                                updateItem(sectionIndex, itemIndex, {
                                  id: event.target.value,
                                })
                              }
                            />
                          </label>
                          <label className='space-y-1.5 text-sm'>
                            <span className='text-muted-foreground'>
                              {t('Name')}
                            </span>
                            <Input
                              value={item.label}
                              disabled={!canEdit}
                              onChange={(event) =>
                                updateItem(sectionIndex, itemIndex, {
                                  label: event.target.value,
                                })
                              }
                            />
                          </label>
                          <label className='space-y-1.5 text-sm'>
                            <span className='text-muted-foreground'>
                              {t('URL')}
                            </span>
                            <Input
                              value={item.url ?? ''}
                              disabled={!canEdit}
                              onChange={(event) =>
                                updateItem(sectionIndex, itemIndex, {
                                  url: event.target.value,
                                })
                              }
                            />
                          </label>
                          <label className='space-y-1.5 text-sm'>
                            <span className='text-muted-foreground'>
                              {t('Icon')}
                            </span>
                            <Input
                              value={item.icon ?? ''}
                              disabled={!canEdit}
                              placeholder='Link'
                              onChange={(event) =>
                                updateItem(sectionIndex, itemIndex, {
                                  icon: event.target.value,
                                })
                              }
                            />
                          </label>
                        </div>

                        <div className='mt-3 grid gap-2 sm:grid-cols-2'>
                          <ToggleRow
                            label={t('Enabled')}
                            checked={item.enabled}
                            disabled={!canEdit || !section.enabled}
                            onCheckedChange={(enabled) =>
                              updateItem(sectionIndex, itemIndex, { enabled })
                            }
                          />
                          <ToggleRow
                            label={t('External link')}
                            checked={Boolean(item.external)}
                            disabled={!canEdit}
                            onCheckedChange={(external) =>
                              updateItem(sectionIndex, itemIndex, { external })
                            }
                          />
                        </div>
                      </div>
                    )
                  })}

                  <Button
                    type='button'
                    variant='outline'
                    onClick={() => addItem(sectionIndex)}
                    disabled={!canEdit}
                  >
                    <Plus data-icon='inline-start' />
                    {t('Add module')}
                  </Button>
                </div>
              </div>
            )
          })}
        </div>

        <Button
          type='button'
          variant='outline'
          className='w-fit'
          onClick={addSection}
          disabled={!canEdit}
        >
          <Plus data-icon='inline-start' />
          {t('Add section')}
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
