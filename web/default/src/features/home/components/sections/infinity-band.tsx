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
import { useRef, useState } from 'react'

function lemniscatePath() {
  const steps = 180
  const parts: string[] = []
  for (let i = 0; i <= steps; i += 1) {
    const t = (i / steps) * Math.PI * 2
    const s = Math.sin(t)
    const c = Math.cos(t)
    const denom = 1 + s * s
    const x = 500 + (c / denom) * 440
    const y = 250 + ((s * c) / denom) * 520
    parts.push(`${i === 0 ? 'M' : 'L'}${x.toFixed(1)} ${y.toFixed(1)}`)
  }
  return parts.join(' ')
}

const BAND_PATH = lemniscatePath()

export function InfinityBand() {
  const tilt = useRef({ x: 0, y: 0, rx: 0, ry: 0 })
  const [drag, setDrag] = useState({ active: false, rx: 0, ry: 0 })

  function reduced() {
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches
  }

  return (
    <div
      className={
        drag.active ? 'wharf-band-tilt is-dragging' : 'wharf-band-tilt'
      }
      style={{
        transform: `rotateX(${drag.rx}deg) rotateY(${drag.ry}deg)`,
      }}
      onPointerDown={(event) => {
        if (event.pointerType === 'touch' || reduced()) return
        event.currentTarget.setPointerCapture(event.pointerId)
        tilt.current = {
          x: event.clientX,
          y: event.clientY,
          rx: drag.rx,
          ry: drag.ry,
        }
        setDrag((current) => ({ ...current, active: true }))
      }}
      onPointerMove={(event) => {
        if (!tilt.current.x && !drag.active) return
        if (!event.currentTarget.hasPointerCapture(event.pointerId)) return
        const dx = event.clientX - tilt.current.x
        const dy = event.clientY - tilt.current.y
        setDrag({
          active: true,
          rx: Math.max(-28, Math.min(28, tilt.current.rx - dy * 0.08)),
          ry: Math.max(-36, Math.min(36, tilt.current.ry + dx * 0.1)),
        })
      }}
      onPointerUp={(event) => {
        if (event.currentTarget.hasPointerCapture(event.pointerId)) {
          event.currentTarget.releasePointerCapture(event.pointerId)
        }
        tilt.current = { x: 0, y: 0, rx: 0, ry: 0 }
        setDrag({ active: false, rx: 0, ry: 0 })
      }}
    >
      <svg className='wharf-band' viewBox='0 0 1000 500' aria-hidden='true'>
        <defs>
          <filter id='wharf-band-blur' x='-30%' y='-30%' width='160%' height='160%'>
            <feGaussianBlur stdDeviation='7' />
          </filter>
        </defs>
        <path className='wharf-band-track' pathLength='1000' d={BAND_PATH} />
        <path className='wharf-band-soft' pathLength='1000' d={BAND_PATH} />
        <path className='wharf-band-rim' pathLength='1000' d={BAND_PATH} />
        <path className='wharf-band-core' pathLength='1000' d={BAND_PATH} />
      </svg>
    </div>
  )
}
