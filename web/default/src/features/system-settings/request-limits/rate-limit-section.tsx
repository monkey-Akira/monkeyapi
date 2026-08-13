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
import * as z from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useQuery } from '@tanstack/react-query'
import { Code2, Palette, Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { ComboboxInput } from '@/components/ui/combobox-input'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { getEnabledModels } from '@/features/channels/api'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import { RateLimitVisualEditor } from './rate-limit-visual-editor'

const isValidJSON = (value: string | undefined) => {
  if (!value || value.trim() === '') return true
  try {
    const parsed = JSON.parse(value)
    if (typeof parsed !== 'object' || Array.isArray(parsed)) {
      return false
    }
    for (const [, val] of Object.entries(parsed)) {
      if (!Array.isArray(val) || val.length !== 2) return false
      if (typeof val[0] !== 'number' || typeof val[1] !== 'number') return false
      if (val[0] < 0 || val[1] < 1) return false
      if (val[0] > 2147483647 || val[1] > 2147483647) return false
    }
    return true
  } catch {
    return false
  }
}

type ModelRateLimitConfig = {
  enabled: boolean
  limits: Record<string, number>
}

const EMPTY_MODEL_RATE_LIMIT_CONFIG: ModelRateLimitConfig = {
  enabled: false,
  limits: {},
}

function parseModelRateLimitConfig(value: string): ModelRateLimitConfig {
  try {
    const parsed = JSON.parse(value) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return EMPTY_MODEL_RATE_LIMIT_CONFIG
    }
    const candidate = parsed as Partial<ModelRateLimitConfig>
    if (
      typeof candidate.enabled !== 'boolean' ||
      !candidate.limits ||
      typeof candidate.limits !== 'object' ||
      Array.isArray(candidate.limits)
    ) {
      return EMPTY_MODEL_RATE_LIMIT_CONFIG
    }
    const limits: Record<string, number> = {}
    for (const [modelName, rpm] of Object.entries(candidate.limits)) {
      if (
        modelName.trim() !== modelName ||
        modelName.length === 0 ||
        typeof rpm !== 'number' ||
        !Number.isInteger(rpm) ||
        rpm < 1 ||
        rpm > 100000000
      ) {
        return EMPTY_MODEL_RATE_LIMIT_CONFIG
      }
      limits[modelName] = rpm
    }
    return { enabled: candidate.enabled, limits }
  } catch {
    return EMPTY_MODEL_RATE_LIMIT_CONFIG
  }
}

function isValidModelRateLimitConfig(value: string | undefined) {
  if (!value) return false
  try {
    const parsed = JSON.parse(value) as unknown
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return false
    }
    const candidate = parsed as Partial<ModelRateLimitConfig>
    if (
      typeof candidate.enabled !== 'boolean' ||
      !candidate.limits ||
      typeof candidate.limits !== 'object' ||
      Array.isArray(candidate.limits)
    ) {
      return false
    }
    return Object.entries(candidate.limits).every(
      ([modelName, rpm]) =>
        modelName.trim() === modelName &&
        modelName.length > 0 &&
        typeof rpm === 'number' &&
        Number.isInteger(rpm) &&
        rpm >= 1 &&
        rpm <= 100000000
    )
  } catch {
    return false
  }
}

type ModelRateLimitEditorProps = {
  value: string
  onChange: (value: string) => void
}

