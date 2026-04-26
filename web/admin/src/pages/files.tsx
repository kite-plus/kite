import { useState, useCallback, useEffect, useMemo, useRef } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Upload,
  Trash2,
  Copy,
  LayoutGrid,
  List,
  Search,
  Check,
  ExternalLink,
  ChevronLeft,
  ChevronRight,
  Folder,
  Layers,
  Image as ImageIcon,
  Video,
  Music,
  FileText,
  File as FileIcon,
  Eye,
  Link2,
  MoreHorizontal,
  Download,
  Share2,
  X,
  Zap,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'

import { albumApi, authApi, fileApi, statsApi } from '@/lib/api'
import { useI18n } from '@/i18n'
import { cn, formatRelativeTime, formatSize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Skeleton } from '@/components/ui/skeleton'
import { Progress } from '@/components/ui/progress'
import {
  getFileIconInfo,
  getFileTypeLabel,
  isImagePreviewable,
} from '@/lib/file-utils'
import { useAdaptiveGridPageSize } from '@/hooks/use-adaptive-grid-page-size'
import { PageHeader } from '@/components/page-header'
import { EmptyKite } from '@/components/empty-state'

const LIST_PAGE_SIZE = 20
const DEFAULT_GRID_PAGE_SIZE = 20
const DEFAULT_UPLOAD_MAX_FILE_SIZE_BYTES = 100 * 1024 * 1024

interface PublicAuthOptions {
  allow_registration: boolean
  upload_max_file_size_mb?: string
  upload_max_file_size_bytes?: number
}

function resolveUploadMaxFileSizeBytes(options?: PublicAuthOptions): number {
  if (
    typeof options?.upload_max_file_size_bytes === 'number' &&
    options.upload_max_file_size_bytes > 0
  ) {
    return options.upload_max_file_size_bytes
  }
  const parsedMB = Number.parseInt(options?.upload_max_file_size_mb ?? '', 10)
  if (Number.isFinite(parsedMB) && parsedMB > 0) {
    return parsedMB * 1024 * 1024
  }
  return DEFAULT_UPLOAD_MAX_FILE_SIZE_BYTES
}

function readUploadErrorMessage(
  xhr: XMLHttpRequest,
  tooManyRequestsMessage?: string
): string | null {
  try {
    const payload = JSON.parse(xhr.responseText) as {
      code?: number
      message?: string
      data?: { message?: string }
    }
    if (payload.code === 42900) {
      return (
        tooManyRequestsMessage ??
        payload.message ??
        payload.data?.message ??
        null
      )
    }
    return payload.message ?? payload.data?.message ?? null
  } catch {
    return null
  }
}

/* ────────────────────────────────────────────────────────────
 * Types
 * ──────────────────────────────────────────────────────────── */
interface FileItem {
  id: string
  original_name: string
  file_type: string
  mime_type: string
  size_bytes: number
  hash_md5: string
  url: string
  source_url?: string
  thumb_url?: string
  created_at: string
  width?: number
  height?: number
}

interface UploadTask {
  id: string
  file: File
  progress: number
  status: 'uploading' | 'processing' | 'done' | 'error'
}

interface UserStats {
  total_files: number
  total_size: number
  images: number
  videos: number
  audios: number
  others: number
}

interface PendingDelete {
  ids: string[]
  displayName?: string
  closeDetail?: boolean
}

interface FolderItem {
  id: string
  name: string
  parent_id?: string
}

interface FolderOption {
  id: string
  label: string
}

/* ────────────────────────────────────────────────────────────
 * Helpers
 * ──────────────────────────────────────────────────────────── */
function formatBytes(bytes: number) {
  if (!bytes || bytes <= 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

/** Splice React nodes into a translated template string at `{key}` markers.
 *  Used for phrases like "已选择 {n} 个项" where the count should render as
 *  a styled element (e.g. <b>). Unknown markers are left as-is. */
function renderWithBold(
  template: string,
  replacements: Record<string, React.ReactNode>
): React.ReactNode[] {
  const keys = Object.keys(replacements)
  if (keys.length === 0) return [template]
  const re = new RegExp(
    `(${keys.map((k) => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})`,
    'g'
  )
  return template
    .split(re)
    .map((chunk, i) =>
      keys.includes(chunk) ? (
        <span key={i}>{replacements[chunk]}</span>
      ) : (
        <span key={i}>{chunk}</span>
      )
    )
}

/** Stable hue hash for per-tile color tinting. Pulls a deterministic
 *  number in [0, 360) out of the file id so image/audio visuals keep a
 *  consistent look across renders without needing any backend field. */
function hashHue(seed: string): number {
  let h = 0
  for (let i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.charCodeAt(i)) | 0
  }
  return Math.abs(h) % 360
}

/** Document extension allow-list. Anything inside this set, when the backend
 *  classifies the file as `file`, is surfaced under the `文档` pill; everything
 *  else under that bucket falls through to `其它`. The backend has no notion
 *  of a document category, so this split is a frontend-only convenience. */
const DOC_EXTS = new Set([
  'pdf',
  'doc',
  'docx',
  'odt',
  'rtf',
  'pages',
  'xls',
  'xlsx',
  'ods',
  'csv',
  'numbers',
  'ppt',
  'pptx',
  'odp',
  'keynote',
  'txt',
  'md',
  'markdown',
  'epub',
  'mobi',
  'azw',
  'azw3',
])

function fileExt(name: string): string | null {
  const m = name.match(/\.([A-Za-z0-9]{1,5})$/)
  return m ? m[1].toLowerCase() : null
}

function isDocument(file: {
  file_type: string
  original_name: string
}): boolean {
  if (file.file_type !== 'file') return false
  const ext = fileExt(file.original_name)
  return ext ? DOC_EXTS.has(ext) : false
}

/** Short uppercase badge label: extension if present, otherwise a type tag. */
function badgeLabel(file: FileItem): string {
  const ext = fileExt(file.original_name)
  if (ext) return ext.toUpperCase()
  switch (file.file_type) {
    case 'image':
      return 'IMG'
    case 'video':
      return 'VID'
    case 'audio':
      return 'AUD'
    default:
      return 'FILE'
  }
}

/* ────────────────────────────────────────────────────────────
 * FilterPill — rounded pill with icon + label + count
 * ──────────────────────────────────────────────────────────── */
function FilterPill({
  icon: Icon,
  label,
  count,
  active,
  onClick,
}: {
  icon: React.ElementType
  label: string
  count: number
  active: boolean
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        'inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-xs font-medium transition-colors',
        active
          ? 'border-foreground/30 bg-foreground text-background'
          : 'border-border text-muted-foreground hover:border-foreground/30 hover:text-foreground'
      )}
    >
      <Icon className="size-3.5" strokeWidth={1.8} />
      <span>{label}</span>
      <span className="tabular-nums opacity-70">{count.toLocaleString()}</span>
    </button>
  )
}

