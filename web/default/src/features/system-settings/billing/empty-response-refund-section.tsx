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
import { useMemo, useState } from 'react'
import * as z from 'zod'
import type { Resolver } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { CheckCheck, Trash2, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { MultiSelect } from '@/components/multi-select'
import { getChannels, getEnabledModels } from '@/features/channels/api'
import type { Channel } from '@/features/channels/types'
import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const emptyResponseRefundSchema = z.object({
  mode: z.enum(['off', 'observe', 'refund']),
  models: z.array(z.string()),
  custom_response_enabled: z.boolean(),
  custom_response_text: z.string().max(4000),
})

type EmptyResponseRefundFormValues = z.infer<typeof emptyResponseRefundSchema>

type EmptyResponseRefundSectionProps = {
  defaultMode: string
  defaultModels: string
  defaultCustomResponseEnabled: boolean
  defaultCustomResponseText: string
}

const MODE_DESCRIPTIONS: Record<EmptyResponseRefundFormValues['mode'], string> =
  {
    off: 'No empty-response detection or refunds are performed.',
    observe:
      'Matching requests are marked in usage logs, but charges are unchanged.',
    refund:
      'Matching requests are settled at zero and receive a separate refund log.',
  }

function parseModels(value: string): string[] {
  try {
    const parsed = JSON.parse(value) as unknown
    if (!Array.isArray(parsed)) return []
    return parsed.filter(
      (modelName): modelName is string =>
        typeof modelName === 'string' && modelName.trim().length > 0
    )
  } catch {
    return []
  }
}

function parseMode(value: string): EmptyResponseRefundFormValues['mode'] {
  if (value === 'observe' || value === 'refund') return value
  return 'off'
}

function parseChannelModels(value: string): string[] {
  return value
    .split(',')
    .map((modelName) => modelName.trim())
    .filter(Boolean)
}

async function getAllEnabledChannels(): Promise<Channel[]> {
  const pageSize = 100
  const channels: Channel[] = []

  for (let page = 1; ; page += 1) {
    const response = await getChannels({
      p: page,
      page_size: pageSize,
      status: 'enabled',
      id_sort: true,
    })
    if (!response.success || !response.data) {
      throw new Error(response.message || 'Failed to load channels')
    }

    channels.push(...response.data.items)
    if (
      channels.length >= response.data.total ||
      response.data.items.length === 0
    ) {
      return channels
    }
  }
}

export function EmptyResponseRefundSection(
  props: EmptyResponseRefundSectionProps
) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [channelFilter, setChannelFilter] = useState('all')
  const enabledModelsQuery = useQuery({
    queryKey: ['channels', 'enabled-models'],
    queryFn: getEnabledModels,
    staleTime: 60_000,
  })
  const channelsQuery = useQuery({
    queryKey: ['channels', 'all-enabled-for-empty-response-refund'],
    queryFn: getAllEnabledChannels,
    staleTime: 60_000,
  })
  const enabledModels = useMemo(
    () => [...new Set(enabledModelsQuery.data?.data ?? [])].sort(),
    [enabledModelsQuery.data?.data]
  )
  const filteredModels = useMemo(() => {
    if (channelFilter === 'all') return enabledModels

    const channelId = Number(channelFilter)
    const channel = channelsQuery.data?.find((item) => item.id === channelId)
    if (!channel) return []

    const enabledModelSet = new Set(enabledModels)
    return [...new Set(parseChannelModels(channel.models))]
      .filter((modelName) => enabledModelSet.has(modelName))
      .sort()
  }, [channelFilter, channelsQuery.data, enabledModels])
  const modelOptions = useMemo(
    () =>
      filteredModels.map((modelName) => ({
        label: modelName,
        value: modelName,
      })),
    [filteredModels]
  )

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<EmptyResponseRefundFormValues>({
      resolver: zodResolver(emptyResponseRefundSchema) as Resolver<
        EmptyResponseRefundFormValues,
        unknown,
        EmptyResponseRefundFormValues
      >,
      defaultValues: {
        mode: parseMode(props.defaultMode),
        models: parseModels(props.defaultModels),
        custom_response_enabled: props.defaultCustomResponseEnabled,
        custom_response_text: props.defaultCustomResponseText,
      },
      onSubmit: async (_data, changedFields) => {
        for (const [key, value] of Object.entries(changedFields)) {
          await updateOption.mutateAsync({
            key: `empty_response_refund_setting.${key}`,
            value: key === 'models' ? JSON.stringify(value) : String(value),
          })
        }
      },
    })

  const mode = form.watch('mode')
  const selectedModels = form.watch('models')
  const modeDescription = MODE_DESCRIPTIONS[mode]
  const filteredModelSet = useMemo(
    () => new Set(filteredModels),
    [filteredModels]
  )
  const visibleSelectedModels = useMemo(
    () => selectedModels.filter((modelName) => filteredModelSet.has(modelName)),
    [filteredModelSet, selectedModels]
  )

  const updateVisibleSelection = (nextVisibleModels: string[]) => {
    const hiddenModels = selectedModels.filter(
      (modelName) => !filteredModelSet.has(modelName)
    )
    form.setValue('models', [...hiddenModels, ...nextVisibleModels], {
      shouldDirty: true,
      shouldValidate: true,
    })
  }

  const selectAllFilteredModels = () => {
    form.setValue(
      'models',
      [...new Set([...selectedModels, ...filteredModels])],
      {
        shouldDirty: true,
        shouldValidate: true,
      }
    )
  }

  const removeModel = (modelName: string) => {
    form.setValue(
      'models',
      selectedModels.filter((item) => item !== modelName),
      { shouldDirty: true, shouldValidate: true }
    )
  }

  return (
    <SettingsSection title={t('Empty Response Refund')}>
      <FormNavigationGuard when={isDirty} />
      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending || isSubmitting}
          />
          <FormDirtyIndicator isDirty={isDirty} />

          <FormField
            control={form.control}
            name='mode'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Refund Mode')}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl>
                    <SelectTrigger className='w-full'>
                      <SelectValue />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value='off'>{t('Off')}</SelectItem>
                    <SelectItem value='observe'>{t('Observe Only')}</SelectItem>
                    <SelectItem value='refund'>
                      {t('Automatic Refund')}
                    </SelectItem>
                  </SelectContent>
                </Select>
                <FormDescription>{t(modeDescription)}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='models'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Refund Models')}</FormLabel>
                <Select value={channelFilter} onValueChange={setChannelFilter}>
                  <SelectTrigger className='w-full'>
                    <SelectValue placeholder={t('Filter by channel')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value='all'>
                      {t('All enabled models')}
                    </SelectItem>
                    {(channelsQuery.data ?? []).map((channel) => (
                      <SelectItem key={channel.id} value={String(channel.id)}>
                        {channel.name} (#{channel.id})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <FormControl>
                  <MultiSelect
                    id='empty-response-refund-models'
                    options={modelOptions}
                    selected={visibleSelectedModels}
                    onChange={updateVisibleSelection}
                    placeholder={t('Search and select existing models')}
                    emptyText={t('No matching models')}
                    maxVisibleChips={5}
                    disabled={
                      enabledModelsQuery.isLoading || channelsQuery.isLoading
                    }
                  />
                </FormControl>
                <div className='flex flex-wrap items-center gap-2'>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={selectAllFilteredModels}
                    disabled={filteredModels.length === 0}
                  >
                    <CheckCheck />
                    {t('Select filtered models')}
                  </Button>
                  <Button
                    type='button'
                    variant='destructive'
                    size='sm'
                    onClick={() => field.onChange([])}
                    disabled={field.value.length === 0}
                  >
                    <Trash2 />
                    {t('Clear selected models')}
                  </Button>
                  <span className='text-muted-foreground text-sm'>
                    {t('{{count}} models selected', {
                      count: field.value.length,
                    })}
                  </span>
                </div>
                <FormDescription>
                  {t(
                    'The channel only filters the model list. Refunds still match exact original model names across all channels.'
                  )}
                </FormDescription>
                {field.value.length > 0 && (
                  <div className='border-border divide-border max-h-60 overflow-y-auto rounded-md border'>
                    {field.value.map((modelName) => (
                      <div
                        key={modelName}
                        className='flex min-h-9 items-center justify-between gap-3 border-b px-3 py-1.5 last:border-b-0'
                      >
                        <span className='min-w-0 truncate text-sm'>
                          {modelName}
                        </span>
                        <Button
                          type='button'
                          variant='ghost'
                          size='icon-xs'
                          onClick={() => removeModel(modelName)}
                          aria-label={t('Remove {{model}}', {
                            model: modelName,
                          })}
                          title={t('Remove model')}
                        >
                          <X />
                        </Button>
                      </div>
                    ))}
                  </div>
                )}
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='custom_response_enabled'
            render={({ field }) => (
              <FormItem className='border-border flex items-center justify-between gap-4 rounded-md border p-3'>
                <div className='space-y-1'>
                  <FormLabel>{t('Custom Empty Response Text')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Return the text below when an automatic refund is triggered.'
                    )}
                  </FormDescription>
                </div>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={mode !== 'refund'}
                  />
                </FormControl>
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='custom_response_text'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Response Text')}</FormLabel>
                <FormControl>
                  <Textarea
                    rows={4}
                    maxLength={4000}
                    placeholder={t('Enter the text returned to users')}
                    disabled={
                      mode !== 'refund' ||
                      !form.watch('custom_response_enabled')
                    }
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'This text does not add output tokens. Structured output and forced tool calls are refunded without injecting text.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
