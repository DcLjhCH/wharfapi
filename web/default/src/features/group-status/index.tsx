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
import { useQuery } from '@tanstack/react-query'
import { RefreshCw } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { SectionPageLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { api } from '@/lib/api'
import dayjs from '@/lib/dayjs'
import { cn } from '@/lib/utils'

type HistoryStatus = 'empty' | 'success' | 'failure' | 'mixed'
type StatusView = 'simple' | 'detailed'

type GroupStatusModel = {
  name: string
  requests: number
  availability: number | null
  avg_response_seconds: number | null
  avg_first_token_seconds: number | null
}

type GroupStatusGroup = {
  name: string
  description: string
  ratio: number
  has_enabled_models: boolean
  model_count: number
  models_without_requests: number
  success_count: number
  failure_count: number
  availability: number | null
  avg_response_seconds: number | null
  avg_first_token_seconds: number | null
  cache_rate: number | null
  last_sample_at: number | null
  history: HistoryStatus[]
  models: GroupStatusModel[]
}

type GroupStatusData = {
  window_start: number
  window_end: number
  groups: GroupStatusGroup[]
}

const HISTORY_CLASS: Record<HistoryStatus, string> = {
  empty: 'bg-muted-foreground/25',
  success: 'bg-success',
  failure: 'bg-destructive',
  mixed: 'bg-warning',
}

export function GroupStatusPage() {
  const { t } = useTranslation()
  const [view, setView] = useState<StatusView>('simple')
  const query = useQuery({
    queryKey: ['group-status'],
    queryFn: async () => {
      const response = await api.get('/api/group-status', {
        skipBusinessError: true,
        skipErrorHandler: true,
      })
      if (!response.data?.success) {
        throw new Error(
          response.data?.message || t('Failed to load group status')
        )
      }
      return response.data.data as GroupStatusData
    },
    refetchInterval: 60_000,
    retry: false,
  })

  const groups = query.data?.groups ?? []

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Group Status')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <div className='bg-muted/60 flex items-center rounded-lg border p-0.5'>
          <ViewButton
            active={view === 'simple'}
            onClick={() => setView('simple')}
          >
            {t('Simple')}
          </ViewButton>
          <ViewButton
            active={view === 'detailed'}
            onClick={() => setView('detailed')}
          >
            {t('Detailed')}
          </ViewButton>
        </div>
        <Button
          variant='outline'
          size='sm'
          onClick={() => void query.refetch()}
          disabled={query.isFetching}
        >
          <RefreshCw className={cn(query.isFetching && 'animate-spin')} />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <div className='flex flex-col gap-4'>
          <StatusNotice />
          <StatusBody
            loading={query.isLoading}
            errorMessage={
              query.isError ? errorMessage(query.error, t) : null
            }
            groups={groups}
            detailed={view === 'detailed'}
            windowStart={query.data?.window_start ?? 0}
            onRetry={() => void query.refetch()}
          />
        </div>
      </SectionPageLayout.Content>
    </SectionPageLayout>
  )
}

function errorMessage(error: unknown, t: (key: string) => string): string {
  if (error instanceof Error && error.message) return error.message
  return t('Failed to load group status')
}

function StatusBody(props: {
  loading: boolean
  errorMessage: string | null
  groups: GroupStatusGroup[]
  detailed: boolean
  windowStart: number
  onRetry: () => void
}) {
  const { t } = useTranslation()
  if (props.loading) return <StatusSkeleton />
  if (props.errorMessage != null) {
    return <StatusError message={props.errorMessage} onRetry={props.onRetry} />
  }
  if (props.groups.length === 0) {
    return (
      <div className='text-muted-foreground rounded-xl border border-dashed px-4 py-10 text-center text-sm'>
        {t('This account has no groups to view')}
      </div>
    )
  }
  return (
    <div className='grid gap-3 xl:grid-cols-2'>
      {props.groups.map((group) => (
        <GroupCard
          key={group.name}
          group={group}
          detailed={props.detailed}
          windowStart={props.windowStart}
        />
      ))}
    </div>
  )
}