function ModelRateLimitEditor({ value, onChange }: ModelRateLimitEditorProps) {
  const { t } = useTranslation()
  const config = useMemo(() => parseModelRateLimitConfig(value), [value])
  const [selectedModel, setSelectedModel] = useState('')
  const [rpm, setRpm] = useState(10)
  const enabledModelsQuery = useQuery({
    queryKey: ['channels', 'enabled-models'],
    queryFn: getEnabledModels,
    staleTime: 60_000,
  })
  const modelOptions = useMemo(
    () =>
      [...new Set(enabledModelsQuery.data?.data ?? [])]
        .filter((modelName) => !(modelName in config.limits))
        .sort()
        .map((modelName) => ({ label: modelName, value: modelName })),
    [config.limits, enabledModelsQuery.data?.data]
  )
  const configuredModels = useMemo(
    () =>
      Object.entries(config.limits).sort(([left], [right]) =>
        left.localeCompare(right)
      ),
    [config.limits]
  )

  const updateConfig = (next: ModelRateLimitConfig) => {
    onChange(JSON.stringify(next))
  }

  const addModel = () => {
    if (!selectedModel || !Number.isInteger(rpm) || rpm < 1 || rpm > 100000000)
      return
    updateConfig({
      ...config,
      limits: { ...config.limits, [selectedModel]: rpm },
    })
    setSelectedModel('')
  }

  const updateRPM = (modelName: string, nextRPM: number) => {
    if (!Number.isInteger(nextRPM) || nextRPM < 1 || nextRPM > 100000000) {
      return
    }
    updateConfig({
      ...config,
      limits: { ...config.limits, [modelName]: nextRPM },
    })
  }

  const removeModel = (modelName: string) => {
    const nextLimits = { ...config.limits }
    delete nextLimits[modelName]
    updateConfig({ ...config, limits: nextLimits })
  }

  return (
    <div className='space-y-4'>
      <div className='flex items-center justify-between gap-4'>
        <div className='space-y-1'>
          <FormLabel>{t('Enable per-model limits')}</FormLabel>
          <FormDescription>
            {t(
              'Limits are counted separately for each user and original model name.'
            )}
          </FormDescription>
        </div>
        <Switch
          checked={config.enabled}
          onCheckedChange={(enabled) => updateConfig({ ...config, enabled })}
        />
      </div>

      <div className='grid gap-2 sm:grid-cols-[minmax(0,1fr)_9rem_auto]'>
        <ComboboxInput
          options={modelOptions}
          value={selectedModel}
          onValueChange={setSelectedModel}
          placeholder={t('Search enabled models')}
          emptyText={t('No available models')}
        />
        <div className='relative'>
          <Input
            type='number'
            min={1}
            max={100000000}
            step={1}
            value={rpm}
            onChange={(event) => setRpm(Number(event.target.value))}
            aria-label={t('Requests per minute')}
            className='pr-12'
          />
          <span className='text-muted-foreground pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-xs'>
            RPM
          </span>
        </div>
        <Button
          type='button'
          onClick={addModel}
          disabled={
            !selectedModel ||
            !Number.isInteger(rpm) ||
            rpm < 1 ||
            rpm > 100000000
          }
        >
          <Plus />
          {t('Add limit')}
        </Button>
      </div>

      {configuredModels.length === 0 ? (
        <div className='text-muted-foreground rounded-md border border-dashed px-4 py-6 text-center text-sm'>
          {t('No model limits configured')}
        </div>
      ) : (
        <div className='divide-border rounded-md border'>
          {configuredModels.map(([modelName, modelRPM]) => (
            <div
              key={modelName}
              className='grid min-h-12 items-center gap-3 border-b px-3 py-2 last:border-b-0 sm:grid-cols-[minmax(0,1fr)_9rem_2.25rem]'
            >
              <span className='min-w-0 truncate text-sm font-medium'>
                {modelName}
              </span>
              <div className='relative'>
                <Input
                  type='number'
                  min={1}
                  max={100000000}
                  step={1}
                  value={modelRPM}
                  onChange={(event) =>
                    updateRPM(modelName, Number(event.target.value))
                  }
                  aria-label={t('{{model}} requests per minute', {
                    model: modelName,
                  })}
                  className='h-8 pr-12'
                />
                <span className='text-muted-foreground pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-xs'>
                  RPM
                </span>
              </div>
              <Button
                type='button'
                variant='ghost'
                size='icon'
                title={t('Remove {{model}}', { model: modelName })}
                onClick={() => removeModel(modelName)}
              >
                <Trash2 />
                <span className='sr-only'>
                  {t('Remove {{model}}', { model: modelName })}
                </span>
              </Button>
            </div>
          ))}
        </div>
      )}

      <FormDescription>
        {t(
          'All attempts count toward the 60-second window. Exceeded requests return HTTP 429 before an upstream channel is selected.'
        )}
      </FormDescription>
    </div>
  )
}

const createRateLimitSchema = (t: (key: string) => string) =>
  z.object({
    ModelRequestRateLimitEnabled: z.boolean(),
    ModelRequestRateLimitDurationMinutes: z.number().min(0),
    ModelRequestRateLimitCount: z.number().min(0).max(100000000),
    ModelRequestRateLimitSuccessCount: z.number().min(1).max(100000000),
    ModelRequestRateLimitGroup: z
      .string()
      .optional()
      .refine(isValidJSON, {
        message: t('Invalid JSON format or values out of allowed range'),
      }),
    ModelRequestRateLimitModels: z
      .string()
      .refine(isValidModelRateLimitConfig, {
        message: t('Invalid per-model rate limit configuration'),
      }),
  })

type RateLimitFormValues = z.infer<ReturnType<typeof createRateLimitSchema>>

type RateLimitSectionProps = {
  defaultValues: RateLimitFormValues
}

