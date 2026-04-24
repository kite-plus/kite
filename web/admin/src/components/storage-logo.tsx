import { Cloud, HardDrive, Server } from 'lucide-react'

import { cn } from '@/lib/utils'

// Storage-backend brand marks are inlined as SVG components. The prior
// implementation fetched them at runtime from two external CDNs (svgl.app
// for AWS / Cloudflare / Google Cloud / DigitalOcean, and the Iconify API
// for Simple Icons slugs), which was wrong on three counts for a
// self-hosted product:
//
//   1. Every render of the admin storage page fired multiple cross-origin
//      requests; on slow or restricted networks the brand logos visibly
//      popped in a beat after the cards drew, and on air-gapped installs
//      they never appeared.
//   2. Pulling an entire Iconify icon pack (or registering bulk packs via
//      `@iconify-json/*`) to reach a handful of brand glyphs would add a
//      large chunk of unused SVG data to the bundle.
//   3. The component shipped a dual-img light/dark hack plus a `failed`
//      fallback state — both existed only because those CDN requests
//      could fail or flash.
//
// This rewrite bundles only the ~11 SVGs actually referenced (roughly
// 5 KB gzipped) and drops the `@iconify/react` dependency from this
// file. Paths come from the upstream sources the old component targeted
// (Simple Icons CC0 1.0 for most brands; svgl.app renderings of the
// official brand guidelines for the multi-color marks). Generic slots
// that never had a real brand (S3-compatible, Tencent Cloud, FTP,
// local filesystem) fall through to `lucide-react` icons already in
// the bundle.

type LogoVendor =
  | 's3'
  | 'aws'
  | 'cloudflare'
  | 'aliyun'
  | 'tencent'
  | 'huawei'
  | 'baidu'
  | 'gcp'
  | 'backblaze'
  | 'minio'
  | 'wasabi'
  | 'do'
  | 'ftp'
  | 'local'

type MarkComponent = (props: { className?: string }) => React.ReactElement

interface VendorMeta {
  label: string
  // Brand accent kept on the meta so the wrapping tile can still tint a
  // single-color mark via `style={{ color }}` (the old behavior). For
  // multi-color marks the wrapper color is ignored by the SVG, but the
  // value is still useful elsewhere (swatches, storage dashboards).
  color: string
  Mark: MarkComponent
}

// ─── Single-color path helper ────────────────────────────────────────
// Most Simple-Icons marks are one <path> on a 24×24 viewBox. Wrapping
// them through this factory cuts the repetition and guarantees
// `fill="currentColor"` so the wrapping tile's color tints them.

function makeSingleColorMark(
  path: string,
  viewBox = '0 0 24 24'
): MarkComponent {
  return ({ className }) => (
    <svg
      viewBox={viewBox}
      xmlns="http://www.w3.org/2000/svg"
      fill="currentColor"
      aria-hidden="true"
      className={className}
    >
      <path d={path} />
    </svg>
  )
}

// ─── Single-color marks (tinted by wrapper color) ────────────────────

const MarkAlibabaCloud = makeSingleColorMark(
  'M3.996 4.517h5.291L8.01 6.324 4.153 7.506a1.668 1.668 0 0 0-1.165 1.601v5.786a1.668 1.668 0 0 0 1.165 1.6l3.857 1.183 1.277 1.807H3.996A3.996 3.996 0 0 1 0 15.487V8.513a3.996 3.996 0 0 1 3.996-3.996m16.008 0h-5.291l1.277 1.807 3.857 1.182c.715.227 1.17.889 1.165 1.601v5.786a1.668 1.668 0 0 1-1.165 1.6l-3.857 1.183-1.277 1.807h5.291A3.996 3.996 0 0 0 24 15.487V8.513a3.996 3.996 0 0 0-3.996-3.996m-4.007 8.345H8.002v-1.804h7.995Z'
)

