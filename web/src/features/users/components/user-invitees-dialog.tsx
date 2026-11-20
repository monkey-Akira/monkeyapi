/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or (at your
option) any later version.
*/
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Ban, ChevronLeft, ChevronRight, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { StatusBadge } from '@/components/status-badge'
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
import { formatQuota, formatTimestamp } from '@/lib/format'
import { handleServerError } from '@/lib/handle-server-error'

import { disableUserInvitees, getUserInvitees, manageUser } from '../api'
import { USER_STATUS, USER_STATUSES } from '../constants'
import type { User } from '../types'
import { useUsers } from './users-provider'

const PAGE_SIZE = 20

type UserInviteesDialogProps = {
  user: User
}

export function UserInviteesDialog(props: UserInviteesDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { triggerRefresh } = useUsers()
  const [open, setOpen] = useState(false)
  const [page, setPage] = useState(1)
  const [disableId, setDisableId] = useState<number | null>(null)
  const [bulkDisableOpen, setBulkDisableOpen] = useState(false)

  const inviteesQuery = useQuery({
    queryKey: ['user-invitees', props.user.id, page],
    queryFn: async () => {
      const response = await getUserInvitees(props.user.id, {
        p: page,
        page_size: PAGE_SIZE,
      })
      if (!response.success) {
        throw new Error(response.message || t('Operation failed'))
      }
      return (
        response.data ?? {
          items: [],
          total: 0,
          page,
          page_size: PAGE_SIZE,
        }
      )
    },
    enabled: open,
  })

  const invalidateInvitees = () => {
    triggerRefresh()
    queryClient.invalidateQueries({ queryKey: ['user-invitees'] })
    queryClient.invalidateQueries({ queryKey: ['users'] })
  }

  const disableMutation = useMutation({
    mutationFn: (id: number) => manageUser(id, 'disable'),
    onSuccess: (response) => {
      if (!response.success) {
        handleServerError(response, t('Operation failed'))
        return
      }
      toast.success(t('User disabled successfully'))
      setDisableId(null)
      invalidateInvitees()
    },
    onError: (error) => handleServerError(error, t('Operation failed')),
  })

  const bulkDisableMutation = useMutation({
    mutationFn: () => disableUserInvitees(props.user.id),
    onSuccess: (response) => {
      if (!response.success) {
        handleServerError(response, t('Operation failed'))
        return
      }
      toast.success(t('User disabled successfully'))
      setBulkDisableOpen(false)
      invalidateInvitees()
    },
    onError: (error) => handleServerError(error, t('Operation failed')),
  })

  const invitees = inviteesQuery.data?.items ?? []
  const total = inviteesQuery.data?.total ?? 0
  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE))

  return (
    <>
      <Button
        type='button'
        size='xs'
        variant='outline'
        disabled={!props.user.aff_count}
        onClick={() => {
          setPage(1)
          setOpen(true)
        }}
      >
        <Users className='size-3' />
        {t('Invited')}: {props.user.aff_count ?? 0}
      </Button>
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className='sm:max-w-6xl'>
          <DialogHeader>
            <DialogTitle>
              {t('Invited Users')} - {props.user.username}
            </DialogTitle>
          </DialogHeader>
          <div className='flex flex-wrap items-center justify-between gap-3 text-sm'>
            <div className='text-muted-foreground flex flex-wrap gap-3'>
              <span>
                {t('Invitation Code')}: {props.user.aff_code || '-'}
              </span>
              <span>
                {t('Total')}: {total}
              </span>
              <StatusBadge
                label={t(props.user.aff_code_disabled ? 'Disabled' : 'Enabled')}
                variant={props.user.aff_code_disabled ? 'danger' : 'success'}
                copyable={false}
              />
            </div>
            <Button
              type='button'
              variant='destructive'
              size='sm'
              disabled={total === 0 || bulkDisableMutation.isPending}
              onClick={() => setBulkDisableOpen(true)}
            >
              <Ban className='size-3.5' />
              {t('Disable All')}
            </Button>
          </div>
          {inviteesQuery.isLoading ? (
            <div className='text-muted-foreground text-sm'>{t('Loading...')}</div>
          ) : invitees.length === 0 ? (
            <div className='text-muted-foreground text-sm'>{t('No data')}</div>
          ) : (
            <div className='overflow-x-auto'>
              <Table className='min-w-[920px]'>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t('Username')}</TableHead>
                    <TableHead>{t('Status')}</TableHead>
                    <TableHead>{t('Created At')}</TableHead>
                    <TableHead>{t('IP')}</TableHead>
                    <TableHead>{t('Last Login')}</TableHead>
                    <TableHead>{t('Quota')}</TableHead>
                    <TableHead>{t('Used Quota')}</TableHead>
                    <TableHead>{t('Requests')}</TableHead>
                    <TableHead className='text-right'>{t('Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {invitees.map((invitee) => {
                    const statusConfig =
                      USER_STATUSES[invitee.status as keyof typeof USER_STATUSES]
                    const canDisable =
                      invitee.status === USER_STATUS.ENABLED && invitee.role < 100
                    return (
                      <TableRow key={invitee.id}>
                        <TableCell>{invitee.id}</TableCell>
                        <TableCell>{invitee.username}</TableCell>
                        <TableCell>
                          <StatusBadge
                            label={t(statusConfig?.labelKey ?? 'Disabled')}
                            variant={statusConfig?.variant ?? 'neutral'}
                            copyable={false}
                          />
                        </TableCell>
                        <TableCell>
                          {formatTimestamp(invitee.created_at ?? 0)}
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
                        <TableCell className='text-right'>
                          <Button
                            type='button'
                            size='sm'
                            variant='destructive'
                            disabled={!canDisable || disableMutation.isPending}
                            onClick={() => setDisableId(invitee.id)}
                          >
                            <Ban className='size-3.5' />
                            {t('Disable')}
                          </Button>
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
              onClick={() => setPage((value) => Math.max(1, value - 1))}
              aria-label={t('Previous page')}
            >
              <ChevronLeft className='size-3.5' />
            </Button>
            <Button
              type='button'
              size='icon-sm'
              variant='outline'
              disabled={page >= pageCount || inviteesQuery.isFetching}
              onClick={() => setPage((value) => Math.min(pageCount, value + 1))}
              aria-label={t('Next page')}
            >
              <ChevronRight className='size-3.5' />
            </Button>
          </div>
        </DialogContent>
      </Dialog>
      <ConfirmDialog
        open={disableId !== null}
        onOpenChange={(value) => !value && setDisableId(null)}
        title={t('Confirm disable')}
        desc={t('Confirm disable')}
        confirmText={t('Disable')}
        destructive
        isLoading={disableMutation.isPending}
        handleConfirm={() => {
          if (disableId !== null) disableMutation.mutate(disableId)
        }}
      />
      <ConfirmDialog
        open={bulkDisableOpen}
        onOpenChange={setBulkDisableOpen}
        title={t('Confirm disable')}
        desc={t('Confirm disable')}
        confirmText={t('Disable All')}
        destructive
        isLoading={bulkDisableMutation.isPending}
        handleConfirm={() => bulkDisableMutation.mutate()}
      />
    </>
  )
}