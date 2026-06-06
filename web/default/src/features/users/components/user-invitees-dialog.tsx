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
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Ban, ChevronLeft, ChevronRight, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatQuota, formatTimestamp } from '@/lib/format'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { StatusBadge } from '@/components/status-badge'
import { getUserInvitees, manageUser } from '../api'
import { USER_STATUS, USER_STATUSES } from '../constants'
import type { User } from '../types'
import { useUsers } from './users-provider'

const pageSize = 20

function getUserStatusConfig(user: User) {
  return (
    USER_STATUSES[user.status as keyof typeof USER_STATUSES] ??
    USER_STATUSES[USER_STATUS.DISABLED]
  )
}

type UserInviteesDialogProps = {
  user: User
}

export function UserInviteesDialog(props: UserInviteesDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { triggerRefresh } = useUsers()
  const [open, setOpen] = useState(false)
  const [page, setPage] = useState(1)

  const affCount = props.user.aff_count || 0
  const inviteesQuery = useQuery({
    queryKey: ['user-invitees', props.user.id, page],
    queryFn: async () => {
      const res = await getUserInvitees(props.user.id, {
        p: page,
        page_size: pageSize,
      })
      if (!res.success) {
        throw new Error(res.message || t('Failed to load invited users'))
      }
      return res.data || { items: [], total: 0, page: 1, page_size: pageSize }
    },
    enabled: open,
  })

  const disableMutation = useMutation({
    mutationFn: async (userId: number) => {
      const res = await manageUser(userId, 'disable')
      if (!res.success) {
        throw new Error(res.message || t('Failed to disable user'))
      }
    },
    onSuccess: () => {
      toast.success(t('User disabled'))
      triggerRefresh()
      queryClient.invalidateQueries({ queryKey: ['users'] })
      queryClient.invalidateQueries({ queryKey: ['user-invitees'] })
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Operation failed')
      )
    },
  })

  const invitees = inviteesQuery.data?.items || []
  const total = inviteesQuery.data?.total || 0
  const pageCount = Math.max(1, Math.ceil(total / pageSize))

  return (
    <>
      <Button
        type='button'
        size='xs'
        variant='outline'
        disabled={affCount === 0}
        onClick={() => {
          setPage(1)
          setOpen(true)
        }}
      >
        <Users className='size-3' />
        {t('Invited')}: {affCount}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className='sm:max-w-6xl'>
          <DialogHeader>
            <DialogTitle>
              {t('Invited Users')} - {props.user.username}
            </DialogTitle>
          </DialogHeader>

          <div className='text-muted-foreground flex flex-wrap gap-3 text-xs'>
            <span>
              {t('Inviter ID')}: {props.user.id}
            </span>
            <span>
              {t('Total invited users')}: {total}
            </span>
            <span>
              {t('Invitation revenue')}:{' '}
              {formatQuota(props.user.aff_history_quota || 0)}
            </span>
          </div>

          {inviteesQuery.isLoading && (
            <div className='text-muted-foreground text-sm'>
              {t('Loading...')}
            </div>
          )}

          {!inviteesQuery.isLoading && invitees.length === 0 && (
            <div className='text-muted-foreground text-sm'>
              {t('No invited users')}
            </div>
          )}

          {invitees.length > 0 && (
            <div className='overflow-x-auto'>
              <Table className='min-w-[980px]'>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t('Username')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('Created At')}</TableHead>
                    <TableHead>{t('Register IP')}</TableHead>
                    <TableHead>{t('Last Login IP')}</TableHead>
                    <TableHead>{t('Quota')}</TableHead>
                    <TableHead>{t('Used Quota')}</TableHead>
                    <TableHead>{t('Requests')}</TableHead>
                    <TableHead className='text-right'>{t('Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invitees.map((invitee) => {
                    const statusConfig = getUserStatusConfig(invitee)
                    return (
                      <TableRow key={invitee.id}>
                        <TableCell>{invitee.id}</TableCell>
                        <TableCell>
                          <div className='flex flex-col'>
                            <span>{invitee.username}</span>
                            <span className='text-muted-foreground text-xs'>
                              {invitee.email || invitee.display_name || '-'}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <StatusBadge
                            label={t(statusConfig.labelKey)}
                            variant={statusConfig.variant}
                            copyable={false}
                          />
                        </TableCell>
                        <TableCell>
                          {formatTimestamp(invitee.created_at || 0)}
                        </TableCell>
                        <TableCell className='font-mono'>
                          {invitee.register_ip || '-'}
                        </TableCell>
                        <TableCell className='font-mono'>
                          {invitee.last_login_ip || '-'}
                        </TableCell>
                        <TableCell>{formatQuota(invitee.quota)}</TableCell>
                        <TableCell>{formatQuota(invitee.used_quota)}</TableCell>
                        <TableCell>{invitee.request_count}</TableCell>
                        <TableCell>
                          <div className='flex justify-end'>
                            <Button
                              type='button'
                              size='sm'
                              variant='destructive'
                              disabled={
                                invitee.status !== USER_STATUS.ENABLED ||
                                disableMutation.isPending
                              }
                              onClick={() => disableMutation.mutate(invitee.id)}
                            >
                              <Ban className='size-3.5' />
                              {t('Disable')}
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          )}

          <div className='flex items-center justify-end gap-2'>
            <span className='text-muted-foreground text-xs'>
              {page} / {pageCount}
            </span>
            <Button
              type='button'
              size='icon-sm'
              variant='outline'
              disabled={page <= 1 || inviteesQuery.isFetching}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
              aria-label={t('Previous page')}
            >
              <ChevronLeft className='size-3.5' />
            </Button>
            <Button
              type='button'
              size='icon-sm'
              variant='outline'
              disabled={page >= pageCount || inviteesQuery.isFetching}
              onClick={() =>
                setPage((current) => Math.min(pageCount, current + 1))
              }
              aria-label={t('Next page')}
            >
              <ChevronRight className='size-3.5' />
            </Button>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