const MarkBaidu = makeSingleColorMark(
  'M9.154 0C7.71 0 6.54 1.658 6.54 3.707c0 2.051 1.171 3.71 2.615 3.71 1.446 0 2.614-1.659 2.614-3.71C11.768 1.658 10.6 0 9.154 0zm7.025.594C14.86.58 13.347 2.589 13.2 3.927c-.187 1.745.25 3.487 2.179 3.735 1.933.25 3.175-1.806 3.422-3.364.252-1.555-.995-3.364-2.362-3.674a1.218 1.218 0 0 0-.261-.03zM3.582 5.535a2.811 2.811 0 0 0-.156.008c-2.118.19-2.428 3.24-2.428 3.24-.287 1.41.686 4.425 3.297 3.864 2.617-.561 2.262-3.68 2.183-4.362-.125-1.018-1.292-2.773-2.896-2.75zm16.534 1.753c-2.308 0-2.617 2.119-2.617 3.616 0 1.43.121 3.425 2.988 3.362 2.867-.063 2.553-3.238 2.553-3.988 0-.745-.62-2.99-2.924-2.99zm-8.264 2.478c-1.424.014-2.708.925-3.323 1.947-1.118 1.868-2.863 3.05-3.112 3.363-.25.309-3.61 2.116-2.864 5.42.746 3.301 3.365 3.237 3.365 3.237s1.93.19 4.171-.31c2.24-.495 4.17.123 4.17.123s5.233 1.748 6.665-1.616c1.43-3.364-.808-5.109-.808-5.109s-2.99-2.306-4.736-4.798c-1.072-1.665-2.348-2.268-3.528-2.257zm-2.234 3.84l1.542.024v8.197H7.758c-1.47-.291-2.055-1.292-2.13-1.462-.072-.173-.488-.976-.268-2.343.635-2.049 2.447-2.196 2.447-2.196h1.81zm3.964 2.39v3.881c.096.413.612.488.612.488h1.614v-4.343h1.689v5.782h-3.915c-1.517-.39-1.59-1.465-1.59-1.465v-4.317zm-5.458 1.147c-.66.197-.978.708-1.05.928-.076.22-.247.78-.1 1.269.294 1.095 1.248 1.144 1.248 1.144h1.37v-3.34z'
)

const MarkHuawei = makeSingleColorMark(
  'M3.67 6.14S1.82 7.91 1.72 9.78v.35c.08 1.51 1.22 2.4 1.22 2.4 1.83 1.79 6.26 4.04 7.3 4.55 0 0 .06.03.1-.01l.02-.04v-.04C7.52 10.8 3.67 6.14 3.67 6.14zM9.65 18.6c-.02-.08-.1-.08-.1-.08l-7.38.26c.8 1.43 2.15 2.53 3.56 2.2.96-.25 3.16-1.78 3.88-2.3.06-.05.04-.09.04-.09zm.08-.78C6.49 15.63.21 12.28.21 12.28c-.15.46-.2.9-.21 1.3v.07c0 1.07.4 1.82.4 1.82.8 1.69 2.34 2.2 2.34 2.2.7.3 1.4.31 1.4.31.12.02 4.4 0 5.54 0 .05 0 .08-.05.08-.05v-.06c0-.03-.03-.05-.03-.05zM9.06 3.19a3.42 3.42 0 00-2.57 3.15v.41c.03.6.16 1.05.16 1.05.66 2.9 3.86 7.65 4.55 8.65.05.05.1.03.1.03a.1.1 0 00.06-.1c1.06-10.6-1.11-13.42-1.11-13.42-.32.02-1.19.23-1.19.23zm8.299 2.27s-.49-1.8-2.44-2.28c0 0-.57-.14-1.17-.22 0 0-2.18 2.81-1.12 13.43.01.07.06.08.06.08.07.03.1-.03.1-.03.72-1.03 3.9-5.76 4.55-8.64 0 0 .36-1.4.02-2.34zm-2.92 13.07s-.07 0-.09.05c0 0-.01.07.03.1.7.51 2.85 2 3.88 2.3 0 0 .16.05.43.06h.14c.69-.02 1.9-.37 3-2.26l-7.4-.25zm7.83-8.41c.14-2.06-1.94-3.97-1.94-3.98 0 0-3.85 4.66-6.67 10.8 0 0-.03.08.02.13l.04.01h.06c1.06-.53 5.46-2.77 7.28-4.54 0 0 1.15-.93 1.21-2.42zm1.52 2.14s-6.28 3.37-9.52 5.55c0 0-.05.04-.03.11 0 0 .03.06.07.06 1.16 0 5.56 0 5.67-.02 0 0 .57-.02 1.27-.29 0 0 1.56-.5 2.37-2.27 0 0 .73-1.45.17-3.14z'
)