/* ────────────────────────────────────────────────────────────
 * FileVisual — square visual area, branches by file_type
 * ──────────────────────────────────────────────────────────── */
function FileVisual({ file }: { file: FileItem }) {
  const [imgSrc, setImgSrc] = useState(file.thumb_url || file.url || '')
  const [imgFailed, setImgFailed] = useState(false)

  useEffect(() => {
    setImgSrc(file.thumb_url || file.url || '')
    setImgFailed(false)
  }, [file.id, file.thumb_url, file.url])

  const ext = fileExt(file.original_name) ?? ''

  // Images: use the real thumbnail when we have one; fall back to the
  // HTML design's per-file hue gradient (+ soft radial highlight + IMG tag)
  // so the placeholder reads as an image placeholder even without a URL.
  if (file.file_type === 'image') {
    if (imgSrc && !imgFailed) {
      // Transparency checkerboard behind the image — `object-contain` fits
      // the whole source inside the square so transparent PNGs/SVGs reveal
      // the grid through their transparent regions (matching the preview
      // modal behaviour) instead of hiding it behind a cropped `cover` fill.
      return (
        <div className="checker-bg h-full w-full">
          <img
            src={imgSrc}
            alt={file.original_name}
            loading="lazy"
            className="h-full w-full object-contain transition-transform duration-300 group-hover:scale-[1.03]"
            onError={() => {
              if (imgSrc !== file.url && file.url) {
                setImgSrc(file.url)
                return
              }
              setImgFailed(true)
            }}
          />
        </div>
      )
    }
    const hue = hashHue(file.id)
    const hue2 = (hue + 40) % 360
    return (
      <div
        className="relative h-full w-full"
        style={{
          background: `linear-gradient(135deg, hsl(${hue} 60% 70%), hsl(${hue2} 60% 55%))`,
        }}
      >
        <div
          className="absolute inset-0 opacity-40"
          style={{
            backgroundImage:
              'radial-gradient(circle at 30% 25%, rgba(255,255,255,0.4), transparent 60%)',
          }}
        />
        <div className="absolute bottom-1.5 left-1.5 rounded bg-black/30 px-1.5 py-0.5 font-mono text-[9px] text-white backdrop-blur">
          IMG
        </div>
      </div>
    )
  }

  // Videos: slate gradient + diagonal stripes, centered play puck.
  if (file.file_type === 'video') {
    // Deterministic placeholder duration so the badge is consistent across
    // renders. Anything beats showing the same "0:00" on every tile.
    const secs = hashHue(file.id) % 60
    return (
      <div className="relative h-full w-full bg-gradient-to-br from-slate-800 to-slate-900">
        <div className="striped-placeholder absolute inset-0 opacity-10" />
        <div className="absolute inset-0 flex items-center justify-center">
          <div className="flex size-9 items-center justify-center rounded-full bg-white/90 text-slate-900">
            <Video className="size-4" strokeWidth={1.8} />
          </div>
        </div>
        <div className="absolute bottom-1.5 right-1.5 rounded bg-black/40 px-1.5 py-0.5 font-mono text-[9px] text-white">
          0:{String(secs).padStart(2, '0')}
        </div>
      </div>
    )
  }

  // Audio: hue gradient + tiny waveform bars.
  if (file.file_type === 'audio') {
    const hue = hashHue(file.id)
    return (
      <div
        className="relative h-full w-full"
        style={{
          background: `linear-gradient(135deg, hsl(${hue} 50% 85%), hsl(${hue} 40% 70%))`,
        }}
      >
        <div className="absolute inset-0 flex items-center justify-center gap-0.5">
          {Array.from({ length: 18 }).map((_, i) => {
            const h = 20 + Math.abs(Math.sin(i + hue)) * 40
            return (
              <span
                key={i}
                className="w-1 rounded-full bg-foreground/60"
                style={{ height: `${h}%` }}
              />
            )
          })}
        </div>
      </div>
    )
  }

  // Documents + everything else: muted background with diagonal stripes
  // and a centered icon/ext stack.
  return (
    <div className="relative flex h-full w-full items-center justify-center bg-muted">
      <div className="striped-placeholder absolute inset-0 opacity-30" />
      <div className="relative flex flex-col items-center gap-1">
        <FileText className="size-7 text-muted-foreground" strokeWidth={1.6} />
        {ext && (
          <span className="font-mono text-[9px] uppercase text-muted-foreground">
            {ext}
          </span>
        )}
      </div>
    </div>
  )
}

/* ────────────────────────────────────────────────────────────
 * Tile — grid card with checkbox + wrapped hover action bar
 * ──────────────────────────────────────────────────────────── */
