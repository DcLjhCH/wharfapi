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
import { Link } from '@tanstack/react-router'
import { HarborScene } from './harbor-scene'
import { MODEL_MARKS } from './model-marks'
import { useTranslation } from 'react-i18next'
import { useSystemConfig } from '@/hooks/use-system-config'

interface MarketingHomeProps {
  isAuthenticated?: boolean
}

const UPSTREAMS = [
  { name: 'GPT', vendor: 'OpenAI', wide: false },
  { name: 'Claude', vendor: 'Anthropic', wide: false },
  { name: 'Gemini', vendor: 'Google', wide: false },
  { name: 'Grok', vendor: 'xAI', wide: false },
  { name: 'GLM', vendor: 'Z.ai', wide: false },
  { name: 'DeepSeek', vendor: 'DeepSeek AI', wide: true },
] as const

const CAPABILITIES = [
  {
    index: '01',
    title: 'One compatible entry',
    body: 'Chat and completion calls stay on an OpenAI-style API, for a quick prompt, a shared team key, or a batch job.',
  },
  {
    index: '02',
    title: 'Change one parameter',
    body: 'Point an existing OpenAI client at this gateway and change the model name. You do not keep a separate client for each upstream.',
  },
  {
    index: '03',
    title: 'Route across channels',
    body: 'Add upstream channels, groups, and priority in the console. Requests follow those rules instead of being hardcoded in the app.',
  },
  {
    index: '04',
    title: 'Bill from usage logs',
    body: 'Calls can update quota, balance, and logs. The price a user sees comes from the pricing page and the rates configured here.',
  },
  {
    index: '05',
    title: 'Separate keys and limits',
    body: 'Issue tokens with their own quota, model access, and group, so other people can call the API without the admin account.',
  },
] as const

function ArrowIcon() {
  return (
    <svg viewBox='0 0 24 24' aria-hidden='true'>
      <path d='M5 12h14' />
      <path d='m13 6 6 6-6 6' />
    </svg>
  )
}