const MarkBackblaze = makeSingleColorMark(
  'M9.3108.0003c.6527 1.3502 1.5666 4.0812-1.3887 7.1738-1.8096 1.8796-3.078 3.8487-2.3496 6.0644.3642 1.1037 1.1864 2.5079 2.8867 2.7852.6107.1008 1.3425-.0006 1.7403-.1406 2.4538-.8544 2.098-3.4138 1.5546-5.0469-.07-.2129-.1915-.7333-.2363-.9238-.3726-1.6023.776-2.6562 1.129-3.8047.028-.0925.0534-.1819.0702-.2715.042-.21.067-.423.0781-.6387 0-1.8264-.9882-2.6303-1.7754-3.5996C10.1794.5643 9.3107.0003 9.3107.0003Zm6.2754 6.0175s-.709.3366-1.2188.8829c-.4454.4818-.8635.8789-1.2949 1.8593-.028.14-.0518.2863-.0742.4375-.2325 1.6416 1.1473 3.1446.7187 5.1895-.112.535-.3554.7123-.7812 1.6367-.5098 1.1065-.383 2.588.3594 3.5293.6723.8488 1.879 1.2321 3.0527.9492 2.1065-.5042 3.0646-2.2822 2.8965-4.2851-.1317-1.58-.8154-2.7536-2.754-4.961-.9607-1.0925-1.6072-2.409-1.5624-3.4062.1373-1.2074.6582-1.832.6582-1.832zM4.8928 15.1936c-.0222.0145-.0439.0614-.0586.1602a.0469.0469 0 0 1-.0059.0195v.01c-.1148.5406-.1649 1.823.1153 2.9687.353 1.4427 1.4175 3.902 4.412 5.129 2.5184 1.0336 5.718.5411 7.8497-1.627.5294-.5435.408-.4897-.4883-.2012v-.002c-1.1121.3558-3.5182.5463-4.7676-1-1.5239-1.8852-.4302-3.3633-1.3574-3.1504-3.6164.8348-5.2667-1.4657-5.5469-2.1016-.0023-.002-.0857-.2487-.1523-.205z'
)

const MarkMinIO = makeSingleColorMark(
  'M13.2072.006c-.6216-.0478-1.2.1943-1.6211.582a2.15 2.15 0 0 0-.0938 3.0352l3.4082 3.5507a3.042 3.042 0 0 1-.664 4.6875l-.463.2383V7.2853a15.4198 15.4198 0 0 0-8.0174 10.4862v.0176l6.5487-3.3281v7.621L13.7794 24V13.6817l.8965-.4629a4.4432 4.4432 0 0 0 1.2207-7.0292l-3.371-3.5254a.7489.7489 0 0 1 .037-1.0547.7522.7522 0 0 1 1.0567.0371l.4668.4863-.006.0059 4.0704 4.2441a.0566.0566 0 0 0 .082 0 .06.06 0 0 0 0-.0703l-3.1406-5.1425-.1484.1425.1484-.1445C14.4945.3926 13.8287.0538 13.2072.006Zm-.9024 9.8652v2.9941l-4.1523 2.1484a13.9787 13.9787 0 0 1 2.7676-3.9277 14.1784 14.1784 0 0 1 1.3847-1.2148z'
)

const MarkWasabi = makeSingleColorMark(
  'M20.483 3.517A11.91 11.91 0 0 0 12 0a11.91 11.91 0 0 0-8.483 3.517A11.91 11.91 0 0 0 0 12a11.91 11.91 0 0 0 3.517 8.483A11.91 11.91 0 0 0 12 24a11.91 11.91 0 0 0 8.483-3.517A11.91 11.91 0 0 0 24 12a11.91 11.91 0 0 0-3.517-8.483Zm1.29 7.387-5.16-4.683-5.285 4.984-2.774 2.615V9.932l4.206-3.994 3.146-2.969c3.163 1.379 5.478 4.365 5.867 7.935zm-.088 2.828a10.632 10.632 0 0 1-1.025 2.951l-2.952-2.668v-3.87Zm-8.183-11.47-2.227 2.103-2.739 2.598v-4.17A9.798 9.798 0 0 1 12 2.155c.513 0 1.007.035 1.502.106zM6.398 13.891l-4.083-3.658a9.744 9.744 0 0 1 1.078-2.987L6.398 9.95zm0-9.968v3.129l-1.75-1.573a8.623 8.623 0 0 1 1.75-1.556Zm-4.189 9.102 5.284 4.736 5.302-4.983 2.74-2.598v3.817l-7.423 7.016a9.823 9.823 0 0 1-5.903-7.988Zm8.306 8.695 5.02-4.754v4.206a9.833 9.833 0 0 1-3.553.654c-.495 0-.99-.035-1.467-.106zm7.176-1.714v-3.11l1.714 1.555a9.604 9.604 0 0 1-1.714 1.555z'
)