function Tile({
  file,
  selected,
  anySelected,
  copied,
  moreLabel,
  downloadLabel,
  deleteLabel,
  onOpen,
  onToggle,
  onCopy,
  onDownload,
  onDelete,
}: {
  file: FileItem
  selected: boolean
  anySelected: boolean
  copied: boolean
  moreLabel: string
  downloadLabel: string
  deleteLabel: string
  onOpen: () => void
  onToggle: () => void
  onCopy: () => void
  onDownload: () => void
  onDelete: () => void
}) {
  const badge = badgeLabel(file)

  const handleTileClick = () => {
    if (anySelected) onToggle()
    else onOpen()
  }

  return (
    <div
      onClick={handleTileClick}
      className={cn(
        'group relative flex cursor-pointer flex-col overflow-hidden rounded-lg border bg-card transition-all',
        selected
          ? 'ring-2 ring-foreground'
          : 'hover:border-foreground/20 hover:shadow-md'
      )}
    >
      <div className="relative aspect-square overflow-hidden">
        <FileVisual file={file} />

        {/* checkbox (top-left) — hover/selected visibility */}
        <button
          type="button"
          aria-label={selected ? 'Deselect' : 'Select'}
          onClick={(e) => {
            e.stopPropagation()
            onToggle()
          }}
          className={cn(
            'absolute left-2 top-2 flex size-5 items-center justify-center rounded-md border bg-background/80 backdrop-blur transition-opacity',
            selected
              ? 'border-foreground bg-foreground text-background opacity-100'
              : 'opacity-0 group-hover:opacity-100',
            anySelected && !selected && 'opacity-100'
          )}
        >
          {selected && <Check className="size-3" strokeWidth={3} />}
        </button>

        {/* hover action bar (top-right) — hidden during multi-select */}
        {!anySelected && (
          <div className="absolute right-2 top-2 opacity-0 transition-opacity group-hover:opacity-100">
            <div className="flex items-center gap-1 rounded-md border bg-background/80 p-0.5 backdrop-blur">
              <button
                type="button"
                aria-label="Preview"
                onClick={(e) => {
                  e.stopPropagation()
                  onOpen()
                }}
                className="flex size-5 items-center justify-center rounded hover:bg-accent"
              >
                <Eye className="size-3" strokeWidth={1.8} />
              </button>
              <button
                type="button"
                aria-label="Copy link"
                onClick={(e) => {
                  e.stopPropagation()
                  onCopy()
                }}
                className="flex size-5 items-center justify-center rounded hover:bg-accent"
              >
                {copied ? (
                  <Check className="size-3" strokeWidth={2} />
                ) : (
                  <Link2 className="size-3" strokeWidth={1.8} />
                )}
              </button>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button
                    type="button"
                    aria-label={moreLabel}
                    onClick={(e) => e.stopPropagation()}
                    className="flex size-5 items-center justify-center rounded hover:bg-accent"
                  >
                    <MoreHorizontal className="size-3" strokeWidth={1.8} />
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent
                  align="end"
                  onClick={(e) => e.stopPropagation()}
                >
                  <DropdownMenuItem
                    onSelect={(e) => {
                      e.preventDefault()
                      onDownload()
                    }}
                  >
                    <Download className="size-3.5" />
                    {downloadLabel}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    variant="destructive"
                    onSelect={(e) => {
                      e.preventDefault()
                      onDelete()
                    }}
                  >
                    <Trash2 className="size-3.5" />
                    {deleteLabel}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        )}
      </div>

      <div className="min-w-0 p-2.5">
        <div
          className="truncate text-xs font-medium"
          title={file.original_name}
        >
          {file.original_name}
        </div>
        <div className="mt-0.5 flex items-center justify-between text-[10px] tabular-nums text-muted-foreground">
          <span className="uppercase">{badge}</span>
          <span>{formatBytes(file.size_bytes)}</span>
        </div>
      </div>
    </div>
  )
}

/* ────────────────────────────────────────────────────────────
 * FilesPage
 * ──────────────────────────────────────────────────────────── */
