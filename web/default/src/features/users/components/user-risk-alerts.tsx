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
import { Ban, Check, Eye, RefreshCw, ShieldAlert } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { formatQuota, formatTimestamp } from '@/lib/format'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'
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
import {
  getUserRiskAlert,
  getUserRiskAlerts,
  manageUser,
  updateUserRiskAlertStatus,
} from '../api'
import type { RiskAlertStatus } from '../types'
import { useUsers } from './users-provider'

const riskStatusOptions: Array<RiskAlertStatus | 'all'> = [
  'open',
  'handled',
  'ignored',
  'all',
]

function getRiskStatusVariant(status: RiskAlertStatus) {
  if (status === 'open') return 'destructive'
  if (status === 'handled') return 'secondary'
  return 'outline'
}

function getRiskStatusLabel(status: RiskAlertStatus | 'all') {
  if (status === 'open') return 'Open'
  if (status === 'handled') return 'Handled'
  if (status === 'ignored') return 'Ignored'
  return 'All'
}

export function UserRiskAlerts() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const { triggerRefresh } = useUsers()
  const [status, setStatus] = useState<RiskAlertStatus | 'all'>('open')
  const [selectedAlertId, setSelectedAlertId] = useState<number | null>(null)

  const alertsQuery = useQuery({
    queryKey: ['user-risk-alerts', status],
    queryFn: async () => {
      const res = await getUserRiskAlerts({ status, page_size: 10 })
      if (!res.success) {
        throw new Error(res.message || t('Failed to load risk alerts'))
      }
      return res.data || { items: [], total: 0, page: 1, page_size: 10 }
    },
  })

  const detailQuery = useQuery({
    queryKey: ['user-risk-alert-detail', selectedAlertId],
    queryFn: async () => {
      if (!selectedAlertId) {
        throw new Error(t('Risk alert not selected'))
      }
      const res = await getUserRiskAlert(selectedAlertId)
      if (!res.success || !res.data) {
        throw new Error(res.message || t('Failed to load risk alert detail'))
      }
      return res.data
    },
    enabled: selectedAlertId !== null,
  })

  const statusMutation = useMutation({
    mutationFn: async (payload: { id: number; status: RiskAlertStatus }) => {
      const res = await updateUserRiskAlertStatus(payload.id, payload.status)
      if (!res.success) {
        throw new Error(res.message || t('Failed to update risk alert'))
      }
    },
    onSuccess: () => {
      toast.success(t('Risk alert updated'))
      queryClient.invalidateQueries({ queryKey: ['user-risk-alerts'] })
      if (selectedAlertId) {
        queryClient.invalidateQueries({
          queryKey: ['user-risk-alert-detail', selectedAlertId],
        })
      }
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Operation failed')
      )
    },
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
      if (selectedAlertId) {
        queryClient.invalidateQueries({
          queryKey: ['user-risk-alert-detail', selectedAlertId],
        })
      }
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : t('Operation failed')
      )
    },
  })

  const alerts = alertsQuery.data?.items || []
  const total = alertsQuery.data?.total || 0
  const detail = detailQuery.data

  return (
    <div className='border-border bg-background rounded-lg border'>
      <div className='flex flex-col gap-3 border-b p-3 sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex items-center gap-2'>
          <ShieldAlert className='text-destructive size-4' />
          <div>
            <div className='text-sm font-medium'>{t('Risk Alerts')}</div>
            <div className='text-muted-foreground text-xs'>
              {t('Same register or login IP across multiple accounts')}
            </div>
          </div>
          <Badge variant='outline'>{total}</Badge>
        </div>
        <div className='flex flex-wrap items-center gap-2'>
          {riskStatusOptions.map((option) => (
            <Button
              key={option}
              type='button'
              size='sm'
              variant={status === option ? 'secondary' : 'outline'}
              onClick={() => setStatus(option)}
            >
              {t(getRiskStatusLabel(option))}
            </Button>
          ))}
          <Button
            type='button'
            size='icon-sm'
            variant='outline'
            onClick={() => alertsQuery.refetch()}
            disabled={alertsQuery.isFetching}
            aria-label={t('Refresh')}
          >
            <RefreshCw
              className={cn(
                'size-3.5',
                alertsQuery.isFetching && 'animate-spin'
              )}
            />
          </Button>
        </div>
      </div>

      {alerts.length === 0 && (
        <div className='text-muted-foreground p-4 text-sm'>
          {alertsQuery.isLoading ? t('Loading...') : t('No risk alerts')}
        </div>
      )}

      {alerts.length > 0 && (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('IP')}</TableHead>
              <TableHead>{t('Users')}</TableHead>
              <TableHead>{t('Register')}</TableHead>
              <TableHead>{t('Login')}</TableHead>
              <TableHead>{t('Status')}</TableHead>
              <TableHead>{t('Updated At')}</TableHead>
              <TableHead className='text-right'>{t('Actions')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {alerts.map((alert) => (
              <TableRow key={alert.id}>
                <TableCell className='font-mono'>{alert.ip}</TableCell>
                <TableCell>{alert.user_count}</TableCell>
                <TableCell>{alert.register_count}</TableCell>
                <TableCell>{alert.login_count}</TableCell>
                <TableCell>
                  <Badge variant={getRiskStatusVariant(alert.status)}>
                    {t(getRiskStatusLabel(alert.status))}
                  </Badge>
                </TableCell>
                <TableCell>{formatTimestamp(alert.updated_at)}</TableCell>
                <TableCell>
                  <div className='flex justify-end gap-2'>
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      onClick={() => setSelectedAlertId(alert.id)}
                    >
                      <Eye className='size-3.5' />
                      {t('Details')}
                    </Button>
                    <Button
                      type='button'
                      size='sm'
                      variant='outline'
                      disabled={statusMutation.isPending}
                      onClick={() =>
                        statusMutation.mutate({
                          id: alert.id,
                          status: 'handled',
                        })
                      }
                    >
                      <Check className='size-3.5' />
                      {t('Handled')}
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <Dialog
        open={selectedAlertId !== null}
        onOpenChange={(open) => {
          if (!open) setSelectedAlertId(null)
        }}
      >
        <DialogContent className='sm:max-w-5xl'>
          <DialogHeader>
            <DialogTitle>{t('Risk Alert Detail')}</DialogTitle>
          </DialogHeader>
          {detailQuery.isLoading && (
            <div className='text-muted-foreground text-sm'>
              {t('Loading...')}
            </div>
          )}
          {detail && (
            <div className='space-y-4'>
              <div className='grid gap-2 text-sm sm:grid-cols-5'>
                <div>
                  <div className='text-muted-foreground text-xs'>{t('IP')}</div>
                  <div className='font-mono'>{detail.alert.ip}</div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Users')}
                  </div>
                  <div>{detail.alert.user_count}</div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Register')}
                  </div>
                  <div>{detail.alert.register_count}</div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Login')}
                  </div>
                  <div>{detail.alert.login_count}</div>
                </div>
                <div>
                  <div className='text-muted-foreground text-xs'>
                    {t('Status')}
                  </div>
                  <Badge variant={getRiskStatusVariant(detail.alert.status)}>
                    {t(getRiskStatusLabel(detail.alert.status))}
                  </Badge>
                </div>
              </div>

              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>ID</TableHead>
                    <TableHead>{t('Username')}</TableHead>
                    <TableHead>{t('Quota')}</TableHead>
                    <TableHead>{t('Register IP')}</TableHead>
                    <TableHead>{t('Last Login IP')}</TableHead>
                    <TableHead>{t('Last Login')}</TableHead>
                    <TableHead className='text-right'>{t('Actions')}</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {detail.users.map((user) => (
                    <TableRow key={user.id}>
                      <TableCell>{user.id}</TableCell>
                      <TableCell>
                        <div className='flex flex-col'>
                          <span>{user.username}</span>
                          <span className='text-muted-foreground text-xs'>
                            {user.email || user.display_name || '-'}
                          </span>
                        </div>
                      </TableCell>
                      <TableCell>{formatQuota(user.quota)}</TableCell>
                      <TableCell>
                        <span className='font-mono'>
                          {user.register_ip || '-'}
                        </span>
                        {user.register_matched && (
                          <Badge className='ml-2' variant='destructive'>
                            {t('Match')}
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        <span className='font-mono'>
                          {user.last_login_ip || '-'}
                        </span>
                        {user.login_matched && (
                          <Badge className='ml-2' variant='destructive'>
                            {t('Match')}
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell>
                        {formatTimestamp(user.last_login_at)}
                      </TableCell>
                      <TableCell>
                        <div className='flex justify-end'>
                          <Button
                            type='button'
                            size='sm'
                            variant='destructive'
                            disabled={
                              user.status !== 1 || disableMutation.isPending
                            }
                            onClick={() => disableMutation.mutate(user.id)}
                          >
                            <Ban className='size-3.5' />
                            {t('Disable')}
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>

              <div className='flex justify-end gap-2'>
                <Button
                  type='button'
                  variant='outline'
                  disabled={statusMutation.isPending}
                  onClick={() =>
                    statusMutation.mutate({
                      id: detail.alert.id,
                      status: 'ignored',
                    })
                  }
                >
                  {t('Ignore')}
                </Button>
                <Button
                  type='button'
                  disabled={statusMutation.isPending}
                  onClick={() =>
                    statusMutation.mutate({
                      id: detail.alert.id,
                      status: 'handled',
                    })
                  }
                >
                  <Check className='size-3.5' />
                  {t('Mark handled')}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