// DigitalOcean's upstream SVG uses an explicit #0080FF fill on a <g>;
// swapping it for `currentColor` lets the wrapper tint it the same way
// the other single-color marks are tinted, so dark-mode hover states
// and accent-color overrides work out of the box.
const MarkDigitalOcean: MarkComponent = ({ className }) => (
  <svg
    viewBox="0 -3.954 53.927 53.954"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    className={className}
  >
    <g fill="currentColor" fillRule="evenodd">
      <path d="M24.915 50v-9.661c10.226 0 18.164-10.141 14.237-20.904a14.438 14.438 0 0 0-8.615-8.616C19.774 6.921 9.633 14.83 9.633 25.056H0C0 8.758 15.763-3.954 32.853 1.384 40.311 3.73 46.271 9.661 48.588 17.12 53.927 34.237 41.243 50 24.915 50" />
      <path d="M15.339 40.367h9.604v-9.604H15.34zm-7.401 7.401h7.4v-7.4h-7.4zm-6.187-7.4h6.187V34.18H1.751z" />
    </g>
  </svg>
)

// ─── Multi-color brand marks (fills baked per brand guidelines) ──────

// AWS: smoky "aws" wordmark + orange smile accent. Brand guidelines
// want #232F3E on light surfaces and pure white on dark — we ship both
// and swap with Tailwind's dark variant so the mark always reads
// against whatever surface the tile sits on.
const MarkAWS: MarkComponent = ({ className }) => (
  <>
    <svg
      viewBox="0 0 304 182"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className={cn(className, 'dark:hidden')}
    >
      <path
        fill="#252f3e"
        d="m86 66 2 9c0 3 1 5 3 8v2l-1 3-7 4-2 1-3-1-4-5-3-6c-8 9-18 14-29 14-9 0-16-3-20-8-5-4-8-11-8-19s3-15 9-20c6-6 14-8 25-8a79 79 0 0 1 22 3v-7c0-8-2-13-5-16-3-4-8-5-16-5l-11 1a80 80 0 0 0-14 5h-2c-1 0-2-1-2-3v-5l1-3c0-1 1-2 3-2l12-5 16-2c12 0 20 3 26 8 5 6 8 14 8 25v32zM46 82l10-2c4-1 7-4 10-7l3-6 1-9v-4a84 84 0 0 0-19-2c-6 0-11 1-15 4-3 2-4 6-4 11s1 8 3 11c3 2 6 4 11 4zm80 10-4-1-2-3-23-78-1-4 2-2h10l4 1 2 4 17 66 15-66 2-4 4-1h8l4 1 2 4 16 67 17-67 2-4 4-1h9c2 0 3 1 3 2v2l-1 2-24 78-2 4-4 1h-9l-4-1-1-4-16-65-15 64-2 4-4 1h-9zm129 3a66 66 0 0 1-27-6l-3-3-1-2v-5c0-2 1-3 2-3h2l3 1a54 54 0 0 0 23 5c6 0 11-2 14-4 4-2 5-5 5-9l-2-7-10-5-15-5c-7-2-13-6-16-10a24 24 0 0 1 5-34l10-5a44 44 0 0 1 20-2 110 110 0 0 1 12 3l4 2 3 2 1 4v4c0 3-1 4-2 4l-4-2c-6-2-12-3-19-3-6 0-11 0-14 2s-4 5-4 9c0 3 1 5 3 7s5 4 11 6l14 4c7 3 12 6 15 10s5 9 5 14l-3 12-7 8c-3 3-7 5-11 6l-14 2z"
      />
      <path
        fill="#f90"
        d="M274 144A220 220 0 0 1 4 124c-4-3-1-6 2-4a300 300 0 0 0 263 16c5-2 10 4 5 8z"
      />
      <path
        fill="#f90"
        d="M287 128c-4-5-28-3-38-1-4 0-4-3-1-5 19-13 50-9 53-5 4 5-1 36-18 51-3 2-6 1-5-2 5-10 13-33 9-38z"
      />
    </svg>
    <svg
      viewBox="0 0 304 182"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className={cn(className, 'hidden dark:block')}
    >
      <path
        fill="#ffffff"
        d="m86 66 2 9c0 3 1 5 3 8v2l-1 3-7 4-2 1-3-1-4-5-3-6c-8 9-18 14-29 14-9 0-16-3-20-8-5-4-8-11-8-19s3-15 9-20c6-6 14-8 25-8a79 79 0 0 1 22 3v-7c0-8-2-13-5-16-3-4-8-5-16-5l-11 1a80 80 0 0 0-14 5h-2c-1 0-2-1-2-3v-5l1-3c0-1 1-2 3-2l12-5 16-2c12 0 20 3 26 8 5 6 8 14 8 25v32zM46 82l10-2c4-1 7-4 10-7l3-6 1-9v-4a84 84 0 0 0-19-2c-6 0-11 1-15 4-3 2-4 6-4 11s1 8 3 11c3 2 6 4 11 4zm80 10-4-1-2-3-23-78-1-4 2-2h10l4 1 2 4 17 66 15-66 2-4 4-1h8l4 1 2 4 16 67 17-67 2-4 4-1h9c2 0 3 1 3 2v2l-1 2-24 78-2 4-4 1h-9l-4-1-1-4-16-65-15 64-2 4-4 1h-9zm129 3a66 66 0 0 1-27-6l-3-3-1-2v-5c0-2 1-3 2-3h2l3 1a54 54 0 0 0 23 5c6 0 11-2 14-4 4-2 5-5 5-9l-2-7-10-5-15-5c-7-2-13-6-16-10a24 24 0 0 1 5-34l10-5a44 44 0 0 1 20-2 110 110 0 0 1 12 3l4 2 3 2 1 4v4c0 3-1 4-2 4l-4-2c-6-2-12-3-19-3-6 0-11 0-14 2s-4 5-4 9c0 3 1 5 3 7s5 4 11 6l14 4c7 3 12 6 15 10s5 9 5 14l-3 12-7 8c-3 3-7 5-11 6l-14 2z"
      />
      <path
        fill="#f90"
        d="M274 144A220 220 0 0 1 4 124c-4-3-1-6 2-4a300 300 0 0 0 263 16c5-2 10 4 5 8z"
      />
      <path
        fill="#f90"
        d="M287 128c-4-5-28-3-38-1-4 0-4-3-1-5 19-13 50-9 53-5 4 5-1 36-18 51-3 2-6 1-5-2 5-10 13-33 9-38z"
      />
    </svg>
  </>
)