function StatusNotice() {
  const { t } = useTranslation()
  return (
    <div className='bg-muted/40 text-muted-foreground space-y-1 rounded-xl border px-4 py-3 text-sm'>
      <p>{t('View recent availability and response time for each group')}</p>
      <p>
        {t(
          'Numbers come from real requests in the last 30 minutes. A dash means there were no requests, not that the group is healthy.'
        )}
      </p>
      <p>
        {t(
          'Statistics include requests from every user in the group, but not the request contents.'
        )}
      </p>
    </div>
  )
}

function StatusError(props: { message: string; onRetry: () => void }) {
  const { t } = useTranslation()
  return (
    <div className='border-destructive/40 bg-destructive/5 flex flex-col items-start gap-3 rounded-xl border px-4 py-5'>
      <div>
        <p className='text-sm font-medium'>{t('Failed to load group status')}</p>
        <p className='text-muted-foreground mt-1 text-sm'>{props.message}</p>
      </div>
      <Button variant='outline' size='sm' onClick={props.onRetry}>
        {t('Retry')}
      </Button>
    </div>
  )
}

function StatusSkeleton() {
  return (
    <div className='grid gap-3 xl:grid-cols-2'>
      {Array.from({ length: 2 }, (_, index) => (
        <div key={index} className='space-y-3 rounded-xl border p-4'>
          <Skeleton className='h-5 w-32' />
          <Skeleton className='h-4 w-48' />
          <div className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
            {Array.from({ length: 4 }, (_, cell) => (
              <Skeleton key={cell} className='h-12' />
            ))}
          </div>
          <Skeleton className='h-3 w-full' />
        </div>
      ))}
    </div>
  )
}

function GroupCard(props: {
  group: GroupStatusGroup
  detailed: boolean
  windowStart: number
}) {
  const { t } = useTranslation()
  const { group } = props
  const requests = group.success_count + group.failure_count
  const description =
    group.description && group.description !== group.name
      ? group.description
      : null

  return (
    <section className='bg-card flex flex-col gap-4 rounded-xl border p-4'>
      <div className='flex items-start justify-between gap-3'>
        <div className='min-w-0'>
          <h3 className='truncate text-base font-semibold'>{group.name}</h3>
          {description != null && (
            <p className='text-muted-foreground mt-0.5 text-sm'>{description}</p>
          )}
        </div>
        <div className='text-muted-foreground shrink-0 text-xs'>
          {t('Ratio')} {formatRatio(group.ratio)}
        </div>
      </div>

      <dl className='grid grid-cols-2 gap-2 sm:grid-cols-4'>
        <Metric
          label={t('Has enabled models')}
          value={group.has_enabled_models ? t('Enabled') : t('No enabled models')}
        />
        <Metric label={t('Model count')} value={String(group.model_count)} />
        <Metric label={t('Cache rate')} value={formatPercent(group.cache_rate)} />
        <Metric
          label={t('Availability')}
          value={requests === 0 ? '-' : formatPercent(group.availability)}
        />
        <Metric
          label={t('Average response')}
          value={requests === 0 ? '-' : formatSeconds(group.avg_response_seconds)}
        />
        <Metric
          label={t('Average first token')}
          value={
            requests === 0 ? '-' : formatSeconds(group.avg_first_token_seconds)
          }
        />
        <Metric label={t('Requests')} value={String(requests)} />
        <Metric
          label={t('Last sample')}
          value={formatSampleTime(group.last_sample_at)}
        />
      </dl>

      <History
        history={group.history}
        windowStart={props.windowStart}
      />

      {group.models_without_requests > 0 && (
        <p className='text-muted-foreground text-xs'>
          {t('{{count}} enabled models had no requests in this window', {
            count: group.models_without_requests,
          })}
        </p>
      )}

      {props.detailed && <ModelDetails models={group.models} />}
    </section>
  )
}

function Metric(props: { label: string; value: string }) {
  return (
    <div className='bg-muted/30 rounded-lg px-2.5 py-2'>
      <dt className='text-muted-foreground text-[11px]'>{props.label}</dt>
      <dd className='mt-0.5 truncate text-sm font-medium'>{props.value}</dd>
    </div>
  )
}

