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
import type { TFunction } from 'i18next'
import { z } from 'zod'

export function createCheckinSchema(t: TFunction) {
  const amount = z.coerce.number().int().min(0).max(2147483647)
  return z
    .object({
      enabled: z.boolean(),
      minQuota: amount,
      maxQuota: amount,
      minPreviousDayRequests: amount,
      minSingleRedemptionQuota: amount,
      last10PercentConsumeQuota: amount,
      twentyToTenPercentConsumeQuota: amount,
    })
    .superRefine((values, ctx) => {
      if (!values.enabled) return
      if (values.minQuota < 1 || values.minQuota > 5) {
        ctx.addIssue({
          code: 'custom',
          path: ['minQuota'],
          message: t('The ordinary minimum reward must be between 1 and 5.'),
        })
      }
      if (values.maxQuota < 7) {
        ctx.addIssue({
          code: 'custom',
          path: ['maxQuota'],
          message: t('The maximum reward must be at least 7.'),
        })
      }
      const topMinimum = Math.ceil((values.maxQuota * 9) / 10)
      const secondMinimum = Math.ceil((values.maxQuota * 8) / 10)
      if (
        values.last10PercentConsumeQuota <= 50 ||
        Math.floor((values.last10PercentConsumeQuota * 15) / 100) < topMinimum
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['last10PercentConsumeQuota'],
          message: t(
            'The first-tier threshold must exceed 50 and its 15% cap must cover the 90% reward minimum.'
          ),
        })
      }
      if (
        values.twentyToTenPercentConsumeQuota !== 0 &&
        (values.twentyToTenPercentConsumeQuota <= 50 ||
          values.twentyToTenPercentConsumeQuota >=
            values.last10PercentConsumeQuota ||
          Math.floor((values.twentyToTenPercentConsumeQuota * 15) / 100) <
            secondMinimum)
      ) {
        ctx.addIssue({
          code: 'custom',
          path: ['twentyToTenPercentConsumeQuota'],
          message: t(
            'Set the second tier to 0 to disable it; otherwise it must exceed 50, be below the first tier, and its 15% cap must cover the 80% reward minimum.'
          ),
        })
      }
    })
}

export type CheckinValues = z.infer<ReturnType<typeof createCheckinSchema>>