// Cloudflare cloud mark, three-path two-tone orange + white.
const MarkCloudflare: MarkComponent = ({ className }) => (
  <svg
    viewBox="0 0 256 116"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    className={className}
  >
    <path
      fill="#FFF"
      d="m202.357 49.394-5.311-2.124C172.085 103.434 72.786 69.289 66.81 85.997c-.996 11.286 54.227 2.146 93.706 4.059 12.039.583 18.076 9.671 12.964 24.484l10.069.031c11.615-36.209 48.683-17.73 50.232-29.68-2.545-7.857-42.601 0-31.425-35.497Z"
    />
    <path
      fill="#F4811F"
      d="M176.332 108.348c1.593-5.31 1.062-10.622-1.593-13.809-2.656-3.187-6.374-5.31-11.154-5.842L71.17 87.634c-.531 0-1.062-.53-1.593-.53-.531-.532-.531-1.063 0-1.594.531-1.062 1.062-1.594 2.124-1.594l92.946-1.062c11.154-.53 22.839-9.56 27.087-20.182l5.312-13.809c0-.532.531-1.063 0-1.594C191.203 20.182 166.772 0 138.091 0 111.535 0 88.697 16.995 80.73 40.896c-5.311-3.718-11.684-5.843-19.12-5.31-12.747 1.061-22.838 11.683-24.432 24.43-.531 3.187 0 6.374.532 9.56C16.996 70.107 0 87.103 0 108.348c0 2.124 0 3.718.531 5.842 0 1.063 1.062 1.594 1.594 1.594h170.489c1.062 0 2.125-.53 2.125-1.594l1.593-5.842Z"
    />
    <path
      fill="#FAAD3F"
      d="M205.544 48.863h-2.656c-.531 0-1.062.53-1.593 1.062l-3.718 12.747c-1.593 5.31-1.062 10.623 1.594 13.809 2.655 3.187 6.373 5.31 11.153 5.843l19.652 1.062c.53 0 1.062.53 1.593.53.53.532.53 1.063 0 1.594-.531 1.063-1.062 1.594-2.125 1.594l-20.182 1.062c-11.154.53-22.838 9.56-27.087 20.182l-1.063 4.78c-.531.532 0 1.594 1.063 1.594h70.108c1.062 0 1.593-.531 1.593-1.593 1.062-4.25 2.124-9.03 2.124-13.81 0-27.618-22.838-50.456-50.456-50.456"
    />
  </svg>
)

