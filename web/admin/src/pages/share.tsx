import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  AlertCircle,
  ArrowLeft,
  Copy,
  Download,
  ExternalLink,
  FileIcon,
  FileText,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'

import { shareApi } from '@/lib/api'
import { useI18n } from '@/i18n'
import { formatSize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { KiteLogo } from '@/components/kite-logo'

interface ShareInfo {
  hash: string
  original_name: string
  file_type: 'image' | 'video' | 'audio' | 'file'
  mime_type: string
  size_bytes: number
  width?: number | null
  height?: number | null
  duration?: number | null
  created_at: string
  raw_url: string
  download_url: string
  thumb_url?: string | null
}

// MIME prefixes that browsers can render reliably as plain text inside a
// <pre> block. Anything outside this list (or larger than the cap below)
// falls through to the download-only fallback.
const TEXT_MIME_PREFIXES = [
  'text/',
  'application/json',
  'application/xml',
  'application/javascript',
  'application/x-yaml',
  'application/x-sh',
]
const TEXT_PREVIEW_BYTES_CAP = 256 * 1024

function isTextLike(mime: string, name: string): boolean {
  const m = mime.toLowerCase()
  if (TEXT_MIME_PREFIXES.some((p) => m.startsWith(p))) return true
  // Fallback to extension heuristics for files mis-detected as octet-stream
  // (uploaded README.md, .log, .csv etc.).
  const ext = name.split('.').pop()?.toLowerCase() ?? ''
  return [
    'md',
    'markdown',
    'txt',
    'log',
    'csv',
    'tsv',
    'json',
    'yaml',
    'yml',
    'xml',
    'html',
    'htm',
    'css',
    'js',
    'ts',
    'tsx',
    'jsx',
    'go',
    'py',
    'rs',
    'java',
    'c',
    'h',
    'cpp',
    'rb',
    'sh',
    'sql',
    'toml',
    'ini',
    'conf',
  ].includes(ext)
}

function isPdf(mime: string, name: string): boolean {
  return (
    mime.toLowerCase() === 'application/pdf' ||
    name.toLowerCase().endsWith('.pdf')
  )
}

export default function SharePage() {
  const { t } = useI18n()
  const { hash = '' } = useParams<{ hash: string }>()

  const { data, isLoading, error } = useQuery<ShareInfo>({
    queryKey: ['share', hash],
    queryFn: () => shareApi.info(hash).then((r) => r.data.data),
    enabled: !!hash,
    retry: 0,
    staleTime: 60_000,
  })

  // Tab title reflects the shared file once metadata has loaded.
  useEffect(() => {
    if (data?.original_name) {
      document.title = `${data.original_name} · Kite`
    }
  }, [data?.original_name])

  if (isLoading) {
    return (
      <ShareShell>
        <div className="flex min-h-[40vh] items-center justify-center text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
        </div>
      </ShareShell>
    )
  }

  if (error || !data) {
    const status = (error as { response?: { status?: number } } | undefined)
      ?.response?.status
    return (
      <ShareShell>
        <div className="mx-auto flex max-w-md flex-col items-center gap-4 py-12 text-center">
          <div className="inline-flex size-12 items-center justify-center rounded-full border bg-muted/50 text-foreground">
            <AlertCircle className="size-5" />
          </div>
          <div className="space-y-1">
            <h2 className="text-xl font-semibold tracking-tight">
              {status === 404
                ? t('share.notFoundTitle')
                : t('share.errorTitle')}
            </h2>
            <p className="text-sm text-muted-foreground">
              {status === 404 ? t('share.notFoundDesc') : t('share.errorDesc')}
            </p>
          </div>
          <Button asChild variant="outline" size="sm">
            <Link to="/">
              <ArrowLeft className="size-4" />
              {t('share.backHome')}
            </Link>
          </Button>
        </div>
      </ShareShell>
    )
  }

  return (
    <ShareShell>
      <FileMeta info={data} />
      <FilePreview info={data} />
    </ShareShell>
  )
}

function ShareShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-svh bg-background">
      <header className="border-b">
        <div className="mx-auto flex h-14 max-w-5xl items-center gap-3 px-4 sm:px-6">
          <Link to="/" className="flex items-center gap-2">
            <KiteLogo className="size-5" />
            <span className="text-sm font-medium">Kite</span>
          </Link>
        </div>
      </header>
      <main className="mx-auto w-full max-w-5xl px-4 py-6 sm:px-6 sm:py-10">
        {children}
      </main>
    </div>
  )
}