export default function FilesPage() {
  const { t, locale } = useI18n()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  // Filter state accepts backend-native values ("", "image", "video", "audio",
  // "file") plus a frontend-only value "document" which sub-filters `file` by
  // extension. See DOC_EXTS / isDocument().
  const [fileType, setFileType] = useState('')
  const [uploadOpen, setUploadOpen] = useState(false)
  const [detailFile, setDetailFile] = useState<FileItem | null>(null)
  const [pendingDelete, setPendingDelete] = useState<PendingDelete | null>(null)
  const [bulkMoveOpen, setBulkMoveOpen] = useState(false)
  const [bulkMoveTarget, setBulkMoveTarget] = useState<string>('__root__')
  const [uploads, setUploads] = useState<UploadTask[]>([])
  const [copied, setCopied] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [selected, setSelected] = useState<Set<string>>(() => new Set())
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (searchParams.get('upload') !== '1') return
    setUploadOpen(true)

    const next = new URLSearchParams(searchParams)
    next.delete('upload')
    setSearchParams(next, { replace: true })
  }, [searchParams, setSearchParams])

  const {
    gridRef,
    pageSize: adaptiveGridPageSize,
    paginationRef,
  } = useAdaptiveGridPageSize({
    enabled: viewMode === 'grid',
    defaultPageSize: DEFAULT_GRID_PAGE_SIZE,
    targetRows: 4,
    minRows: 2,
    maxRows: 4,
  })
  const pageSize = viewMode === 'grid' ? adaptiveGridPageSize : LIST_PAGE_SIZE

  // Reset pagination when the adaptive page size changes. Adjusting state
  // during render sidesteps the `react-hooks/set-state-in-effect` rule.
  const [trackedPageSize, setTrackedPageSize] = useState(pageSize)
  if (trackedPageSize !== pageSize) {
    setTrackedPageSize(pageSize)
    setPage(1)
  }

  /* ── data ─────────────────────────────────────────────── */
  // "document" is frontend-only; translate to backend `file` before querying.
  const backendFileType = fileType === 'document' ? 'file' : fileType

  const { data, isLoading } = useQuery({
    queryKey: ['files', page, keyword, backendFileType, pageSize],
    queryFn: () =>
      fileApi
        .list({
          page,
          size: pageSize,
          keyword,
          file_type: backendFileType,
          only_self: true,
        })
        .then((r) => r.data.data),
  })

  const { data: stats } = useQuery<UserStats>({
    queryKey: ['files', 'stats'],
    queryFn: () => statsApi.get().then((r) => r.data.data),
    staleTime: 30_000,
  })

  const { data: authOptions } = useQuery<PublicAuthOptions>({
    queryKey: ['auth', 'options'],
    queryFn: () => authApi.options().then((r) => r.data.data),
    staleTime: 30_000,
  })

  // Supplementary query: fetch a bounded window of `file_type=file` items so we
  // can derive the document/other split by extension. Cap at 1000 — for typical
  // workspaces this is enough; very large estates accept approximate counts.
  const { data: fileBucketSample } = useQuery<{
    items: FileItem[]
    total: number
  }>({
    queryKey: ['files', 'fileBucketSample'],
    queryFn: () =>
      fileApi
        .list({ page: 1, size: 1000, file_type: 'file', only_self: true })
        .then((r) => r.data.data),
    staleTime: 60_000,
  })

  const { documentCount, otherCount } = useMemo(() => {
    const sample = fileBucketSample?.items ?? []
    let doc = 0
    for (const f of sample) if (isDocument(f)) doc++
    const other = sample.length - doc
    return { documentCount: doc, otherCount: other }
  }, [fileBucketSample])

  /* ── mutations ────────────────────────────────────────── */
  const deleteMutation = useMutation({
    mutationFn: (id: string) => fileApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['files'] })
    },
  })

  /* ── selection ────────────────────────────────────────── */
  const toggleSelect = useCallback((id: string) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }, [])

  const clearSelection = useCallback(() => setSelected(new Set()), [])

  // Selection is page-scoped — reset when the filter identity changes.
  // Adjust during render to avoid the effect-based cascade.
  const filterIdentity = `${page}|${keyword}|${fileType}`
  const [trackedFilterIdentity, setTrackedFilterIdentity] =
    useState(filterIdentity)
  if (trackedFilterIdentity !== filterIdentity) {
    setTrackedFilterIdentity(filterIdentity)
    setSelected(new Set())
  }

  const uploadMaxFileSizeBytes = resolveUploadMaxFileSizeBytes(authOptions)

  /* ── uploads ──────────────────────────────────────────── */
  const uploadFiles = useCallback(
    (files: FileList | File[]) => {
      const acceptedFiles = Array.from(files).filter((file) => {
        if (file.size <= uploadMaxFileSizeBytes) return true
        toast.error(
          t('files.uploadTooLarge')
            .replace('{name}', file.name)
            .replace('{size}', formatSize(uploadMaxFileSizeBytes))
        )
        return false
      })
      if (acceptedFiles.length === 0) return

      const newTasks: UploadTask[] = acceptedFiles.map((file) => ({
        id: Math.random().toString(36).slice(2, 10),
        file,
        progress: 0,
        status: 'uploading' as const,
      }))
      setUploads((prev) => [...prev, ...newTasks])

      newTasks.forEach((task) => {
        const formData = new FormData()
        formData.append('file', task.file)

        const xhr = new XMLHttpRequest()
        xhr.open('POST', '/api/v1/upload')
        // Auth cookies are HttpOnly and same-origin; withCredentials tells
        // the browser to attach them on this cross-instance XHR call the
        // same way axios would with its withCredentials flag.
        xhr.withCredentials = true

        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            const pct = Math.min(90, Math.round((e.loaded / e.total) * 90))
            setUploads((prev) =>
              prev.map((u) => (u.id === task.id ? { ...u, progress: pct } : u))
            )
          }
        }
        xhr.upload.onload = () => {
          setUploads((prev) =>
            prev.map((u) =>
              u.id === task.id
                ? { ...u, status: 'processing', progress: 95 }
                : u
            )
          )
        }
        xhr.onload = () => {
          if (xhr.status === 200) {
            setUploads((prev) =>
              prev.map((u) =>
                u.id === task.id ? { ...u, status: 'done', progress: 100 } : u
              )
            )
            queryClient.invalidateQueries({ queryKey: ['files'] })
            toast.success(
              t('files.uploadOneSuccess').replace('{name}', task.file.name)
            )
          } else {
            setUploads((prev) =>
              prev.map((u) =>
                u.id === task.id ? { ...u, status: 'error' } : u
              )
            )
            toast.error(
              readUploadErrorMessage(xhr, t('toast.tooManyRequests')) ??
                t('files.uploadOneFailed').replace('{name}', task.file.name)
            )
          }
        }
        xhr.onerror = () => {
          setUploads((prev) =>
            prev.map((u) => (u.id === task.id ? { ...u, status: 'error' } : u))
          )
        }
        xhr.send(formData)
      })
    },
    [queryClient, t, uploadMaxFileSizeBytes]
  )

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault()
      if (e.dataTransfer.files.length > 0) uploadFiles(e.dataTransfer.files)
    },
    [uploadFiles]
  )

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      uploadFiles(e.target.files)
      e.target.value = ''
    }
  }

  const copyUrl = (text: string, key: string) => {
    navigator.clipboard.writeText(text)
    toast.success(t('files.linkCopied'))
    setCopied(key)
    setTimeout(() => setCopied(''), 1500)
  }

  const clearDone = () =>
    setUploads((prev) => prev.filter((u) => u.status === 'uploading'))

  /* ── bulk actions ─────────────────────────────────────── */
  const rawItems: FileItem[] = useMemo(() => data?.items ?? [], [data])
  // When the user picks a frontend-only filter (document / file-other), the
  // backend returns all `file_type=file` rows — narrow them client-side here.
  const items = useMemo(() => {
    if (fileType === 'document') return rawItems.filter(isDocument)
    if (fileType === 'file') return rawItems.filter((f) => !isDocument(f))
    return rawItems
  }, [rawItems, fileType])
  const selectedFiles = useMemo(
    () => items.filter((f) => selected.has(f.id)),
    [items, selected]
  )

  const { data: folderOptions = [], isLoading: folderOptionsLoading } =
    useQuery<FolderOption[]>({
      queryKey: ['albums', 'all-options'],
      enabled: bulkMoveOpen,
      queryFn: async () => {
        const options: FolderOption[] = []

        const walk = async (parentId: string | null, prefix: string) => {
          let page = 1
          while (true) {
            const params: Record<string, string | number> = { page, size: 100 }
            if (parentId) params.parent_id = parentId
            const res = await albumApi.list(params)
            const payload = res.data?.data as {
              items?: FolderItem[]
              total?: number
              size?: number
            }
            const rows = payload.items ?? []
            if (rows.length === 0) break

            for (const row of rows) {
              const label = prefix ? `${prefix} / ${row.name}` : row.name
              options.push({ id: row.id, label })
              await walk(row.id, label)
            }

            const total = Number(payload.total ?? rows.length)
            const pageSize = Number(payload.size ?? rows.length ?? 1)
            if (page * pageSize >= total) break
            page += 1
          }
        }

        await walk(null, '')
        options.sort((a, b) => a.label.localeCompare(b.label))
        return options
      },
      staleTime: 60_000,
    })

  const bulkDownload = () => {
    if (selectedFiles.length === 0) return
    toast.info(
      t('files.downloadStarting').replace('{n}', String(selectedFiles.length))
    )
    // Trigger a download for each file. Using <a download> so the browser opts
    // into saving rather than navigating when the mime type is previewable.
    selectedFiles.forEach((file) => {
      const a = document.createElement('a')
      a.href = file.url
      a.download = file.original_name
      a.target = '_blank'
      a.rel = 'noopener noreferrer'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
    })
  }

  const bulkDelete = () => {
    if (selectedFiles.length === 0) return
    setPendingDelete({
      ids: selectedFiles.map((f) => f.id),
    })
  }

  const requestSingleDelete = useCallback(
    (file: FileItem, options?: { closeDetail?: boolean }) => {
      setPendingDelete({
        ids: [file.id],
        displayName: file.original_name,
        closeDetail: options?.closeDetail,
      })
    },
    []
  )

  const confirmDelete = useCallback(() => {
    if (!pendingDelete) return

    const ids = pendingDelete.ids
    const isSingle = ids.length === 1

    if (pendingDelete.closeDetail) {
      setDetailFile(null)
    }
    setPendingDelete(null)

    if (isSingle) {
      deleteMutation.mutate(ids[0], {
        onSuccess: () => toast.success(t('files.deleteSuccess')),
        onError: () => toast.error(t('files.deleteFailed')),
      })
      return
    }

    Promise.allSettled(ids.map((id) => fileApi.delete(id))).then((results) => {
      const ok = results.filter((r) => r.status === 'fulfilled').length
      queryClient.invalidateQueries({ queryKey: ['files'] })
      queryClient.invalidateQueries({ queryKey: ['files', 'stats'] })
      if (ok > 0) {
        toast.success(t('files.bulkDeleteSuccess').replace('{n}', String(ok)))
      }
      if (ok < ids.length) {
        toast.error(t('files.deleteFailed'))
      }
      clearSelection()
    })
  }, [clearSelection, deleteMutation, pendingDelete, queryClient, t])

  const bulkMove = () => {
    if (selectedFiles.length === 0) return
    setBulkMoveTarget('__root__')
    setBulkMoveOpen(true)
  }

  const confirmBulkMove = () => {
    if (selectedFiles.length === 0) return
    const folderId = bulkMoveTarget === '__root__' ? null : bulkMoveTarget

    Promise.allSettled(
      selectedFiles.map((f) => fileApi.move(f.id, folderId))
    ).then((results) => {
      const ok = results.filter((r) => r.status === 'fulfilled').length
      queryClient.invalidateQueries({ queryKey: ['files'] })
      if (ok > 0) {
        toast.success(t('files.bulkMoveSuccess').replace('{n}', String(ok)))
      }
      if (ok < selectedFiles.length) {
        toast.error(t('albums.moveFailed'))
      }
      setBulkMoveOpen(false)
      clearSelection()
    })
  }

  const bulkShare = async () => {
    if (selectedFiles.length === 0) return
    // Universal share page — clicking the link lands on /share/:hash where a
    // type-aware preview is rendered, regardless of whether the file is an
    // image, video, audio, PDF or anything else.
    const origin = window.location.origin
    const text = selectedFiles
      .map((f) => `${origin}/share/${f.hash_md5}`)
      .join('\n')
    try {
      await navigator.clipboard.writeText(text)
      toast.success(
        t('files.bulkShareCopied').replace('{n}', String(selectedFiles.length))
      )
    } catch {
      toast.error(t('files.bulkShareFailed'))
    }
  }

  /* ── filter pills ─────────────────────────────────────── */
  // When the sample query covered the full bucket, counts are exact; otherwise
  // (sample < stats.others) we fall back to stats.others for 其它 and leave
  // 文档 at its sample-derived count.
  const sampleCovered =
    fileBucketSample != null &&
    stats != null &&
    fileBucketSample.total <= (fileBucketSample.items?.length ?? 0)
  const otherPillCount = sampleCovered
    ? otherCount
    : Math.max(0, (stats?.others ?? 0) - documentCount)

  const filterPills = useMemo(
    () => [
      {
        value: '',
        icon: Layers,
        label: t('common.all'),
        count: stats?.total_files ?? 0,
      },
      {
        value: 'image',
        icon: ImageIcon,
        label: t('files.images'),
        count: stats?.images ?? 0,
      },
      {
        value: 'video',
        icon: Video,
        label: t('files.videos'),
        count: stats?.videos ?? 0,
      },
      {
        value: 'audio',
        icon: Music,
        label: t('files.audio'),
        count: stats?.audios ?? 0,
      },
      {
        value: 'document',
        icon: FileText,
        label: t('files.documents'),
        count: documentCount,
      },
      {
        value: 'file',
        icon: FileIcon,
        label: t('files.otherFiles'),
        count: otherPillCount,
      },
    ],
    [stats, t, documentCount, otherPillCount]
  )

  const anySelected = selected.size > 0

  return (
    <div className="space-y-6">
      {/* ── Header ─────────────────────────────────────── */}
      <PageHeader
        title={t('files.title')}
        description={t('files.description')}
        actions={
          <>
            <Button
              variant="outline"
              size="sm"
              onClick={() => navigate('/user/folders')}
            >
              <Folder className="size-3.5" />
              {t('files.newFolder')}
            </Button>
            <Dialog open={uploadOpen} onOpenChange={setUploadOpen}>
              <DialogTrigger asChild>
                <Button size="sm">
                  <Upload className="size-3.5" />
                  {t('common.upload')}
                </Button>
              </DialogTrigger>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>{t('files.uploadFile')}</DialogTitle>
                </DialogHeader>
                <div
                  onDragOver={(e) => e.preventDefault()}
                  onDrop={handleDrop}
                  onClick={() => fileInputRef.current?.click()}
                  className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-10 text-center transition-colors hover:border-primary/50 hover:bg-accent/30"
                >
                  <Upload className="mb-3 size-8 text-muted-foreground" />
                  <p className="text-sm font-medium">
                    {t('files.dragOrClick')}
                  </p>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {t('files.supportedTypes')}
                  </p>
                  <p className="mt-1 text-[11px] text-muted-foreground">
                    {t('files.maxUploadSize').replace(
                      '{size}',
                      formatSize(uploadMaxFileSizeBytes)
                    )}
                  </p>
                  <input
                    ref={fileInputRef}
                    type="file"
                    multiple
                    className="hidden"
                    onChange={handleFileSelect}
                  />
                </div>

                {uploads.length > 0 && (
                  <div className="max-h-48 space-y-2 overflow-y-auto">
                    <div className="flex items-center justify-between">
                      <span className="text-xs text-muted-foreground">
                        {uploads.filter((u) => u.status === 'done').length}/
                        {uploads.length} {t('files.completed')}
                      </span>
                      {uploads.some(
                        (u) => u.status === 'done' || u.status === 'error'
                      ) && (
                        <button
                          onClick={clearDone}
                          className="text-xs text-muted-foreground hover:text-foreground"
                        >
                          {t('files.clearDone')}
                        </button>
                      )}
                    </div>
                    {uploads.map((task) => (
                      <div
                        key={task.id}
                        className="flex items-center gap-3 rounded-lg border px-3 py-2.5"
                      >
                        <div className="min-w-0 flex-1">
                          <p className="truncate text-xs font-medium">
                            {task.file.name}
                          </p>
                          <Progress
                            className="mt-1.5 h-1"
                            value={task.progress}
                            indicatorClassName={
                              task.status === 'error'
                                ? 'bg-destructive'
                                : task.status === 'done'
                                  ? 'bg-emerald-500'
                                  : task.status === 'processing'
                                    ? 'bg-amber-500'
                                    : undefined
                            }
                          />
                        </div>
                        <span className="shrink-0 text-[10px] text-muted-foreground">
                          {task.status === 'error' ? (
                            t('files.failed')
                          ) : task.status === 'done' ? (
                            <Check className="size-3.5 text-emerald-500" />
                          ) : task.status === 'processing' ? (
                            <span className="inline-flex items-center gap-1">
                              <Loader2 className="size-3 animate-spin text-amber-500" />
                              {t('files.processing')}
                            </span>
                          ) : (
                            `${task.progress}%`
                          )}
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </DialogContent>
            </Dialog>
          </>
        }
      />

      {/* ── Filter pills + search + view toggle (one row) ── */}
      <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
        <div className="flex flex-wrap items-center gap-1.5">
          {filterPills.map((pill) => (
            <FilterPill
              key={pill.value}
              icon={pill.icon}
              label={pill.label}
              count={pill.count}
              active={fileType === pill.value}
              onClick={() => {
                setFileType(pill.value)
                setPage(1)
              }}
            />
          ))}
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder={t('files.searchFiles')}
              value={keyword}
              onChange={(e) => {
                setKeyword(e.target.value)
                setPage(1)
              }}
              className="h-8 w-48 pl-8 text-xs"
            />
          </div>
          <div className="inline-flex h-8 items-center justify-center rounded-lg bg-muted/70 p-1 text-muted-foreground">
            <button
              type="button"
              onClick={() => setViewMode('grid')}
              className={cn(
                'inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-2.5 py-0.5 text-xs font-medium transition-all',
                viewMode === 'grid'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'hover:text-foreground'
              )}
            >
              <LayoutGrid className="size-3.5" strokeWidth={1.8} />
              {t('files.gridView')}
            </button>
            <button
              type="button"
              onClick={() => setViewMode('list')}
              className={cn(
                'inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-2.5 py-0.5 text-xs font-medium transition-all',
                viewMode === 'list'
                  ? 'bg-background text-foreground shadow-sm'
                  : 'hover:text-foreground'
              )}
            >
              <List className="size-3.5" strokeWidth={1.8} />
              {t('files.listView')}
            </button>
          </div>
        </div>
      </div>

      {/* ── Selection toolbar — additive, appears below filter row ── */}
      {anySelected && (
        <div className="flex items-center justify-between rounded-lg border bg-card px-3 py-2">
          <div className="flex items-center gap-2 text-xs">
            <span className="inline-flex size-5 items-center justify-center rounded-md bg-foreground text-background">
              <Check className="size-3" strokeWidth={3} />
            </span>
            {renderWithBold(t('files.selectedN'), {
              '{n}': <b>{selected.size}</b>,
            })}
          </div>
          <div className="flex items-center gap-1">
            <Button size="xs" variant="ghost" onClick={bulkDownload}>
              <Download className="size-3" />
              {t('files.download')}
            </Button>
            <Button size="xs" variant="ghost" onClick={bulkMove}>
              <Folder className="size-3" />
              {t('files.move')}
            </Button>
            <Button size="xs" variant="ghost" onClick={bulkShare}>
              <Share2 className="size-3" />
              {t('files.share')}
            </Button>
            <Button
              size="xs"
              variant="ghost"
              onClick={bulkDelete}
              className="text-destructive hover:text-destructive"
            >
              <Trash2 className="size-3" />
              {t('common.delete')}
            </Button>
            <Button
              size="icon-xs"
              variant="ghost"
              onClick={clearSelection}
              aria-label={t('files.close')}
            >
              <X className="size-3" />
            </Button>
          </div>
        </div>
      )}

      {/* ── Info bar — loaded count + virtual scroll hint ── */}
      {!isLoading && data && (
        <div className="flex items-center justify-between text-[11px] text-muted-foreground">
          <span className="tabular-nums">
            {renderWithBold(t('files.loadedCount'), {
              '{n}': (
                <b className="text-foreground">
                  {items.length.toLocaleString()}
                </b>
              ),
              '{total}': data.total.toLocaleString(),
            })}
          </span>
          <span className="inline-flex items-center gap-1 opacity-70">
            <Zap className="size-3" strokeWidth={1.8} />
            {t('files.virtualScroll')}
          </span>
        </div>
      )}

      {/* ── Grid / List ────────────────────────────────── */}
      {isLoading ? (
        viewMode === 'grid' ? (
          <div
            ref={gridRef}
            className="grid grid-cols-2 gap-3 sm:[grid-template-columns:repeat(auto-fill,minmax(180px,1fr))]"
          >
            {Array.from({ length: pageSize }).map((_, i) => (
              <Skeleton key={i} className="aspect-square rounded-lg" />
            ))}
          </div>
        ) : (
          <div className="space-y-2">
            {Array.from({ length: 10 }).map((_, i) => (
              <Skeleton key={i} className="h-16 rounded-lg" />
            ))}
          </div>
        )
      ) : (
        <>
          {viewMode === 'grid' ? (
            <div
              ref={gridRef}
              className="grid grid-cols-2 gap-3 sm:[grid-template-columns:repeat(auto-fill,minmax(180px,1fr))]"
            >
              {items.map((file) => (
                <Tile
                  key={file.id}
                  file={file}
                  selected={selected.has(file.id)}
                  anySelected={anySelected}
                  copied={copied === file.id}
                  moreLabel={t('files.more')}
                  downloadLabel={t('files.download')}
                  deleteLabel={t('common.delete')}
                  onOpen={() => setDetailFile(file)}
                  onToggle={() => toggleSelect(file.id)}
                  onCopy={() => copyUrl(file.url, file.id)}
                  onDownload={() => {
                    const a = document.createElement('a')
                    a.href = file.url
                    a.download = file.original_name
                    a.target = '_blank'
                    a.rel = 'noopener noreferrer'
                    document.body.appendChild(a)
                    a.click()
                    document.body.removeChild(a)
                  }}
                  onDelete={() => {
                    requestSingleDelete(file)
                  }}
                />
              ))}
            </div>
          ) : (
            <div className="space-y-2">
              {items.map((file) => {
                const isSelected = selected.has(file.id)
                const badge = badgeLabel(file)
                return (
                  <div
                    key={file.id}
                    onClick={() => {
                      if (anySelected) toggleSelect(file.id)
                      else setDetailFile(file)
                    }}
                    className={cn(
                      'group flex cursor-pointer items-center gap-3 rounded-lg border bg-card px-3 py-2.5 transition-colors hover:border-foreground/20',
                      isSelected && 'ring-2 ring-foreground'
                    )}
                  >
                    <button
                      type="button"
                      aria-label={isSelected ? 'Deselect' : 'Select'}
                      onClick={(e) => {
                        e.stopPropagation()
                        toggleSelect(file.id)
                      }}
                      className={cn(
                        'flex size-5 shrink-0 items-center justify-center rounded-md border transition-all',
                        isSelected
                          ? 'border-foreground bg-foreground text-background'
                          : 'border-foreground/30 text-transparent opacity-0 group-hover:opacity-100',
                        anySelected && !isSelected && 'opacity-100'
                      )}
                    >
                      <Check className="size-3" strokeWidth={3} />
                    </button>
                    <div className="relative size-10 shrink-0 overflow-hidden rounded-md border">
                      <FileVisual file={file} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {file.original_name}
                      </p>
                      <div className="mt-0.5 flex items-center gap-2 text-[11px] tabular-nums text-muted-foreground">
                        <span className="font-mono uppercase">{badge}</span>
                        <span className="text-muted-foreground/60">·</span>
                        <span>{formatBytes(file.size_bytes)}</span>
                        <span className="text-muted-foreground/60">·</span>
                        <span>
                          {formatRelativeTime(file.created_at, locale)}
                        </span>
                      </div>
                    </div>
                    {!anySelected && (
                      <div className="flex shrink-0 items-center gap-1">
                        <Button
                          size="icon-xs"
                          variant="ghost"
                          onClick={(e) => {
                            e.stopPropagation()
                            copyUrl(file.url, file.id)
                          }}
                        >
                          {copied === file.id ? (
                            <Check className="size-3" />
                          ) : (
                            <Copy className="size-3" />
                          )}
                        </Button>
                        <Button
                          size="icon-xs"
                          variant="ghost"
                          onClick={(e) => {
                            e.stopPropagation()
                            requestSingleDelete(file)
                          }}
                        >
                          <Trash2 className="size-3 text-destructive" />
                        </Button>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          {items.length === 0 && (
            <EmptyKite
              title={t('files.noFiles')}
              hint={t('files.noFilesHint')}
              action={
                <Button size="sm" onClick={() => setUploadOpen(true)}>
                  <Upload className="size-3.5" />
                  {t('common.upload')}
                </Button>
              }
            />
          )}

          {data && data.total > pageSize && (
            <div
              ref={paginationRef}
              className="flex items-center justify-center gap-2"
            >
              <Button
                variant="outline"
                size="icon-sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <ChevronLeft className="size-4" />
              </Button>
              <span className="min-w-15 text-center text-sm text-muted-foreground">
                {page} / {Math.ceil(data.total / pageSize)}
              </span>
              <Button
                variant="outline"
                size="icon-sm"
                disabled={page >= Math.ceil(data.total / pageSize)}
                onClick={() => setPage((p) => p + 1)}
              >
                <ChevronRight className="size-4" />
              </Button>
            </div>
          )}
        </>
      )}

      {/* ── Detail Dialog ──────────────────────────────── */}
      <Dialog
        open={!!detailFile}
        onOpenChange={(open) => !open && setDetailFile(null)}
      >
        <DialogContent className="grid-cols-1 sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="truncate pr-8">
              {detailFile?.original_name}
            </DialogTitle>
          </DialogHeader>
          {detailFile && (
            <div className="space-y-4">
              {isImagePreviewable(detailFile) ? (
                <div className="checker-bg overflow-hidden rounded-lg border">
                  <img
                    src={detailFile.url}
                    alt={detailFile.original_name}
                    className="max-h-64 w-full object-contain"
                  />
                </div>
              ) : (
                (() => {
                  // Fallback tile for non-previewable files (PSD, PDF, zips,
                  // video/audio — anything the browser won't render in an
                  // <img>). Reuses the file-type icon + label so the dialog
                  // stays visually anchored instead of collapsing to a bare
                  // metadata grid or — worse — showing a broken-image glyph.
                  const info = getFileIconInfo(detailFile)
                  const Icon = info.icon
                  const ext = (() => {
                    const n = detailFile.original_name ?? ''
                    const i = n.lastIndexOf('.')
                    return i > 0 ? n.substring(i + 1).toUpperCase() : ''
                  })()
                  return (
                    <div className="relative flex h-48 items-center justify-center overflow-hidden rounded-lg border bg-muted/30">
                      <div className="striped-placeholder absolute inset-0 opacity-20" />
                      <div className="relative flex flex-col items-center gap-2.5">
                        <div className="flex size-14 items-center justify-center rounded-xl border border-border/60 bg-background/80 text-muted-foreground shadow-sm backdrop-blur-sm">
                          <Icon className="size-7" strokeWidth={1.5} />
                        </div>
                        <div className="flex flex-col items-center gap-0.5">
                          <span className="text-sm font-medium">
                            {t(info.labelKey)}
                          </span>
                          {ext && (
                            <span className="font-mono text-[10px] uppercase tracking-wider text-muted-foreground">
                              .{ext}
                            </span>
                          )}
                        </div>
                      </div>
                    </div>
                  )
                })()
              )}

              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t('common.type')}
                  </span>
                  <p className="font-medium">
                    {getFileTypeLabel(detailFile, t)}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">MIME</span>
                  <p className="text-xs font-medium break-all">
                    {detailFile.mime_type}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t('common.size')}
                  </span>
                  <p className="font-medium">
                    {formatBytes(detailFile.size_bytes)}
                  </p>
                </div>
                {detailFile.width && detailFile.height && (
                  <div>
                    <span className="text-xs text-muted-foreground">
                      {t('files.dimensions')}
                    </span>
                    <p className="font-medium">
                      {detailFile.width} x {detailFile.height}
                    </p>
                  </div>
                )}
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t('common.date')}
                  </span>
                  <p className="font-medium">
                    {new Date(detailFile.created_at).toLocaleString()}
                  </p>
                </div>
              </div>

              <div className="space-y-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t('files.linkFormats')}
                </span>
                {[
                  { label: 'URL', value: detailFile.url },
                  ...(detailFile.source_url
                    ? [
                        {
                          label: t('files.sourceUrl'),
                          value: detailFile.source_url,
                        },
                      ]
                    : []),
                  {
                    label: 'Markdown',
                    value: `![${detailFile.original_name}](${detailFile.url})`,
                  },
                  {
                    label: 'HTML',
                    value: `<img src="${detailFile.url}" alt="${detailFile.original_name}">`,
                  },
                  { label: 'BBCode', value: `[img]${detailFile.url}[/img]` },
                ].map((link) => (
                  <div
                    key={link.label}
                    className="flex items-center gap-2 rounded-md bg-muted/50 px-3 py-2"
                  >
                    <span className="w-16 shrink-0 text-xs text-muted-foreground">
                      {link.label}
                    </span>
                    <div className="min-w-0 flex-1 overflow-x-auto [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
                      <code className="block w-max min-w-full whitespace-nowrap bg-transparent text-xs text-foreground/90">
                        {link.value}
                      </code>
                    </div>
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      onClick={() => copyUrl(link.value, link.label)}
                    >
                      {copied === link.label ? (
                        <Check className="size-3" />
                      ) : (
                        <Copy className="size-3" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>

              <div className="flex justify-end gap-2">
                <Button size="sm" variant="outline" asChild>
                  <a
                    href={detailFile.url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    <ExternalLink className="size-3.5" />
                    {t('files.openOriginal')}
                  </a>
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => {
                    requestSingleDelete(detailFile, { closeDetail: true })
                  }}
                >
                  <Trash2 className="size-3.5" />
                  {t('common.delete')}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={!!pendingDelete}
        onOpenChange={(open) => {
          if (!open) setPendingDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('common.delete')}</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDelete && pendingDelete.ids.length > 1
                ? t('files.bulkDeleteConfirm').replace(
                    '{n}',
                    String(pendingDelete.ids.length)
                  )
                : t('files.deleteConfirm')}
            </AlertDialogDescription>
            {pendingDelete?.displayName && (
              <div className="rounded-md border bg-muted/30 px-3 py-2 text-xs text-muted-foreground">
                {pendingDelete.displayName}
              </div>
            )}
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction onClick={confirmDelete}>
              {t('common.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={bulkMoveOpen} onOpenChange={setBulkMoveOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('files.bulkMoveTitle')}</DialogTitle>
            <DialogDescription>
              {t('files.bulkMoveDesc').replace(
                '{n}',
                String(selectedFiles.length)
              )}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-2">
            <p className="text-xs text-muted-foreground">
              {t('files.targetFolder')}
            </p>
            <Select value={bulkMoveTarget} onValueChange={setBulkMoveTarget}>
              <SelectTrigger>
                <SelectValue placeholder={t('files.targetFolder')} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__root__">
                  {t('albums.moveToRoot')}
                </SelectItem>
                {folderOptions.map((folder) => (
                  <SelectItem key={folder.id} value={folder.id}>
                    {folder.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {folderOptionsLoading && (
              <p className="text-xs text-muted-foreground">
                {t('common.loading')}
              </p>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setBulkMoveOpen(false)}>
              {t('common.cancel')}
            </Button>
            <Button onClick={confirmBulkMove}>{t('files.move')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