// Google Cloud four-color cloud mark.
const MarkGoogleCloud: MarkComponent = ({ className }) => (
  <svg
    viewBox="0 -25 256 256"
    xmlns="http://www.w3.org/2000/svg"
    aria-hidden="true"
    className={className}
  >
    <path
      fill="#EA4335"
      d="m170.252 56.819 22.253-22.253 1.483-9.37C153.437-11.677 88.976-7.496 52.42 33.92 42.267 45.423 34.734 59.764 30.717 74.573l7.97-1.123 44.505-7.34 3.436-3.513c19.797-21.742 53.27-24.667 76.128-6.168l7.496.39Z"
    />
    <path
      fill="#4285F4"
      d="M224.205 73.918a100.249 100.249 0 0 0-30.217-48.722l-31.232 31.232a55.515 55.515 0 0 1 20.379 44.037v5.544c15.35 0 27.797 12.445 27.797 27.796 0 15.352-12.446 27.485-27.797 27.485h-55.671l-5.466 5.934v33.34l5.466 5.231h55.67c39.93.311 72.553-31.494 72.864-71.424a72.303 72.303 0 0 0-31.793-60.453"
    />
    <path
      fill="#34A853"
      d="M71.87 205.796h55.593V161.29H71.87a27.275 27.275 0 0 1-11.399-2.498l-7.887 2.42-22.409 22.253-1.952 7.574c12.567 9.489 27.9 14.825 43.647 14.757"
    />
    <path
      fill="#FBBC05"
      d="M71.87 61.425C31.94 61.664-.237 94.228.001 134.159a72.301 72.301 0 0 0 28.222 56.88l32.248-32.246c-13.99-6.322-20.208-22.786-13.887-36.776 6.32-13.99 22.786-20.208 36.775-13.888a27.796 27.796 0 0 1 13.887 13.888l32.248-32.248A72.224 72.224 0 0 0 71.87 61.425"
    />
  </svg>
)

// ─── Generic-slot marks (lucide) ─────────────────────────────────────
// These vendors never had a real brand glyph — `hugeicons:cloud-server`
// and `lucide:hard-drive` were the pre-existing fallbacks. Using the
// matching lucide-react component here avoids shipping a second icon
// library for literally one icon.

const MarkCloud: MarkComponent = ({ className }) => (
  <Cloud className={className} strokeWidth={1.7} />
)
const MarkServer: MarkComponent = ({ className }) => (
  <Server className={className} strokeWidth={1.7} />
)
const MarkHardDrive: MarkComponent = ({ className }) => (
  <HardDrive className={className} strokeWidth={1.7} />
)

// ─── Vendor registry ─────────────────────────────────────────────────