function FileMeta({ info }: { info: ShareInfo }) {
  const { t } = useI18n()

  const copyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href)
      toast.success(t('share.linkCopied'))
    } catch {
      toast.error(t('share.linkCopyFailed'))
    }
  }

  const meta = useMemo(() => {
    const parts: string[] = []
    parts.push(formatSize(info.size_bytes))
    if (info.width && info.height) parts.push(`${info.width}×${info.height}`)
    if (info.duration) parts.push(formatDuration(info.duration))
    parts.push(new Date(info.created_at).toLocaleString())
    return parts.join(' · ')
  }, [info])

  return (
    <div className="mb-5 flex flex-col gap-3 sm:mb-6 sm:flex-row sm:items-start sm:justify-between">
      <div className="flex min-w-0 items-start gap-3">
        <div className="inline-flex size-10 shrink-0 items-center justify-center rounded-md border bg-muted/50 text-foreground sm:size-11">
          <FileIcon className="size-4" />
        </div>
        <div className="min-w-0 space-y-0.5">
          <h1 className="truncate text-lg font-semibold tracking-tight sm:text-xl">
            {info.original_name}
          </h1>
          <p className="text-xs text-muted-foreground sm:text-sm">{meta}</p>
        </div>
      </div>
      <div className="flex shrink-0 flex-wrap items-center gap-2">
        <Button variant="outline" size="sm" onClick={copyLink}>
          <Copy className="size-3.5" />
          {t('share.copyLink')}
        </Button>
        <Button asChild size="sm">
          <a href={info.download_url}>
            <Download className="size-3.5" />
            {t('share.download')}
          </a>
        </Button>
      </div>
    </div>
  )
}

function FilePreview({ info }: { info: ShareInfo }) {
  if (info.file_type === 'image') return <ImagePreview info={info} />
  if (info.file_type === 'video') return <VideoPreview info={info} />
  if (info.file_type === 'audio') return <AudioPreview info={info} />
  if (isPdf(info.mime_type, info.original_name))
    return <PdfPreview info={info} />
  if (isTextLike(info.mime_type, info.original_name))
    return <TextPreview info={info} />
  return <UnsupportedPreview info={info} />
}

function ImagePreview({ info }: { info: ShareInfo }) {
  return (
    <div className="overflow-hidden rounded-xl border bg-muted/30">
      <img
        src={info.raw_url}
        alt={info.original_name}
        className="mx-auto block max-h-[78svh] w-auto max-w-full object-contain"
      />
    </div>
  )
}

function VideoPreview({ info }: { info: ShareInfo }) {
  return (
    <div className="overflow-hidden rounded-xl border bg-black">
      <video
        src={info.raw_url}
        controls
        playsInline
        preload="metadata"
        className="mx-auto block max-h-[78svh] w-full"
      />
    </div>
  )
}

function AudioPreview({ info }: { info: ShareInfo }) {
  return (
    <div className="rounded-xl border bg-muted/30 p-6">
      <audio
        src={info.raw_url}
        controls
        preload="metadata"
        className="w-full"
      />
    </div>
  )
}

function PdfPreview({ info }: { info: ShareInfo }) {
  return (
    <div className="overflow-hidden rounded-xl border bg-muted/30">
      <iframe
        src={info.raw_url}
        title={info.original_name}
        className="block h-[78svh] w-full"
      />
    </div>
  )
}

