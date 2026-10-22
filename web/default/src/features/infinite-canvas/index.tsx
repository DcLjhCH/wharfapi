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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ExternalLink } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { fetchTokenKey, getApiKeys } from '@/features/keys/api'

async function loadPreferredApiKey(): Promise<string | null> {
  const result = await getApiKeys({ p: 1, size: 20 })
  const items = result.success ? (result.data?.items ?? []) : []
  const preferred = items.find((item) => item.status === 1) ?? items[0]
  if (!preferred) return null
  const keyResult = await fetchTokenKey(preferred.id)
  const key = keyResult.success ? keyResult.data?.key?.trim() : ''
  if (!key) return null
  return key.startsWith('sk-') ? key : `sk-${key}`
}

function reserveEmbedButtonSpace(frame: HTMLIFrameElement) {
  const doc = frame.contentDocument
  if (!doc?.head || doc.getElementById('wharf-canvas-embed-style')) return
  const style = doc.createElement('style')
  style.id = 'wharf-canvas-embed-style'
  style.textContent =
    'header.sticky > div{padding-right:10.75rem!important}'
  doc.head.appendChild(style)
}

export function InfiniteCanvasWindow() {
  const { t } = useTranslation()
  const [src, setSrc] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    ;(async () => {
      const params = new URLSearchParams()
      params.set('silent', '1')
      params.set('baseUrl', window.location.origin)
      try {
        const apiKey = await loadPreferredApiKey()
        if (apiKey) params.set('apiKey', apiKey)
      } catch {
        // The canvas still opens; the user can fill the key in its settings.
      }
      if (!cancelled) setSrc(`/canvas-app/?${params.toString()}`)
    })()
    return () => {
      cancelled = true
    }
  }, [])

  return (
    <div className='relative flex h-full min-h-0 flex-1 flex-col'>
      {src ? (
        <iframe
          title={t('Infinite Canvas')}
          src={src}
          className='h-full min-h-0 w-full flex-1 border-0 bg-[#0c0c0c]'
          allow='clipboard-read; clipboard-write; fullscreen'
          onLoad={(event) => reserveEmbedButtonSpace(event.currentTarget)}
        />
      ) : (
        <div className='text-muted-foreground flex h-full items-center justify-center text-sm'>
          {t('Infinite Canvas')}
        </div>
      )}
      <Button
        variant='outline'
        size='sm'
        className='absolute top-3 right-4 z-10 h-8 gap-1.5 border-white/15 bg-zinc-950/90 text-zinc-100 shadow-none hover:bg-zinc-900 hover:text-white'
        render={<a href={src ?? '/canvas-app/'} target='_blank' rel='noreferrer' />}
      >
        <ExternalLink data-icon='inline-start' />
        {t('Open in new window')}
      </Button>
    </div>
  )
}
