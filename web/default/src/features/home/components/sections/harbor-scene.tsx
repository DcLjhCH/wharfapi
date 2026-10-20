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
import { useEffect, useRef } from 'react'

const PLATE = '/wharf/hero-base.jpg'
const WATER_MASK = '/wharf/water-mask.png'

const GULLS = [
  {
    src: '/wharf/gull-1.png',
    className: 'wharf-gull-a',
    style: { left: '8.11%', top: '22.69%', width: '11.91%' },
  },
  {
    src: '/wharf/gull-2.png',
    className: 'wharf-gull-b',
    style: { left: '23.83%', top: '43.14%', width: '6.35%' },
  },
  {
    src: '/wharf/gull-3.png',
    className: 'wharf-gull-c',
    style: { left: '80.96%', top: '37.46%', width: '6.54%' },
  },
] as const

function loadImage(src: string) {
  const image = new Image()
  image.decoding = 'async'
  image.src = src
  return image
}

export function HarborScene() {
  const canvasRef = useRef<HTMLCanvasElement>(null)

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d', { alpha: false })
    if (!ctx) return

    let alive = true
    let raf = 0
    let started = false
    const reduce = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    const plate = loadImage(PLATE)
    const mask = loadImage(WATER_MASK)

    const boot = () => {
      if (started || !plate.complete || !mask.complete) return
      if (!plate.naturalWidth || !mask.naturalWidth) return
      started = true

      const width = plate.naturalWidth
      const height = plate.naturalHeight
      canvas.width = width
      canvas.height = height
      ctx.drawImage(plate, 0, 0)
      if (reduce) return

      const plateData = ctx.getImageData(0, 0, width, height)
      const srcBytes = new Uint8ClampedArray(plateData.data)

      const scratch = document.createElement('canvas')
      scratch.width = width
      scratch.height = height
      const scratchCtx = scratch.getContext('2d', { willReadFrequently: true })
      if (!scratchCtx) return
      scratchCtx.drawImage(mask, 0, 0, width, height)
      const maskPx = scratchCtx.getImageData(0, 0, width, height).data
      const water = new Uint8Array(width * height)
      let horizon = height
      for (let y = 0; y < height; y++) {
        const row = y * width
        for (let x = 0; x < width; x++) {
          if (maskPx[(row + x) * 4] > 127) {
            water[row + x] = 1
            if (y < horizon) horizon = y
          }
        }
      }
      if (horizon >= height - 1) return

      const bandHeight = height - horizon
      const band = ctx.createImageData(width, bandHeight)
      const data = band.data
      const t0 = performance.now()

      const paint = (now: number) => {
        const time = (now - t0) / 1000
        const span = bandHeight
        for (let y = horizon; y < height; y++) {
          const depth = (y - horizon) / span
          const amp = 0.7 + depth * depth * 22
          const shift =
            Math.sin(time * 1.25 - y * 0.042) * amp +
            Math.sin(time * 0.62 + y * 0.015) * amp * 0.4
          const srcRow = y * width * 4
          const dstRow = (y - horizon) * width * 4
          const rowIndex = y * width
          for (let x = 0; x < width; x++) {
            const di = dstRow + x * 4
            const home = srcRow + x * 4
            if (!water[rowIndex + x]) {
              data[di] = srcBytes[home]
              data[di + 1] = srcBytes[home + 1]
              data[di + 2] = srcBytes[home + 2]
              data[di + 3] = 255
              continue
            }
            let sx = x + shift
            if (sx < 0) sx = 0
            else if (sx > width - 1) sx = width - 1
            const i0 = sx | 0
            const i1 = i0 >= width - 1 ? i0 : i0 + 1
            if (!water[rowIndex + i0] || !water[rowIndex + i1]) {
              data[di] = srcBytes[home]
              data[di + 1] = srcBytes[home + 1]
              data[di + 2] = srcBytes[home + 2]
              data[di + 3] = 255
              continue
            }
            const frac = sx - i0
            const inv = 1 - frac
            const a = srcRow + i0 * 4
            const b = srcRow + i1 * 4
            data[di] = srcBytes[a] * inv + srcBytes[b] * frac
            data[di + 1] = srcBytes[a + 1] * inv + srcBytes[b + 1] * frac
            data[di + 2] = srcBytes[a + 2] * inv + srcBytes[b + 2] * frac
            data[di + 3] = 255
          }
        }
        ctx.putImageData(band, 0, horizon)
      }

      const loop = (now: number) => {
        if (!alive) return
        paint(now)
        raf = requestAnimationFrame(loop)
      }

      const onVisibility = () => {
        if (!alive) return
        cancelAnimationFrame(raf)
        if (!document.hidden) raf = requestAnimationFrame(loop)
      }

      document.addEventListener('visibilitychange', onVisibility)
      raf = requestAnimationFrame(loop)
      cleanupVis = () => document.removeEventListener('visibilitychange', onVisibility)
    }

    let cleanupVis = () => {}
    plate.addEventListener('load', boot)
    mask.addEventListener('load', boot)
    boot()

    return () => {
      alive = false
      cancelAnimationFrame(raf)
      cleanupVis()
    }
  }, [])

  return (
    <div
      className='wharf-harbor'
      aria-hidden='true'
      style={{ backgroundImage: "url('/wharf/hero-base.jpg')" }}
    >
      <canvas ref={canvasRef} />
      {GULLS.map((gull) => (
        <img
          key={gull.src}
          className={`wharf-gull ${gull.className}`}
          src={gull.src}
          alt=''
          style={gull.style}
        />
      ))}
    </div>
  )
}
