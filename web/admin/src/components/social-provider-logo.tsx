import { cn } from '@/lib/utils'
import { useI18n } from '@/i18n'

// Provider brand marks are inlined as SVG components, not fetched from an
// external CDN. The previous implementation pulled github / google logos
// from svgl.app and wechat from iconify's API, which meant:
//   1. every mount of the third-party-accounts tab sent 2–3 cross-origin
//      HTTP requests, so on unreliable networks (or in air-gapped
//      installs) the logos visibly popped in a beat after the card,
//   2. `loading="lazy"` compounded the delay by deferring fetch until
//      scroll intersection — wrong for logos that are always visible,
//   3. the component shipped a failed-state fallback precisely because
//      those CDN requests could fail.
// Since these are static brand marks that ship with the product, not
// user content, bundling them into the JS chunk eliminates every one of
// those failure modes at the cost of ~1 KB gzipped.

type SocialProviderKey = 'wechat' | 'github' | 'google'

// ─── Brand marks ─────────────────────────────────────────────────────
// Each logo is kept to a minimal viewBox-normalised SVG so it scales
// cleanly at any size and stays crisp on Retina. Colors:
//   - GitHub: monochrome, uses `currentColor` so the parent's text
//     color paints the glyph — that way it automatically inverts in
//     dark mode via the ambient `text-foreground` token.
//   - Google: the official four-color G, colors baked in since the
//     brand guidelines disallow single-color recolors.
//   - WeChat: brand green #07C160 baked in for the same reason.

function GithubMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      fill="currentColor"
      aria-hidden="true"
      className={className}
    >
      <path d="M12 .297c-6.63 0-12 5.373-12 12 0 5.303 3.438 9.8 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61C4.422 18.07 3.633 17.7 3.633 17.7c-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.605-2.665-.3-5.466-1.332-5.466-5.93 0-1.31.465-2.38 1.235-3.22-.135-.303-.54-1.523.105-3.176 0 0 1.005-.322 3.3 1.23.96-.267 1.98-.399 3-.405 1.02.006 2.04.138 3 .405 2.28-1.552 3.285-1.23 3.285-1.23.645 1.653.24 2.873.12 3.176.765.84 1.23 1.91 1.23 3.22 0 4.61-2.805 5.625-5.475 5.92.42.36.81 1.096.81 2.22 0 1.606-.015 2.896-.015 3.286 0 .315.21.69.825.57C20.565 22.092 24 17.592 24 12.297c0-6.627-5.373-12-12-12" />
    </svg>
  )
}

function GoogleMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 48 48"
      xmlns="http://www.w3.org/2000/svg"
      aria-hidden="true"
      className={className}
    >
      <path
        fill="#FFC107"
        d="M43.611 20.083H42V20H24v8h11.303c-1.649 4.657-6.08 8-11.303 8-6.627 0-12-5.373-12-12s5.373-12 12-12c3.059 0 5.842 1.154 7.961 3.039l5.657-5.657C34.046 6.053 29.268 4 24 4 12.955 4 4 12.955 4 24s8.955 20 20 20 20-8.955 20-20c0-1.341-.138-2.65-.389-3.917z"
      />
      <path
        fill="#FF3D00"
        d="M6.306 14.691l6.571 4.819C14.655 15.108 18.961 12 24 12c3.059 0 5.842 1.154 7.961 3.039l5.657-5.657C34.046 6.053 29.268 4 24 4 16.318 4 9.656 8.337 6.306 14.691z"
      />
      <path
        fill="#4CAF50"
        d="M24 44c5.166 0 9.86-1.977 13.409-5.192l-6.19-5.238A11.91 11.91 0 0 1 24 36c-5.202 0-9.619-3.317-11.283-7.946l-6.522 5.025C9.505 39.556 16.227 44 24 44z"
      />
      <path
        fill="#1976D2"
        d="M43.611 20.083H42V20H24v8h11.303a12.04 12.04 0 0 1-4.087 5.571l.003-.002 6.19 5.238C36.971 39.205 44 34 44 24c0-1.341-.138-2.65-.389-3.917z"
      />
    </svg>
  )
}