function TextPreview({ info }: { info: ShareInfo }) {
  const { t } = useI18n()
  const [content, setContent] = useState<string | null>(null)
  const [truncated, setTruncated] = useState(false)
  const [failed, setFailed] = useState(false)

  useEffect(() => {
    let cancelled = false
    const ctrl = new AbortController()

    fetch(info.raw_url, { signal: ctrl.signal })
      .then(async (r) => {
        if (!r.ok) throw new Error(`status ${r.status}`)
        // Cap how much we read so a 200MB log doesn't lock the tab. We
        // pull the bytes via the streaming reader so we can stop early
        // without holding the whole body in memory first.
        const reader = r.body?.getReader()
        if (!reader) {
          const text = await r.text()
          if (cancelled) return
          if (text.length > TEXT_PREVIEW_BYTES_CAP) {
            setContent(text.slice(0, TEXT_PREVIEW_BYTES_CAP))
            setTruncated(true)
          } else {
            setContent(text)
          }
          return
        }
        const decoder = new TextDecoder()
        let acc = ''
        while (true) {
          const { value, done } = await reader.read()
          if (done) break
          acc += decoder.decode(value, { stream: true })
          if (acc.length >= TEXT_PREVIEW_BYTES_CAP) {
            setTruncated(true)
            ctrl.abort()
            break
          }
        }
        if (!cancelled) setContent(acc.slice(0, TEXT_PREVIEW_BYTES_CAP))
      })
      .catch((e) => {
        if (e.name === 'AbortError') return
        if (!cancelled) setFailed(true)
      })

    return () => {
      cancelled = true
      ctrl.abort()
    }
  }, [info.raw_url])

  if (failed) {
    return <UnsupportedPreview info={info} />
  }
  if (content === null) {
    return (
      <div className="flex min-h-[40vh] items-center justify-center rounded-xl border text-muted-foreground">
        <Loader2 className="size-5 animate-spin" />
      </div>
    )
  }
  return (
    <div className="rounded-xl border bg-muted/30">
      <div className="flex items-center justify-between border-b px-4 py-2 text-xs text-muted-foreground">
        <span className="inline-flex items-center gap-1.5">
          <FileText className="size-3.5" />
          {info.mime_type || t('share.textPreview')}
        </span>
        {truncated && (
          <span>
            {t('share.textTruncated').replace(
              '{kb}',
              String(TEXT_PREVIEW_BYTES_CAP / 1024)
            )}
          </span>
        )}
      </div>
      <pre className="max-h-[78svh] overflow-auto p-4 text-xs leading-relaxed">
        {content}
      </pre>
    </div>
  )
}

function UnsupportedPreview({ info }: { info: ShareInfo }) {
  const { t } = useI18n()
  return (
    <div className="flex flex-col items-center gap-4 rounded-xl border bg-muted/30 px-6 py-16 text-center">
      <div className="inline-flex size-14 items-center justify-center rounded-full border bg-background text-muted-foreground">
        <FileIcon className="size-6" />
      </div>
      <div className="space-y-1">
        <h2 className="text-lg font-semibold tracking-tight">
          {t('share.unsupportedTitle')}
        </h2>
        <p className="max-w-md text-sm text-muted-foreground">
          {t('share.unsupportedDesc')}
        </p>
      </div>
      <div className="flex flex-wrap items-center justify-center gap-2">
        <Button asChild>
          <a href={info.download_url}>
            <Download className="size-4" />
            {t('share.download')}
          </a>
        </Button>
        <Button asChild variant="outline">
          <a href={info.raw_url} target="_blank" rel="noopener noreferrer">
            <ExternalLink className="size-4" />
            {t('share.openRaw')}
          </a>
        </Button>
      </div>
    </div>
  )
}

function formatDuration(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h > 0) {
    return `${h}:${String(m).padStart(2, '0')}:${String(sec).padStart(2, '0')}`
  }
  return `${m}:${String(sec).padStart(2, '0')}`
}
