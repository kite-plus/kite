import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ExternalLink, Sparkles, X } from 'lucide-react'

import { updateCheckApi } from '@/lib/api'
import { useI18n } from '@/i18n'

interface UpdateInfo {
  current: string
  latest: string
  has_update: boolean
  published_at?: string
  html_url?: string
  error?: string
}

// Per-tag dismissal — once an admin clicks X for v1.2.3, the banner stays
// hidden until v1.2.4 ships. Stored in localStorage so it survives reloads
// without needing a server-side preference table.
const DISMISS_KEY = 'kite.update-dismissed-tag'

export function UpdateBanner() {
  const { t } = useI18n()
  const [dismissedTag, setDismissedTag] = useState<string>(() => {
    if (typeof window === 'undefined') return ''
    return localStorage.getItem(DISMISS_KEY) ?? ''
  })

  const { data } = useQuery<UpdateInfo>({
    queryKey: ['admin', 'update-check'],
    queryFn: () => updateCheckApi.check().then((r) => r.data.data),
    // The backend caches the upstream call for ~6h, so a stricter
    // client-side staleTime would just hide cached data without
    // helping anyone — half an hour is plenty.
    staleTime: 1000 * 60 * 30,
    refetchOnWindowFocus: false,
    retry: 0,
  })

  if (!data?.has_update) return null
  if (dismissedTag && dismissedTag === data.latest) return null

  const handleDismiss = () => {
    if (typeof window !== 'undefined') {
      localStorage.setItem(DISMISS_KEY, data.latest)
    }
    setDismissedTag(data.latest)
  }

  return (
    <div className="flex items-center gap-3 rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-4 py-3 text-sm">
      <Sparkles className="size-4 shrink-0 text-emerald-700 dark:text-emerald-400" />
      <div className="min-w-0 flex-1">
        <span className="font-medium">
          {t('updateBanner.available').replace('{version}', data.latest)}
        </span>
        <span className="ms-2 text-muted-foreground">
          {t('updateBanner.current').replace('{version}', data.current || '—')}
        </span>
      </div>
      {data.html_url && (
        <a
          href={data.html_url}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex shrink-0 items-center gap-1 text-xs font-medium text-emerald-700 underline-offset-4 hover:underline dark:text-emerald-400"
        >
          {t('updateBanner.viewRelease')}
          <ExternalLink className="size-3" />
        </a>
      )}
      <button
        type="button"
        onClick={handleDismiss}
        className="-me-1 inline-flex size-6 shrink-0 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-foreground/5 hover:text-foreground"
        aria-label={t('updateBanner.dismiss')}
      >
        <X className="size-3.5" />
      </button>
    </div>
  )
}
