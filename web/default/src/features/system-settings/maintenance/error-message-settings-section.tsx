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
import { Plus, RotateCcw, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import {
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'

type ErrorMessageRow = {
  id: string
  code: string
  message: string
}

const DEFAULT_ERROR_MESSAGES: Record<string, string> = {
  insufficient_user_quota:
    '\u989d\u5ea6\u4e0d\u8db3\uff0c\u8bf7\u5145\u503c\u540e\u518d\u8bd5\u3002',
  pre_consume_token_quota_failed:
    '\u989d\u5ea6\u9884\u6263\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  'channel:no_available_key':
    '\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5\u3002',
  'channel:invalid_key':
    '\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458\u5904\u7406\u3002',
  'channel:model_mapped_error':
    '\u6a21\u578b\u914d\u7f6e\u5f02\u5e38\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458\u5904\u7406\u3002',
  do_request_failed:
    '\u8fde\u63a5\u4e0a\u6e38\u670d\u52a1\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  bad_response_status_code:
    '\u4e0a\u6e38\u670d\u52a1\u54cd\u5e94\u5f02\u5e38\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  bad_response_body:
    '\u4e0a\u6e38\u670d\u52a1\u8fd4\u56de\u5f02\u5e38\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  model_not_found:
    '\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\u3002',
  prompt_blocked:
    '\u8bf7\u6c42\u5185\u5bb9\u672a\u901a\u8fc7\u5b89\u5168\u68c0\u67e5\uff0c\u8bf7\u8c03\u6574\u540e\u91cd\u8bd5\u3002',
  sensitive_words_detected:
    '\u8bf7\u6c42\u5185\u5bb9\u5305\u542b\u654f\u611f\u8bcd\uff0c\u8bf7\u8c03\u6574\u540e\u91cd\u8bd5\u3002',
  invalid_request:
    '\u8bf7\u6c42\u53c2\u6570\u6709\u8bef\uff0c\u8bf7\u68c0\u67e5\u540e\u91cd\u8bd5\u3002',
}

function parseMappings(value: string): Record<string, string> {
  try {
    const parsed = JSON.parse(value || '{}')
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return {}
    }
    return Object.fromEntries(
      Object.entries(parsed).filter(
        ([code, message]) =>
          code.trim() !== '' && typeof message === 'string'
      )
    ) as Record<string, string>
  } catch {
    return {}
  }
}

function mappingsToRows(mappings: Record<string, string>): ErrorMessageRow[] {
  return Object.entries(mappings).map(([code, message], index) => ({
    id: `${code}-${index}`,
    code,
    message,
  }))
}

function rowsToMappings(rows: ErrorMessageRow[]): Record<string, string> {
  return rows.reduce<Record<string, string>>((acc, row) => {
    const code = row.code.trim()
    const message = row.message.trim()
    if (code && message) {
      acc[code] = message
    }
    return acc
  }, {})
}

function stableJSONString(value: unknown) {
  return JSON.stringify(value)
}

export function ErrorMessageSettingsSection({
  defaultValues,
}: {
  defaultValues: {
    enabled: boolean
    mappings: string
  }
}) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const initialMappings = useMemo(
    () => parseMappings(defaultValues.mappings),
    [defaultValues.mappings]
  )
  const [enabled, setEnabled] = useState(defaultValues.enabled)
  const [rows, setRows] = useState(() => mappingsToRows(initialMappings))

  useEffect(() => {
    setEnabled(defaultValues.enabled)
    setRows(mappingsToRows(initialMappings))
  }, [defaultValues.enabled, initialMappings])

  const mappings = rowsToMappings(rows)
  const isDirty =
    enabled !== defaultValues.enabled ||
    stableJSONString(mappings) !== stableJSONString(initialMappings)

  function updateRow(
    id: string,
    patch: Partial<Pick<ErrorMessageRow, 'code' | 'message'>>
  ) {
    setRows((current) =>
      current.map((row) => (row.id === id ? { ...row, ...patch } : row))
    )
  }

  function addRow() {
    setRows((current) => [
      ...current,
      {
        id: `new-${Date.now()}`,
        code: '',
        message: '',
      },
    ])
  }

  function removeRow(id: string) {
    setRows((current) => current.filter((row) => row.id !== id))
  }

  function resetDefaults() {
    setRows(mappingsToRows(DEFAULT_ERROR_MESSAGES))
  }

  async function onSave() {
    const codes = rows.map((row) => row.code.trim()).filter(Boolean)
    if (new Set(codes).size !== codes.length) {
      toast.error(t('Error codes must be unique'))
      return
    }
    if (rows.some((row) => row.code.trim() && !row.message.trim())) {
      toast.error(t('Message cannot be empty'))
      return
    }

    const updates: Array<{ key: string; value: string }> = []
    if (enabled !== defaultValues.enabled) {
      updates.push({
        key: 'error_message_setting.enabled',
        value: String(enabled),
      })
    }
    if (stableJSONString(mappings) !== stableJSONString(initialMappings)) {
      updates.push({
        key: 'error_message_setting.mappings',
        value: JSON.stringify(mappings),
      })
    }
    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }
    for (const update of updates) {
      await updateOption.mutateAsync(update)
    }
  }

  const disabled = updateOption.isPending

  return (
    <SettingsSection title={t('Error Response Messages')}>
      <div className='grid min-w-0 gap-5'>
        <SettingsSwitchItem>
          <SettingsSwitchContent>
            <div className='text-sm font-medium'>
              {t('Enable custom error messages')}
            </div>
            <div className='text-muted-foreground text-xs'>
              {t(
                'Replace only the returned error.message by error code. Logs and error codes are kept unchanged.'
              )}
            </div>
          </SettingsSwitchContent>
          <Switch
            checked={enabled}
            onCheckedChange={setEnabled}
            disabled={disabled}
          />
        </SettingsSwitchItem>

        <div className='min-w-0 rounded-lg border'>
          <div className='bg-muted/30 hidden grid-cols-[minmax(160px,0.8fr)_minmax(220px,1.2fr)_44px] gap-2 border-b px-3 py-2 text-xs font-medium md:grid'>
            <span>{t('Error code')}</span>
            <span>{t('Returned message')}</span>
            <span />
          </div>
          <div className='grid gap-2 p-3'>
            {rows.map((row) => (
              <div
                key={row.id}
                className='grid gap-2 md:grid-cols-[minmax(160px,0.8fr)_minmax(220px,1.2fr)_44px]'
              >
                <Input
                  value={row.code}
                  placeholder='bad_response_status_code'
                  onChange={(event) =>
                    updateRow(row.id, { code: event.target.value })
                  }
                  disabled={disabled}
                />
                <Textarea
                  value={row.message}
                  rows={1}
                  placeholder={t('Message returned to users')}
                  onChange={(event) =>
                    updateRow(row.id, { message: event.target.value })
                  }
                  disabled={disabled}
                />
                <Button
                  type='button'
                  variant='ghost'
                  size='icon'
                  onClick={() => removeRow(row.id)}
                  disabled={disabled}
                  title={t('Delete')}
                >
                  <Trash2 />
                </Button>
              </div>
            ))}
            {rows.length === 0 ? (
              <div className='text-muted-foreground py-6 text-center text-sm'>
                {t('No custom error messages configured.')}
              </div>
            ) : null}
          </div>
        </div>

        <div className='flex flex-wrap items-center gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={addRow}
            disabled={disabled}
          >
            <Plus />
            {t('Add mapping')}
          </Button>
          <Button
            type='button'
            variant='outline'
            onClick={resetDefaults}
            disabled={disabled}
          >
            <RotateCcw />
            {t('Use default mappings')}
          </Button>
          <Button
            type='button'
            onClick={onSave}
            disabled={!isDirty || disabled}
          >
            {t('Save error messages')}
          </Button>
        </div>
      </div>
    </SettingsSection>
  )
}