function History(props: { history: HistoryStatus[]; windowStart: number }) {
  const { t } = useTranslation()
  const cells = Array.from({ length: 30 }, (_, index) => {
    const status = normalizeHistory(props.history?.[index])
    return { index, status }
  })
  return (
    <div className='space-y-2'>
      <div className='flex gap-1' aria-hidden='false'>
        {cells.map((cell) => (
          <span
            key={cell.index}
            title={`${minuteLabel(props.windowStart, cell.index)} ${historyLabel(t, cell.status)}`}
            className={cn('h-7 min-w-0 flex-1 rounded-sm', HISTORY_CLASS[cell.status])}
          />
        ))}
      </div>
      <div className='text-muted-foreground flex flex-wrap gap-x-3 gap-y-1 text-[11px]'>
        <Legend status='empty' />
        <Legend status='success' />
        <Legend status='failure' />
        <Legend status='mixed' />
      </div>
    </div>
  )
}

function Legend(props: { status: HistoryStatus }) {
  const { t } = useTranslation()
  return (
    <span className='inline-flex items-center gap-1.5'>
      <span className={cn('size-2.5 rounded-sm', HISTORY_CLASS[props.status])} />
      {historyLabel(t, props.status)}
    </span>
  )
}

function ModelDetails(props: { models: GroupStatusModel[] }) {
  const { t } = useTranslation()
  if (props.models.length === 0) {
    return null
  }
  return (
    <div className='overflow-x-auto'>
      <table className='w-full min-w-[36rem] text-left text-sm'>
        <thead className='text-muted-foreground text-xs'>
          <tr className='border-b'>
            <th className='py-2 pr-3 font-medium'>{t('Models')}</th>
            <th className='py-2 pr-3 font-medium'>{t('Availability')}</th>
            <th className='py-2 pr-3 font-medium'>{t('Average response')}</th>
            <th className='py-2 pr-3 font-medium'>{t('Average first token')}</th>
            <th className='py-2 font-medium'>{t('Requests')}</th>
          </tr>
        </thead>
        <tbody>
          {props.models.map((model) => (
            <tr key={model.name || '-'} className='border-b last:border-0'>
              <td className='py-2 pr-3 font-medium'>{model.name || '-'}</td>
              <td className='py-2 pr-3'>
                {model.requests === 0 ? '-' : formatPercent(model.availability)}
              </td>
              <td className='py-2 pr-3'>
                {formatSeconds(model.avg_response_seconds)}
              </td>
              <td className='py-2 pr-3'>
                {formatSeconds(model.avg_first_token_seconds)}
              </td>
              <td className='py-2'>{model.requests}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ViewButton(props: {
  active: boolean
  onClick: () => void
  children: string
}) {
  return (
    <Button
      type='button'
      size='sm'
      variant={props.active ? 'secondary' : 'ghost'}
      className='h-7 px-2.5'
      onClick={props.onClick}
    >
      {props.children}
    </Button>
  )
}

function normalizeHistory(status: string | undefined): HistoryStatus {
  if (
    status === 'success' ||
    status === 'failure' ||
    status === 'mixed' ||
    status === 'empty'
  ) {
    return status
  }
  return 'empty'
}

function historyLabel(
  t: (key: string) => string,
  status: HistoryStatus
): string {
  switch (status) {
    case 'success':
      return t('All succeeded')
    case 'failure':
      return t('All failed')
    case 'mixed':
      return t('Mixed results')
    default:
      return t('No requests in this minute')
  }
}

function formatPercent(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '-'
  return new Intl.NumberFormat(undefined, {
    style: 'percent',
    maximumFractionDigits: 1,
  }).format(value)
}

function formatSeconds(value: number | null | undefined): string {
  if (value == null || Number.isNaN(value)) return '-'
  return `${new Intl.NumberFormat(undefined, {
    maximumFractionDigits: value < 10 ? 2 : 1,
  }).format(value)}s`
}

function formatRatio(value: number): string {
  if (!Number.isFinite(value)) return '-'
  return new Intl.NumberFormat(undefined, { maximumFractionDigits: 4 }).format(
    value
  )
}

function formatSampleTime(value: number | null | undefined): string {
  if (value == null) return '-'
  return dayjs.unix(value).format('MM-DD HH:mm:ss')
}

function minuteLabel(start: number, index: number): string {
  if (!start) return ''
  return dayjs.unix(start + index * 60).format('HH:mm')
}
