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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { toast } from 'sonner'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { SettingsPageProvider } from '../../components/settings-page-context'
import { CheckinSettingsSection } from '../checkin-settings-section'
import type { CheckinValues } from '../lib/checkin-schema'

const defaults: CheckinValues = {
  enabled: true,
  minQuota: 5,
  maxQuota: 50,
  minPreviousDayRequests: 0,
  minSingleRedemptionQuota: 0,
  last10PercentConsumeQuota: 700,
  twentyToTenPercentConsumeQuota: 550,
}

async function renderSettings(values = defaults) {
  function Fixture() {
    const [container, setContainer] = useState<HTMLDivElement | null>(null)
    return (
      <>
        <div ref={setContainer} />
        <SettingsPageProvider actionsContainer={container}>
          <CheckinSettingsSection defaultValues={values} />
        </SettingsPageProvider>
      </>
    )
  }
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  const router = createRouter({
    routeTree: createRootRoute({ component: Fixture }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  render(
    <QueryClientProvider client={client}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  return screen.findByRole('spinbutton', {
    name: 'First-tier consumption threshold',
  })
}

beforeEach(() => {
  vi.spyOn(api, 'put').mockResolvedValue({ data: { success: true } })
  vi.spyOn(toast, 'success').mockImplementation(() => 0)
  vi.spyOn(toast, 'error').mockImplementation(() => 0)
})

afterEach(() => vi.restoreAllMocks())

test('saving tier edits sends all check-in settings in one request', async () => {
  const input = await renderSettings()
  fireEvent.change(input, { target: { value: '800' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save check-in settings' }))
  await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1))
  expect(api.put).toHaveBeenCalledWith('/api/option/', {
    key: 'checkin_setting',
    value: JSON.stringify({
      enabled: true,
      min_quota: 5,
      max_quota: 50,
      min_previous_day_requests: 0,
      min_single_redemption_quota: 0,
      last_10_percent_consume_quota: 800,
      twenty_to_ten_percent_consume_quota: 550,
    }),
  })
  await waitFor(() => expect(toast.success).toHaveBeenCalledTimes(1))
})

test('a first tier whose cap cannot cover its reward cannot be saved', async () => {
  const input = await renderSettings()
  fireEvent.change(input, { target: { value: '250' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save check-in settings' }))
  await waitFor(() => expect(input).toHaveAttribute('aria-invalid', 'true'))
  expect(api.put).not.toHaveBeenCalled()
})

test('zero disables the second tier and is preserved in the saved settings', async () => {
  await renderSettings()
  fireEvent.change(screen.getByRole('spinbutton', {
    name: 'Second-tier consumption threshold',
  }), { target: { value: '0' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save check-in settings' }))
  await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1))
  const request = vi.mocked(api.put).mock.calls[0][1] as { value: string }
  expect(JSON.parse(request.value).twenty_to_ten_percent_consume_quota).toBe(0)
})

test('an invalid legacy configuration can still be disabled', async () => {
  await renderSettings({ ...defaults, minQuota: 1000, maxQuota: 10000 })
  fireEvent.click(screen.getByRole('switch', { name: 'Enable check-in feature' }))
  fireEvent.click(screen.getByRole('button', { name: 'Save check-in settings' }))
  await waitFor(() => expect(api.put).toHaveBeenCalledTimes(1))
  const request = vi.mocked(api.put).mock.calls[0][1] as { value: string }
  expect(JSON.parse(request.value).enabled).toBe(false)
})

test('server rejection preserves edits for retry without a success message', async () => {
  vi.mocked(api.put).mockResolvedValueOnce({
    data: { success: false, message: 'Check-in settings could not be saved' },
  })
  const input = await renderSettings()
  fireEvent.change(input, { target: { value: '800' } })
  fireEvent.click(screen.getByRole('button', { name: 'Save check-in settings' }))
  await waitFor(() => expect(toast.error).toHaveBeenCalled())
  expect(toast.success).not.toHaveBeenCalled()
  expect(input).toHaveValue(800)
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Save check-in settings' })).toBeEnabled()
  )
})

