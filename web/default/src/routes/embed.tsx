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
import { createFileRoute } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'
import { PublicLayout } from '@/components/layout'

type EmbedSearch = {
  url?: string
  title?: string
}

function isEmbeddableUrl(value: string | undefined): value is string {
  if (!value) return false
  try {
    const url = new URL(value)
    return url.protocol === 'http:' || url.protocol === 'https:'
  } catch {
    return false
  }
}

function EmbedPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const title = search.title || t('URL')

  if (!isEmbeddableUrl(search.url)) {
    return (
      <PublicLayout>
        <div className='flex min-h-[60vh] items-center justify-center px-4 text-center'>
          <div className='space-y-2'>
            <h1 className='text-2xl font-semibold'>
              {t('Must be a valid URL')}
            </h1>
            <p className='text-muted-foreground'>
              {t('Provide a valid URL starting with http:// or https://')}
            </p>
          </div>
        </div>
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <iframe
        src={search.url}
        className='mt-16 h-[calc(100svh-4rem)] w-full border-0'
        title={title}
      />
    </PublicLayout>
  )
}

export const Route = createFileRoute('/embed')({
  validateSearch: (search: Record<string, unknown>): EmbedSearch => ({
    url: typeof search.url === 'string' ? search.url : undefined,
    title: typeof search.title === 'string' ? search.title : undefined,
  }),
  component: EmbedPage,
})
