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
import { z } from 'zod'
import { useForm, type Resolver } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
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
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

function getSchema(t: TFunction) {
  return z
    .object({
      enabled: z.boolean(),
      minQuota: z.coerce.number().int().min(0),
      maxQuota: z.coerce.number().int().min(0),
      minPreviousDayRequests: z.coerce.number().int().min(0),
      minSingleRedemptionQuota: z.coerce.number().int().min(0),
      last10PercentConsumeQuota: z.coerce.number().int().min(0),
      twentyToTenPercentConsumeQuota: z.coerce.number().int().min(0),
    })
    .superRefine((values, ctx) => {
      if (
        values.twentyToTenPercentConsumeQuota > 0 &&
        values.last10PercentConsumeQuota <=
          values.twentyToTenPercentConsumeQuota
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['last10PercentConsumeQuota'],
          message: t(
            'The last 10% threshold must be greater than the 20%-10% threshold.'
          ),
        })
      }
    })
}

type Values = z.infer<ReturnType<typeof getSchema>>

export function CheckinSettingsSection({
  defaultValues,
}: {
  defaultValues: {
    enabled: boolean
    minQuota: number
    maxQuota: number
    minPreviousDayRequests: number
    minSingleRedemptionQuota: number
    last10PercentConsumeQuota: number
    twentyToTenPercentConsumeQuota: number
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<Values>({
    resolver: zodResolver(getSchema(t)) as unknown as Resolver<Values>,
    defaultValues: {
      enabled: defaultValues.enabled,
      minQuota: defaultValues.minQuota,
      maxQuota: defaultValues.maxQuota,
      minPreviousDayRequests: defaultValues.minPreviousDayRequests,
      minSingleRedemptionQuota: defaultValues.minSingleRedemptionQuota,
      last10PercentConsumeQuota: defaultValues.last10PercentConsumeQuota,
      twentyToTenPercentConsumeQuota:
        defaultValues.twentyToTenPercentConsumeQuota,
    },
  })

  const { isDirty, isSubmitting } = form.formState
  const enabled = form.watch('enabled')

  async function onSubmit(values: Values) {
    const updates: Array<{ key: string; value: string }> = []

    if (values.enabled !== defaultValues.enabled) {
      updates.push({
        key: 'checkin_setting.enabled',
        value: String(values.enabled),
      })
    }

    if (values.minQuota !== defaultValues.minQuota) {
      updates.push({
        key: 'checkin_setting.min_quota',
        value: String(values.minQuota),
      })
    }

    if (values.maxQuota !== defaultValues.maxQuota) {
      updates.push({
        key: 'checkin_setting.max_quota',
        value: String(values.maxQuota),
      })
    }

    if (
      values.minPreviousDayRequests !== defaultValues.minPreviousDayRequests
    ) {
      updates.push({
        key: 'checkin_setting.min_previous_day_requests',
        value: String(values.minPreviousDayRequests),
      })
    }

    if (
      values.minSingleRedemptionQuota !== defaultValues.minSingleRedemptionQuota
    ) {
      updates.push({
        key: 'checkin_setting.min_single_redemption_quota',
        value: String(values.minSingleRedemptionQuota),
      })
    }

    if (
      values.last10PercentConsumeQuota !==
      defaultValues.last10PercentConsumeQuota
    ) {
      updates.push({
        key: 'checkin_setting.last_10_percent_consume_quota',
        value: String(values.last10PercentConsumeQuota),
      })
    }

    if (
      values.twentyToTenPercentConsumeQuota !==
      defaultValues.twentyToTenPercentConsumeQuota
    ) {
      updates.push({
        key: 'checkin_setting.twenty_to_ten_percent_consume_quota',
        value: String(values.twentyToTenPercentConsumeQuota),
      })
    }

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }

    form.reset(values)
  }

  return (
    <SettingsSection title={t('Check-in Settings')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending || isSubmitting}
            isSaveDisabled={!isDirty}
            saveLabel='Save check-in settings'
          />
          <FormField
            control={form.control}
            name='enabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable check-in feature')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Allow users to check in daily for random quota rewards'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending || isSubmitting}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          {enabled && (
            <div className='grid gap-6 sm:grid-cols-4'>
              <FormField
                control={form.control}
                name='minQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum check-in quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder={t('1000')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Minimum quota amount awarded for check-in')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='maxQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Maximum check-in quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder={t('10000')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t('Maximum quota amount awarded for check-in')}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='last10PercentConsumeQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Last 10% reward threshold')}</FormLabel>
                    <FormControl>
                      <Input type='number' min={1} {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Yesterday consumption quota required to enter the last 10% reward range'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='twentyToTenPercentConsumeQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('20%-10% reward threshold')}</FormLabel>
                    <FormControl>
                      <Input type='number' min={0} {...field} />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Yesterday consumption quota required to enter the 20%-10% reward range. Set 0 to disable this tier.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='minPreviousDayRequests'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum previous-day requests')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder={t('0')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Users must reach this many successful calls yesterday before they can check in. Set 0 to disable this requirement.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name='minSingleRedemptionQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('Minimum single redemption quota')}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={0}
                        placeholder={t('0')}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Users must have used at least one redemption code with this quota amount. Set 0 to disable this requirement.'
                      )}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
