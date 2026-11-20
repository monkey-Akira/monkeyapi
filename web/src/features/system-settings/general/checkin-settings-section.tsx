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
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

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
import { createCheckinSchema, type CheckinValues } from './lib/checkin-schema'

export function CheckinSettingsSection({
  defaultValues,
}: {
  defaultValues: CheckinValues
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()

  const form = useForm<CheckinValues>({
    resolver: zodResolver(createCheckinSchema(t)) as unknown as Resolver<
      CheckinValues
    >,
    defaultValues,
  })

  const { isDirty, isSubmitting } = form.formState
  const enabled = form.watch('enabled')

  async function onSubmit(values: CheckinValues) {
    try {
      await updateOption.mutateAsync({
        key: 'checkin_setting',
        value: JSON.stringify({
          enabled: values.enabled,
          min_quota: values.minQuota,
          max_quota: values.maxQuota,
          min_previous_day_requests: values.minPreviousDayRequests,
          min_single_redemption_quota: values.minSingleRedemptionQuota,
          last_10_percent_consume_quota: values.last10PercentConsumeQuota,
          twenty_to_ten_percent_consume_quota:
            values.twentyToTenPercentConsumeQuota,
        }),
      })
      form.reset(values)
    } catch {
      // The shared mutation displays the server error; keep edits for retry.
    }
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
          <p className='text-muted-foreground text-sm'>
            {t(
              'Check-in uses raw quota units, not currency. Yesterday consumption must reach 10. Consumption from 10 to below 20 awards 5; from 20 through 50 awards 5-7. These rules bypass the ordinary 15% cap.'
            )}
          </p>
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
            <div className='grid gap-6 sm:grid-cols-2'>
              <FormField
                control={form.control}
                name='minQuota'
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>{t('Minimum check-in quota')}</FormLabel>
                    <FormControl>
                      <Input
                        type='number'
                        min={1}
                        max={5}
                        disabled={isSubmitting}
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      {t(
                        'Ordinary minimum reward (1-5). Low-consumption rewards remain 5 or 5-7.'
                      )}
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
                        min={7}
                        max={2147483647}
                        disabled={isSubmitting}
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
              {([
                {
                  name: 'last10PercentConsumeQuota',
                  label: t('First-tier consumption threshold'),
                  description: t(
                    'Yesterday consumption needed for 90%-100% rewards, still subject to the 15% cap. This tier skips the jackpot lottery.'
                  ),
                },
                {
                  name: 'twentyToTenPercentConsumeQuota',
                  label: t('Second-tier consumption threshold'),
                  description: t(
                    'Yesterday consumption needed for 80%-below-90% rewards. Set 0 to disable. The 1% jackpot is checked first when eligible.'
                  ),
                },
                {
                  name: 'minPreviousDayRequests',
                  label: t('Minimum previous-day requests'),
                  description: t(
                    'Minimum number of consumption logs yesterday. Set 0 to disable this requirement.'
                  ),
                },
                {
                  name: 'minSingleRedemptionQuota',
                  label: t('Minimum single redemption quota'),
                  description: t(
                    'At least one previously used redemption code must reach this quota. Multiple codes are not added together. Set 0 to disable.'
                  ),
                },
              ] as const).map((item) => (
                <FormField
                  key={item.name}
                  control={form.control}
                  name={item.name}
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{item.label}</FormLabel>
                      <FormControl>
                        <Input
                          type='number'
                          min={0}
                          max={2147483647}
                          step={1}
                          disabled={isSubmitting}
                          {...field}
                        />
                      </FormControl>
                      <FormDescription>{item.description}</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              ))}
              <p className='text-muted-foreground text-sm sm:col-span-2'>
                {t(
                  'After consumption reaches four times the maximum reward, the jackpot chance is 1% and awards 90%-100% of the maximum. The jackpot bypasses the ordinary 15% cap. Consumption logs must be enabled and retained.'
                )}
              </p>
            </div>
          )}
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
