import type { SVGProps } from 'react'

/**
 * Kite Logo — 风筝 SVG 图标
 */
export function KiteIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <svg
      xmlns='http://www.w3.org/2000/svg'
      viewBox='0 0 1024 1024'
      fill='currentColor'
      {...props}
    >
      <path d='M519.52 417.888l-144.8 272.512L690.4 960l103.968-401.952-274.848-140.16z' />
      <path d='M408.032 64l-249.92 441.344 191.712 163.616 159.872-282.368 293.056 139.584 63.136-244.128L408.032 64z' />
    </svg>
  )
}