function WechatMark({ className }: { className?: string }) {
  // Single <path> with 6 subpaths (2 bubbles + 4 eyes), copied verbatim
  // from simple-icons' `wechat.svg`. A previous hand-transcribed version
  // split the mark into two <path> elements and used slightly wrong
  // coordinates/radii for the eyes — the left bubble rendered as a solid
  // green blob because its eye subpaths no longer wound opposite to the
  // outer shape, so the default `fill-rule="nonzero"` filled them in
  // instead of cutting holes. Using the upstream path keeps the winding
  // that the artwork was drawn for.
  return (
    <svg
      viewBox="0 0 24 24"
      xmlns="http://www.w3.org/2000/svg"
      fill="#07C160"
      aria-hidden="true"
      className={className}
    >
      <path d="M8.691 2.188C3.891 2.188 0 5.476 0 9.53c0 2.212 1.17 4.203 3.002 5.55a.59.59 0 0 1 .213.665l-.39 1.48c-.019.07-.048.141-.048.213 0 .163.13.295.29.295a.326.326 0 0 0 .167-.054l1.903-1.114a.864.864 0 0 1 .717-.098 10.16 10.16 0 0 0 2.837.403c.276 0 .543-.027.811-.05-.857-2.578.157-4.972 1.932-6.446 1.703-1.415 3.882-1.98 5.853-1.838-.576-3.583-4.196-6.348-8.596-6.348zM5.785 5.991c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178A1.17 1.17 0 0 1 4.623 7.17c0-.651.52-1.18 1.162-1.18zm5.813 0c.642 0 1.162.529 1.162 1.18a1.17 1.17 0 0 1-1.162 1.178 1.17 1.17 0 0 1-1.162-1.178c0-.651.52-1.18 1.162-1.18zm5.34 2.867c-1.797-.052-3.746.512-5.28 1.786-1.72 1.428-2.687 3.72-1.78 6.22.942 2.453 3.666 4.229 6.884 4.229.826 0 1.622-.12 2.361-.336a.722.722 0 0 1 .598.082l1.584.926a.272.272 0 0 0 .14.047c.134 0 .24-.111.24-.247 0-.06-.023-.12-.038-.177l-.327-1.233a.582.582 0 0 1-.023-.156.49.49 0 0 1 .201-.398C23.024 18.48 24 16.82 24 14.98c0-3.21-2.931-5.837-6.656-6.088V8.89c-.135-.01-.27-.027-.407-.03zm-2.53 3.274c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.97-.982zm4.844 0c.535 0 .969.44.969.982a.976.976 0 0 1-.969.983.976.976 0 0 1-.969-.983c0-.542.434-.982.969-.982z" />
    </svg>
  )
}

// ─── Registry ────────────────────────────────────────────────────────

// `label` is null for providers whose brand name doesn't translate
// (GitHub, Google) — those keep the hard-coded mark below. WeChat
// resolves through the i18n catalogue because the brand reads
// "WeChat" in English contexts and "微信" in Chinese contexts.
const PROVIDER_META: Record<
  SocialProviderKey,
  {
    label: string | null
    labelKey: string | null
    Mark: (props: { className?: string }) => React.ReactElement
  }
> = {
  wechat: { label: null, labelKey: 'auth.providerWechat', Mark: WechatMark },
  github: { label: 'GitHub', labelKey: null, Mark: GithubMark },
  google: { label: 'Google', labelKey: null, Mark: GoogleMark },
}

// ─── Public component ────────────────────────────────────────────────

interface SocialProviderLogoProps {
  provider: string
  size?: number
  className?: string
  rounded?: string
  appearance?: 'badge' | 'plain'
}

export function SocialProviderLogo({
  provider,
  size = 20,
  className,
  rounded = 'rounded-md',
  appearance = 'badge',
}: SocialProviderLogoProps) {
  const { t } = useI18n()
  // Unknown providers fall through to GitHub — every third-party
  // integration ever shipped here renders a real mark, so the fallback
  // is only relevant for corrupted config and shouldn't draw attention.
  const key =
    (provider as SocialProviderKey) in PROVIDER_META
      ? (provider as SocialProviderKey)
      : 'github'
  const meta = PROVIDER_META[key]
  const label = meta.labelKey ? t(meta.labelKey) : (meta.label ?? '')
  const Mark = meta.Mark

  // Shared inner scaling: the mark fills the container in `plain` mode
  // (logo sits directly on the surrounding surface) but gets a small
  // inset in `badge` mode so the round/square chrome has breathing room.
  const markClassName = cn(appearance === 'badge' ? 'size-[70%]' : 'size-full')

  if (appearance === 'plain') {
    return (
      <div
        className={cn(
          'inline-flex shrink-0 items-center justify-center overflow-hidden text-foreground',
          className
        )}
        style={{ width: size, height: size }}
        aria-label={label}
        title={label}
      >
        <Mark className={markClassName} />
      </div>
    )
  }

  return (
    <div
      className={cn(
        'inline-flex shrink-0 items-center justify-center overflow-hidden border bg-background/80 text-foreground',
        rounded,
        className
      )}
      style={{ width: size, height: size }}
      aria-label={label}
      title={label}
    >
      <Mark className={markClassName} />
    </div>
  )
}