export function MarketingHome(props: MarketingHomeProps) {
  const { t } = useTranslation()
  const { systemName } = useSystemConfig()
  const brand = systemName || 'WharfAPI'
  const year = new Date().getFullYear()
  const primaryTo = props.isAuthenticated ? '/dashboard' : '/sign-up'
  const loop = [...UPSTREAMS, ...UPSTREAMS]

  return (
    <div className='wharf-home'>
      <section className='wharf-hero' aria-labelledby='home-title'>
        <div className='wharf-stage'>
          <HarborScene />
          <div className='wharf-mark'>
            <svg
              className='wharf-mark-logo'
              viewBox='0 0 160 156'
              aria-hidden='true'
            >
              <path fill='#fff' d='M80 2l6 13H74L80 2z' />
              <path
                fill='#fff'
                fillRule='evenodd'
                d='M67 15h26v20H67V15zm4 3h4v14h-4V18zm7 0h4v14h-4V18zm7 0h4v14h-4V18z'
              />
              <path fill='#fff' d='M61 35h38v5H61z' />
              <path
                fill='#fff'
                fillRule='evenodd'
                d='M71 40l-6 68h30l-6-68H71zm7 38h6v18h-6V78z'
              />
              <path
                fill='#fff'
                d='M16 112c18-16 30-12 42-2 9 8 14 6 22-2 9-9 18-12 30-5 16 9 24 5 36-6 9-8 18-7 26 3 3 5 1 11-5 12H14c-7-5-4-12 2 0z'
              />
              <path
                fill='#fff'
                d='M4 130c22-12 36-2 50 8 12 8 20 5 30-2 13-9 24-7 38 2 13 8 22 5 34-3 5 7 2 12-5 12H6c-8-5-6-15-2-17z'
              />
            </svg>
            <p className='wharf-mark-name'>wharfapi.com</p>
            <p className='wharf-mark-tag'>YOUR API TRANSFER STATION</p>
          </div>
          <div className='wharf-hero-copy'>
            <h1 id='home-title'>{t('One interface for mainstream models')}</h1>
            <div className='wharf-hero-actions'>
              <Link className='wharf-hero-primary' to={primaryTo}>
                <span>
                  {props.isAuthenticated ? t('Open console') : t('Start now')}
                </span>
                <span className='wharf-hero-primary-icon'>
                  <ArrowIcon />
                </span>
              </Link>
              <Link className='wharf-hero-secondary' to='/pricing'>
                {t('View pricing')}
              </Link>
            </div>
          </div>
        </div>
      </section>

      <section className='wharf-hero-note'>
        <p className='wharf-lede'>
          {t(
            'The same OpenAI-compatible entry. Switch models with one parameter, and keep channel routing and usage billing in this console.'
          )}
        </p>
        <ul className='wharf-pills'>
          <li>{t('OpenAI-compatible API')}</li>
          <li>{t('Channel routing')}</li>
          <li>{t('Usage-based billing')}</li>
        </ul>
      </section>

      <section className='wharf-section wharf-models' aria-labelledby='home-models-title'>
        <h2 id='home-models-title'>
          {t('Switch models without rewriting the client')}
        </h2>
        <p className='wharf-models-lead'>
          {t(
            'Name the model you want. Which upstream answers depends on the channels configured in the console.'
          )}
        </p>
        <div className='wharf-model-panel'>
          <ul className='wharf-model-list'>
            {UPSTREAMS.map((model) => {
              const Mark = MODEL_MARKS[model.name]
              return (
                <li key={model.name}>
                  <span
                    className={
                      model.wide ? 'wharf-model-mark is-wide' : 'wharf-model-mark'
                    }
                  >
                    <Mark />
                  </span>
                  <span className='wharf-model-name'>{model.name}</span>
                  <span className='wharf-model-vendor'>{model.vendor}</span>
                </li>
              )
            })}
          </ul>
        </div>
      </section>

      <section className='wharf-section wharf-capabilities' aria-labelledby='home-capabilities-title'>
        <div className='wharf-capabilities-inner'>
          <div className='wharf-capabilities-head'>
            <p className='wharf-kicker'>{t('Capabilities')}</p>
            <h2 id='home-capabilities-title'>
              {t('Routing, quotas, and keys stay in one console')}
            </h2>
            <p className='wharf-capabilities-lead'>
              {t(
                'This gateway forwards OpenAI-compatible requests, records usage, and lets you assign keys. Prices and available models follow the channels configured here.'
              )}
            </p>
          </div>
          <ol className='wharf-capability-list'>
            {CAPABILITIES.map((item) => (
              <li key={item.index}>
                <span className='wharf-capability-index'>{item.index}</span>
                <div>
                  <h3>{t(item.title)}</h3>
                  <p>{t(item.body)}</p>
                </div>
              </li>
            ))}
          </ol>
        </div>
      </section>

      <section className='wharf-marquee' aria-label={t('Upstream families')}>
        <h2>{t('Upstream families')}</h2>
        <div className='wharf-marquee-viewport'>
          <div className='wharf-marquee-track'>
            {[0, 1].map((copy) => (
              <div className='wharf-marquee-group' key={copy}>
                {loop.map((model, index) => (
                  <span className='wharf-marquee-item' key={`${copy}-${model.name}-${index}`}>
                    <strong>{model.name}</strong>
                    <span>{model.vendor}</span>
                  </span>
                ))}
              </div>
            ))}
          </div>
        </div>
      </section>

      <footer className='wharf-footer'>
        <Link to='/'>{brand}</Link>
        <nav aria-label='Footer'>
          <Link to='/pricing'>{t('Pricing')}</Link>
          <Link to='/about'>{t('About')}</Link>
          <Link to={props.isAuthenticated ? '/dashboard' : '/sign-in'}>
            {t('Console')}
          </Link>
        </nav>
        <span>
          © {year} {brand}
        </span>
      </footer>
    </div>
  )
}