const STORAGE_VENDOR_META: Record<LogoVendor, VendorMeta> = {
  s3: { label: 'S3 Compatible', color: '#64748B', Mark: MarkCloud },
  aws: { label: 'Amazon Web Services', color: '#FF9900', Mark: MarkAWS },
  cloudflare: { label: 'Cloudflare', color: '#F38020', Mark: MarkCloudflare },
  aliyun: { label: 'Alibaba Cloud', color: '#FF6A00', Mark: MarkAlibabaCloud },
  tencent: { label: 'Tencent Cloud', color: '#00A4FF', Mark: MarkCloud },
  huawei: { label: 'Huawei Cloud', color: '#CF0A2C', Mark: MarkHuawei },
  baidu: { label: 'Baidu Cloud', color: '#2932E1', Mark: MarkBaidu },
  gcp: { label: 'Google Cloud', color: '#4285F4', Mark: MarkGoogleCloud },
  backblaze: { label: 'Backblaze', color: '#E93D25', Mark: MarkBackblaze },
  minio: { label: 'MinIO', color: '#C72E49', Mark: MarkMinIO },
  wasabi: { label: 'Wasabi', color: '#01CD3E', Mark: MarkWasabi },
  do: { label: 'DigitalOcean', color: '#0080FF', Mark: MarkDigitalOcean },
  ftp: { label: 'FTP', color: '#64748B', Mark: MarkServer },
  local: { label: 'Local', color: '#64748B', Mark: MarkHardDrive },
}

export const STORAGE_VENDOR_LABELS: Record<LogoVendor, string> =
  Object.fromEntries(
    Object.entries(STORAGE_VENDOR_META).map(([vendor, meta]) => [
      vendor,
      meta.label,
    ])
  ) as Record<LogoVendor, string>

function normalizeDriver(driver: string): string {
  return (driver ?? '').toLowerCase()
}

export function resolveLogoVendor(
  provider: string | undefined,
  driver: string
): LogoVendor {
  const p = (provider ?? '').toLowerCase()

  if (p.includes('custom-s3')) return 's3'
  if (p.includes('aws') || p.includes('amazon')) return 'aws'
  if (p.includes('cloudflare') || p.includes('r2')) return 'cloudflare'
  if (p.includes('aliyun') || p.includes('alibaba') || p.includes('aliyuncs'))
    return 'aliyun'
  if (p.includes('tencent') || p.includes('cos') || p.includes('qcloud'))
    return 'tencent'
  if (p.includes('huawei') || p.includes('obs') || p.includes('myhuaweicloud'))
    return 'huawei'
  if (p.includes('baidu') || p.includes('bos') || p.includes('bcebos'))
    return 'baidu'
  if (p.includes('gcp') || p.includes('google') || p.includes('gcs'))
    return 'gcp'
  if (p.includes('backblaze') || p.includes('b2')) return 'backblaze'
  if (p.includes('minio')) return 'minio'
  if (p.includes('wasabi')) return 'wasabi'
  if (p.includes('digitalocean') || p.includes('spaces')) return 'do'

  switch (normalizeDriver(driver)) {
    case 'ftp':
      return 'ftp'
    case 'local':
      return 'local'
    case 's3':
    case 'oss':
    case 'cos':
    case 'obs':
    case 'bos':
      return 's3'
    default:
      return 'local'
  }
}

export function getStorageBrandMeta(
  provider: string | undefined,
  driver: string
) {
  const vendor = resolveLogoVendor(provider, driver)
  return {
    vendor,
    ...STORAGE_VENDOR_META[vendor],
  }
}

interface StorageLogoProps {
  vendor: LogoVendor
  size?: number
  rounded?: string
  className?: string
}

export function StorageLogo({
  vendor,
  size = 40,
  rounded = 'rounded-lg',
  className,
}: StorageLogoProps) {
  const meta = STORAGE_VENDOR_META[vendor] ?? STORAGE_VENDOR_META.local
  const { Mark } = meta

  return (
    <div
      className={cn(
        'relative flex shrink-0 items-center justify-center overflow-hidden border border-border/70 bg-linear-to-br from-background to-muted/35',
        rounded,
        className
      )}
      style={{ width: size, height: size }}
      title={meta.label}
    >
      {/* The inner wrapper paints the vendor color for single-color
          marks via `currentColor` inheritance; multi-color marks
          ignore it and show their baked-in brand fills. */}
      <div
        className="flex size-[72%] items-center justify-center"
        style={{ color: meta.color }}
      >
        <Mark className="size-full" />
      </div>
    </div>
  )
}

export type { LogoVendor }
