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
import { useTranslation } from 'react-i18next'
import { PublicLayout } from '@/components/layout'
import { CC_SWITCH_DOWNLOAD_URL } from '@/features/home/download-nav'

export function DownloadsPage() {
  const { t } = useTranslation()

  return (
    <PublicLayout
      showMainContainer={false}
      variant='editorial'
      showNotifications={false}
      showThemeSwitch={false}
    >
      <main className='mx-auto flex w-full max-w-3xl flex-col gap-5 px-6 pt-28 pb-20'>
        <header className='mb-2'>
          <h1 className='wharf-display text-3xl font-bold text-[var(--wharf-foam)]'>
            {t('Download')}
          </h1>
          <p className='mt-2 text-sm text-[var(--wharf-foam)]/80'>
            {t('CC Switch can be downloaded now. The other two addresses are not connected yet.')}
          </p>
        </header>

        <section
          id='cc-switch'
          className='rounded-xl border p-5'
        >
          <h2 className='text-lg font-semibold'>{t('CC Switch download')}</h2>
          <p className='mt-2 text-sm text-[var(--wharf-ink-3)]'>
            {t('Open the latest CC Switch release page.')}
          </p>
          <a
            className='mt-4 inline-flex h-9 items-center rounded-full bg-[#0a315c] px-4 text-sm font-semibold text-[#f4f8fb]'
            href={CC_SWITCH_DOWNLOAD_URL}
            target='_blank'
            rel='noopener noreferrer'
          >
            {t('Open download')}
          </a>
        </section>

        <section id='materials' className='rounded-xl border p-5'>
          <h2 className='text-lg font-semibold'>{t('Materials download')}</h2>
          <p className='mt-2 text-sm text-[var(--wharf-ink-3)]'>
            {t('The download address has not been connected yet.')}
          </p>
        </section>

        <section id='codex' className='rounded-xl border p-5'>
          <h2 className='text-lg font-semibold'>{t('Codex package')}</h2>
          <p className='mt-2 text-sm text-[var(--wharf-ink-3)]'>
            {t('The download address has not been connected yet.')}
          </p>
        </section>
      </main>
    </PublicLayout>
  )
}