export function RateLimitSection({ defaultValues }: RateLimitSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const [useVisualEditor, setUseVisualEditor] = useState(true)

  const rateLimitSchema = createRateLimitSchema(t)

  const form = useForm<RateLimitFormValues>({
    resolver: zodResolver(rateLimitSchema),
    mode: 'onChange', // Enable real-time validation
    defaultValues,
  })

  useEffect(() => {
    form.reset(defaultValues)
  }, [defaultValues, form])

  const onSubmit = async (values: RateLimitFormValues) => {
    const updates = Object.entries(values).filter(
      ([key, value]) =>
        value !== defaultValues[key as keyof RateLimitFormValues]
    )

    for (const [key, value] of updates) {
      await updateOption.mutateAsync({ key, value: value ?? '' })
    }
  }

  return (
    <SettingsSection title={t('Rate Limiting')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
            saveLabel='Save rate limits'
          />
          <FormField
            control={form.control}
            name='ModelRequestRateLimitEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable rate limiting')}</FormLabel>
                  <FormDescription>
                    {t(
                      'This controls model request rate limiting. Web/API route throttling is configured by environment variables and may still return 429.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <div className='grid gap-4 md:grid-cols-3'>
            <FormField
              control={form.control}
              name='ModelRequestRateLimitDurationMinutes'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Limit period')}</FormLabel>
                  <FormControl>
                    <div className='flex items-center gap-2'>
                      <Input
                        type='number'
                        min={0}
                        step={1}
                        {...field}
                        onChange={(e) =>
                          field.onChange(parseInt(e.target.value) || 0)
                        }
                      />
                      <span className='text-muted-foreground text-sm'>
                        {t('minutes')}
                      </span>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('Time window for rate limiting')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='ModelRequestRateLimitCount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max requests per period')}</FormLabel>
                  <FormControl>
                    <div className='flex items-center gap-2'>
                      <Input
                        type='number'
                        min={0}
                        max={100000000}
                        step={1}
                        {...field}
                        onChange={(e) =>
                          field.onChange(parseInt(e.target.value) || 0)
                        }
                      />
                      <span className='text-muted-foreground text-sm'>
                        {t('times')}
                      </span>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('Including failed requests, 0 = unlimited')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='ModelRequestRateLimitSuccessCount'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Max successful requests')}</FormLabel>
                  <FormControl>
                    <div className='flex items-center gap-2'>
                      <Input
                        type='number'
                        min={1}
                        max={100000000}
                        step={1}
                        {...field}
                        onChange={(e) =>
                          field.onChange(parseInt(e.target.value) || 1)
                        }
                      />
                      <span className='text-muted-foreground text-sm'>
                        {t('times')}
                      </span>
                    </div>
                  </FormControl>
                  <FormDescription>
                    {t('Only successful requests')}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>

          <FormField
            control={form.control}
            name='ModelRequestRateLimitGroup'
            render={({ field }) => (
              <FormItem>
                <div className='flex items-center justify-between'>
                  <FormLabel>{t('Group-based rate limits')}</FormLabel>
                  <Button
                    type='button'
                    variant='outline'
                    size='sm'
                    onClick={() => setUseVisualEditor(!useVisualEditor)}
                  >
                    {useVisualEditor ? (
                      <>
                        <Code2 className='mr-2 h-4 w-4' />
                        {t('JSON Mode')}
                      </>
                    ) : (
                      <>
                        <Palette className='mr-2 h-4 w-4' />
                        {t('Visual Mode')}
                      </>
                    )}
                  </Button>
                </div>
                <FormControl>
                  {useVisualEditor ? (
                    <RateLimitVisualEditor
                      value={field.value || ''}
                      onChange={field.onChange}
                    />
                  ) : (
                    <Textarea
                      rows={8}
                      placeholder={`{\n  "default": [200, 100],\n  "vip": [0, 1000]\n}`}
                      className='font-mono text-sm'
                      {...field}
                    />
                  )}
                </FormControl>
                {!useVisualEditor && (
                  <FormDescription>
                    <div className='space-y-1 text-xs'>
                      <p className='font-semibold'>{t('Format:')}</p>
                      <ul className='list-inside list-disc space-y-0.5 pl-2'>
                        <li>
                          {t('JSON object:')}{' '}
                          {`{"groupName": [maxRequests, maxSuccess]}`}
                        </li>
                        <li>
                          {t('Example:')}{' '}
                          {`{"default": [200, 100], "vip": [0, 1000]}`}
                        </li>
                        <li>
                          {t(
                            'maxRequests ≥ 0, maxSuccess ≥ 1, both ≤ 2,147,483,647'
                          )}
                        </li>
                        <li>
                          {t(
                            'Group config overrides global limits, shares the same period'
                          )}
                        </li>
                      </ul>
                    </div>
                  </FormDescription>
                )}
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name='ModelRequestRateLimitModels'
            render={({ field }) => (
              <FormItem className='border-border border-t pt-5'>
                <FormLabel>{t('Per-model rate limits')}</FormLabel>
                <FormControl>
                  <ModelRateLimitEditor
                    value={field.value}
                    onChange={field.onChange}
                  />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
