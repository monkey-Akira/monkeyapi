import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useForm, type Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { z } from 'zod'
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
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import { updateEmptyResponseRefundSetting } from '../api'
import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'

const schema = z.object({
  mode: z.enum(['off', 'observe', 'refund']),
  modelsText: z.string(),
  customResponseEnabled: z.boolean(),
  customResponseText: z.string().max(4000),
})

type Values = z.infer<typeof schema>

type Props = {
  defaultMode: string
  defaultModels: string
  defaultCustomResponseEnabled: boolean
  defaultCustomResponseText: string
}

function parseMode(value: string): Values['mode'] {
  return value === 'observe' || value === 'refund' ? value : 'off'
}

function parseModels(value: string): string[] {
  try {
    const parsed: unknown = JSON.parse(value)
    if (Array.isArray(parsed)) {
      return parsed.filter((item): item is string => typeof item === 'string' && item.trim() !== '')
    }
  } catch {
    // Older or malformed values are shown as an empty editable list.
  }
  return []
}

export function EmptyResponseRefundSection(props: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const mutation = useMutation({
    mutationFn: async (request: Parameters<typeof updateEmptyResponseRefundSetting>[0]) => {
      const response = await updateEmptyResponseRefundSetting(request)
      if (!response.success) {
        throw new Error(response.message || t('Failed to update setting'))
      }
      return response
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['system-options'] })
      toast.success(t('Setting updated successfully'))
    },
    onError: (error: Error) => toast.error(error.message || t('Failed to update setting')),
  })
  const form = useForm<Values>({
    resolver: zodResolver(schema) as unknown as Resolver<Values>,
    defaultValues: {
      mode: parseMode(props.defaultMode),
      modelsText: parseModels(props.defaultModels).join('\n'),
      customResponseEnabled: props.defaultCustomResponseEnabled,
      customResponseText: props.defaultCustomResponseText,
    },
  })
  const mode = form.watch('mode')
  const customResponseEnabled = form.watch('customResponseEnabled')

  async function onSubmit(values: Values) {
    const models = [...new Set(values.modelsText.split(/\r?\n/).map((item) => item.trim()).filter(Boolean))]
    await mutation.mutateAsync({
      mode: values.mode,
      models,
      custom_response_enabled: values.customResponseEnabled,
      custom_response_text: values.customResponseText,
    })
    form.reset(values)
  }

  return (
    <SettingsSection title={t('Empty Response Refund')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)} autoComplete='off'>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={mutation.isPending || form.formState.isSubmitting}
            isSaveDisabled={!form.formState.isDirty}
          />
          <FormField
            control={form.control}
            name='mode'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Refund Mode')}</FormLabel>
                <Select value={field.value} onValueChange={field.onChange}>
                  <FormControl>
                    <SelectTrigger className='w-full'><SelectValue /></SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value='off'>{t('Off')}</SelectItem>
                    <SelectItem value='observe'>{t('Observe Only')}</SelectItem>
                    <SelectItem value='refund'>{t('Automatic Refund')}</SelectItem>
                  </SelectContent>
                </Select>
                <FormDescription>
                  {mode === 'refund'
                    ? t('Matching text requests settle at zero and receive a separate refund log.')
                    : mode === 'observe'
                      ? t('Matching text requests are marked in logs, but charges are unchanged.')
                      : t('No empty-response detection or refunds are performed.')}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='modelsText'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Refund Models')}</FormLabel>
                <FormControl><Textarea rows={8} placeholder={t('Enter one exact model name per line')} {...field} /></FormControl>
                <FormDescription>{t('Only the original requested model name is matched. Image, audio, embedding, rerank, realtime, and structured requests are excluded.')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='customResponseEnabled'
            render={({ field }) => (
              <FormItem className='flex items-center justify-between gap-4 rounded-md border p-3'>
                <div className='space-y-1'>
                  <FormLabel>{t('Custom Empty Response Text')}</FormLabel>
                  <FormDescription>{t('Return text only for plain text automatic refunds; it does not add output tokens.')}</FormDescription>
                </div>
                <FormControl><Switch checked={field.value} onCheckedChange={field.onChange} disabled={mode !== 'refund'} /></FormControl>
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name='customResponseText'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Response Text')}</FormLabel>
                <FormControl><Textarea rows={4} maxLength={4000} disabled={mode !== 'refund' || !customResponseEnabled} {...field} /></FormControl>
                <FormDescription>{t('Structured output and forced tool calls are refunded without injecting text.')}</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}