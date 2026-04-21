import { useEffect, useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Settings as SettingsIcon,
  Globe,
  Upload,
  Shield,
  HardDrive,
  Mail,
  Check,
  Copy,
  Loader2,
} from 'lucide-react'
import { toast } from 'sonner'

import { authProviderApi, settingsApi, storageApi } from '@/lib/api'
import { useI18n } from '@/i18n'
import { cn, formatSize } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Textarea } from '@/components/ui/textarea'
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { PageHeader } from '@/components/page-header'
import { SocialProviderLogo } from '@/components/social-provider-logo'
import { StorageLogo, resolveLogoVendor } from '@/components/storage-logo'

type Tab = 'general' | 'site' | 'upload' | 'auth' | 'storage' | 'email'

const DEFAULT_UPLOAD_PATH_PATTERN = '{year}/{month}/{md5_8}/{uuid}.{ext}'
const DEFAULT_UPLOAD_MAX_FILE_SIZE_MB = '100'
const UPLOAD_SIZE_MB_BYTES = 1024 * 1024

interface StorageListItem {
  id: string
  name: string
  driver: string
  provider?: string
  capacity_limit_bytes: number
  used_bytes: number
  files_count?: number
  priority: number
  is_default: boolean
  is_active: boolean
}

interface OAuthProviderItem {
  key: string
  label: string
  icon_key: string
  protocol: string
  enabled: boolean
  client_id: string
  has_secret: boolean
  callback_url: string
  is_configured: boolean
  scopes: string[]
  site_url: string
  site_url_valid: boolean
}

interface OAuthProviderDraft {
  enabled: boolean
  client_id: string
  client_secret: string
}

function previewUploadPathPattern(pattern: string) {
  const source = (pattern || DEFAULT_UPLOAD_PATH_PATTERN).trim()
  const replacements: Record<string, string> = {
    '{year}': '2026',
    '{month}': '04',
    '{day}': '21',
    '{user_id}': 'user-demo',
    '{file_type}': 'image',
    '{md5}': '0123456789abcdef0123456789abcdef',
    '{md5_8}': '01234567',
    '{uuid}': '123e4567-e89b-12d3-a456-426614174000',
    '{ext}': 'png',
  }

  let rendered = source
  Object.entries(replacements).forEach(([token, value]) => {
    rendered = rendered.split(token).join(value)
  })

  rendered = rendered.replaceAll('\\', '/').replace(/\/+$/g, '')
  return rendered || DEFAULT_UPLOAD_PATH_PATTERN
}

function previewDocumentTitle(siteTitle: string, pageTitle?: string) {
  const resolvedSiteTitle = siteTitle.trim() || 'Kite'
  const resolvedPageTitle = pageTitle?.trim()
  if (!resolvedPageTitle) return resolvedSiteTitle
  return `${resolvedPageTitle} - ${resolvedSiteTitle}`
}

function parseUploadMaxFileSizeMB(raw?: string) {
  const parsed = Number.parseInt(
    (raw ?? DEFAULT_UPLOAD_MAX_FILE_SIZE_MB).trim(),
    10
  )
  if (Number.isFinite(parsed) && parsed > 0) return parsed
  return Number.parseInt(DEFAULT_UPLOAD_MAX_FILE_SIZE_MB, 10)
}

/* ────────────────────────────────────────────────────────────
 * Preference row — label+hint on the left, control on the right.
 *  Uses `divide-y` on the parent Card content to get the
 *  separator lines in the target design.
 * ──────────────────────────────────────────────────────────── */
function Preference({
  label,
  hint,
  children,
}: {
  label: string
  hint?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-3 first:pt-0 last:pb-0">
      <div className="min-w-0">
        <div className="text-sm font-medium">{label}</div>
        {hint && (
          <div className="mt-0.5 text-[11px] text-muted-foreground">{hint}</div>
        )}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  )
}

/* ────────────────────────────────────────────────────────────
 * Pill-style Tabs — mirrors the target `Tabs` component
 *  (inline-flex rounded bg-muted/70 with icon + label).
 * ──────────────────────────────────────────────────────────── */
function TabPills({
  tabs,
  value,
  onChange,
}: {
  tabs: { value: Tab; label: string; icon: React.ElementType }[]
  value: Tab
  onChange: (v: Tab) => void
}) {
  return (
    <div className="overflow-x-auto pb-1">
      <div className="inline-flex min-w-max items-center justify-center rounded-lg bg-muted/70 p-1 text-muted-foreground">
        {tabs.map((tab) => {
          const active = value === tab.value
          return (
            <button
              key={tab.value}
              type="button"
              onClick={() => onChange(tab.value)}
              className={cn(
                'inline-flex items-center justify-center gap-1.5 whitespace-nowrap rounded-md px-3 py-1 text-xs font-medium transition-all',
                active
                  ? 'bg-background text-foreground shadow-sm'
                  : 'hover:text-foreground'
              )}
            >
              <tab.icon className="size-3.5" strokeWidth={1.8} />
              {tab.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}

/* ────────────────────────────────────────────────────────────
 * SettingsPage
 * ──────────────────────────────────────────────────────────── */
export default function SettingsPage() {
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<Tab>('general')
  const [form, setForm] = useState<Record<string, string>>({})
  const [saved, setSaved] = useState(false)
  const [providerDrafts, setProviderDrafts] = useState<
    Record<string, OAuthProviderDraft>
  >({})
  const [providerTouched, setProviderTouched] = useState<
    Record<string, boolean>
  >({})

  const { data, isLoading } = useQuery({
    queryKey: ['settings'],
    queryFn: () => settingsApi.get().then((r) => r.data.data),
  })

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  const { data: providerList } = useQuery<OAuthProviderItem[]>({
    queryKey: ['auth', 'providers'],
    queryFn: () => authProviderApi.list().then((r) => r.data.data),
    enabled: tab === 'auth',
  })

  useEffect(() => {
    if (!providerList) return
    setProviderDrafts(
      Object.fromEntries(
        providerList.map((provider) => [
          provider.key,
          {
            enabled: provider.enabled,
            client_id: provider.client_id ?? '',
            client_secret: '',
          },
        ])
      )
    )
    setProviderTouched({})
  }, [providerList])

  const mutation = useMutation({
    mutationFn: () => settingsApi.update(form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['settings'] })
      queryClient.invalidateQueries({ queryKey: ['auth', 'providers'] })
      queryClient.invalidateQueries({ queryKey: ['auth', 'options'] })
      setSaved(true)
      toast.success(t('settings.saved'))
      setTimeout(() => setSaved(false), 2000)
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('toast.error')
      toast.error(msg)
    },
  })

  const updateField = (key: string, value: string) =>
    setForm((prev) => ({ ...prev, [key]: value }))

  const toggleField = (key: string) =>
    setForm((prev) => ({
      ...prev,
      [key]: prev[key] === 'true' ? 'false' : 'true',
    }))

  const resetForm = () => {
    if (data) setForm(data)
  }

  const boolOf = (key: string): boolean => form[key] === 'true'

  /* ── storage tab ─────────────────────────────────────── */
  const { data: storageList } = useQuery<StorageListItem[]>({
    queryKey: ['storage', 'list'],
    queryFn: () => storageApi.list().then((r) => r.data.data),
    enabled: tab === 'storage',
  })

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => storageApi.setDefault(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['storage', 'list'] })
      toast.success(t('settings.saved'))
    },
    onError: () => toast.error(t('toast.error')),
  })

  const saveProviderMutation = useMutation({
    mutationFn: ({
      provider,
      payload,
    }: {
      provider: string
      payload: OAuthProviderDraft
    }) =>
      authProviderApi.update(provider, {
        enabled: payload.enabled,
        client_id: payload.client_id,
        client_secret: payload.client_secret,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['auth', 'providers'] })
      queryClient.invalidateQueries({ queryKey: ['auth', 'options'] })
      toast.success(t('settings.saved'))
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('toast.error')
      toast.error(msg)
    },
  })

  /* ── early loading state ─────────────────────────────── */
  if (isLoading) {
    return (
      <div className="mx-auto max-w-3xl space-y-5">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-9 w-80" />
        <Skeleton className="h-64 rounded-xl" />
      </div>
    )
  }

  const tabs: { value: Tab; label: string; icon: React.ElementType }[] = [
    { value: 'general', label: t('settings.general'), icon: SettingsIcon },
    { value: 'site', label: t('settings.siteTab'), icon: Globe },
    { value: 'upload', label: t('settings.uploadTab'), icon: Upload },
    { value: 'auth', label: t('settings.auth'), icon: Shield },
    { value: 'storage', label: t('settings.storageTab'), icon: HardDrive },
    { value: 'email', label: t('settings.email'), icon: Mail },
  ]

  const uploadPattern =
    form['upload.path_pattern'] ?? DEFAULT_UPLOAD_PATH_PATTERN
  const uploadPatternPreview = previewUploadPathPattern(uploadPattern)
  const uploadMaxFileSizeMB =
    form['upload.max_file_size_mb'] ?? DEFAULT_UPLOAD_MAX_FILE_SIZE_MB
  const uploadMaxFileSizeBytes =
    parseUploadMaxFileSizeMB(uploadMaxFileSizeMB) * UPLOAD_SIZE_MB_BYTES
  const siteName = (form.site_name ?? '').trim()
  const siteTitle = (form.site_title ?? '').trim() || siteName || 'Kite'
  const siteFaviconURL = (form.site_favicon_url ?? '').trim() || '/favicon.svg'
  const siteTitlePreviewHome = previewDocumentTitle(siteTitle)
  const siteTitlePreviewUpload = previewDocumentTitle(siteTitle, '上传文件')
  const siteTitlePreviewExplore = previewDocumentTitle(siteTitle, '探索广场')
  const uploadPatternVariables = [
    { token: '{year}', description: t('settings.uploadVarYear') },
    { token: '{month}', description: t('settings.uploadVarMonth') },
    { token: '{day}', description: t('settings.uploadVarDay') },
    { token: '{user_id}', description: t('settings.uploadVarUserId') },
    { token: '{file_type}', description: t('settings.uploadVarFileType') },
    { token: '{md5}', description: t('settings.uploadVarMd5') },
    { token: '{md5_8}', description: t('settings.uploadVarMd58') },
    { token: '{uuid}', description: t('settings.uploadVarUuid') },
    { token: '{ext}', description: t('settings.uploadVarExt') },
  ]

  const updateProviderDraft = (
    provider: string,
    patch: Partial<OAuthProviderDraft>
  ) =>
    setProviderDrafts((prev) => ({
      ...prev,
      [provider]: {
        ...prev[provider],
        ...patch,
      },
    }))

  const getProviderDraft = (provider: OAuthProviderItem): OAuthProviderDraft =>
    providerDrafts[provider.key] ?? {
      enabled: provider.enabled,
      client_id: provider.client_id ?? '',
      client_secret: '',
    }

  const clientIdLabel = (provider: string) =>
    provider === 'wechat'
      ? t('settings.oauthAppId')
      : t('settings.oauthClientId')

  const clientSecretLabel = (provider: string) =>
    provider === 'wechat'
      ? t('settings.oauthAppSecret')
      : t('settings.oauthClientSecret')

  const getProviderStatus = (
    provider: OAuthProviderItem,
    draft: OAuthProviderDraft
  ) => {
    if (draft.enabled) return t('settings.oauthStatusEnabled')
    if (provider.is_configured) return t('settings.oauthStatusConfigured')
    return t('settings.oauthStatusEmpty')
  }

  const providerMissingFields = (
    provider: OAuthProviderItem,
    draft: OAuthProviderDraft
  ) => {
    if (!draft.enabled) return false
    return (
      !draft.client_id.trim() ||
      (!draft.client_secret.trim() && !provider.has_secret)
    )
  }

  const handleSaveProvider = (provider: OAuthProviderItem) => {
    const draft = getProviderDraft(provider)
    setProviderTouched((prev) => ({ ...prev, [provider.key]: true }))

    if (draft.enabled && !provider.site_url_valid) {
      toast.error(t('settings.oauthSiteUrlInvalid'))
      return
    }
    if (providerMissingFields(provider, draft)) {
      toast.error(t('settings.oauthMissingFields'))
      return
    }

    saveProviderMutation.mutate({
      provider: provider.key,
      payload: draft,
    })
  }

  /* ── render ──────────────────────────────────────────── */
  return (
    <div className="mx-auto max-w-3xl space-y-5">
      <PageHeader
        title={t('settings.title')}
        description={t('settings.description')}
      />

      <TabPills tabs={tabs} value={tab} onChange={setTab} />

      {/* ── General ──────────────────────────────────── */}
      {tab === 'general' && (
        <Card>
          <CardHeader>
            <CardTitle>{t('settings.generalDesc')}</CardTitle>
          </CardHeader>
          <CardContent className="divide-y">
            <Preference
              label={t('settings.allowRegistration')}
              hint={t('settings.allowRegistrationHint')}
            >
              <Switch
                checked={boolOf('allow_registration')}
                onCheckedChange={() => toggleField('allow_registration')}
              />
            </Preference>
            <Preference
              label={t('settings.defaultQuota')}
              hint={t('settings.defaultQuotaHint')}
            >
              <Input
                value={form.default_quota ?? ''}
                onChange={(e) => updateField('default_quota', e.target.value)}
                placeholder="10 GB"
                className="w-32"
              />
            </Preference>
          </CardContent>
          <CardFooter className="justify-end gap-2">
            <Button variant="outline" size="sm" onClick={resetForm}>
              {t('settings.reset')}
            </Button>
            <Button
              size="sm"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {saved ? (
                <>
                  <Check className="size-3.5" />
                  {t('settings.saved')}
                </>
              ) : mutation.isPending ? (
                t('settings.saving')
              ) : (
                t('settings.saveSettings')
              )}
            </Button>
          </CardFooter>
        </Card>
      )}

      {tab === 'site' && (
        <div className="space-y-5">
          <Card>
            <CardHeader>
              <CardTitle>{t('settings.siteBasicsTitle')}</CardTitle>
              <CardDescription>{t('settings.siteBasicsHint')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-2">
                <Label htmlFor="site-name">{t('settings.siteName')}</Label>
                <Input
                  id="site-name"
                  value={form.site_name ?? ''}
                  onChange={(e) => updateField('site_name', e.target.value)}
                  placeholder="Kite"
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteNameHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-url">{t('settings.siteUrl')}</Label>
                <Input
                  id="site-url"
                  value={form.site_url ?? ''}
                  onChange={(e) => updateField('site_url', e.target.value)}
                  placeholder={t('settings.siteUrlPlaceholder')}
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteUrlHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-title">{t('settings.siteTitle')}</Label>
                <Input
                  id="site-title"
                  value={form.site_title ?? ''}
                  onChange={(e) => updateField('site_title', e.target.value)}
                  placeholder={t('settings.siteTitlePlaceholder')}
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteTitleHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-favicon">
                  {t('settings.siteFaviconUrl')}
                </Label>
                <Input
                  id="site-favicon"
                  value={form.site_favicon_url ?? ''}
                  onChange={(e) =>
                    updateField('site_favicon_url', e.target.value)
                  }
                  placeholder={t('settings.siteFaviconUrlPlaceholder')}
                />
                <div className="flex items-center gap-3 rounded-lg border bg-muted/30 px-3 py-2">
                  <span className="flex size-8 items-center justify-center overflow-hidden rounded-md border bg-background">
                    <img
                      src={siteFaviconURL}
                      alt="favicon"
                      className="size-5 object-contain"
                    />
                  </span>
                  <div className="min-w-0">
                    <div className="text-xs font-medium">
                      {t('settings.siteFaviconPreview')}
                    </div>
                    <div className="truncate text-[11px] text-muted-foreground">
                      {siteFaviconURL}
                    </div>
                  </div>
                </div>
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteFaviconUrlHint')}
                </p>
              </div>

              <div className="space-y-2">
                <div className="text-sm font-medium">
                  {t('settings.siteTitlePreview')}
                </div>
                <div className="grid gap-2 sm:grid-cols-3">
                  <div className="rounded-lg border bg-muted/30 px-3 py-2">
                    <div className="text-[11px] text-muted-foreground">
                      {t('settings.siteTitlePreviewHome')}
                    </div>
                    <div className="mt-1 break-all text-xs font-medium">
                      {siteTitlePreviewHome}
                    </div>
                  </div>
                  <div className="rounded-lg border bg-muted/30 px-3 py-2">
                    <div className="text-[11px] text-muted-foreground">
                      {t('settings.siteTitlePreviewUpload')}
                    </div>
                    <div className="mt-1 break-all text-xs font-medium">
                      {siteTitlePreviewUpload}
                    </div>
                  </div>
                  <div className="rounded-lg border bg-muted/30 px-3 py-2">
                    <div className="text-[11px] text-muted-foreground">
                      {t('settings.siteTitlePreviewExplore')}
                    </div>
                    <div className="mt-1 break-all text-xs font-medium">
                      {siteTitlePreviewExplore}
                    </div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('settings.siteSeoTitle')}</CardTitle>
              <CardDescription>{t('settings.siteSeoHint')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-2">
                <Label htmlFor="site-keywords">
                  {t('settings.siteKeywords')}
                </Label>
                <Input
                  id="site-keywords"
                  value={form.site_keywords ?? ''}
                  onChange={(e) => updateField('site_keywords', e.target.value)}
                  placeholder={t('settings.siteKeywordsPlaceholder')}
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteKeywordsHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-description">
                  {t('settings.siteDescription')}
                </Label>
                <Textarea
                  id="site-description"
                  value={form.site_description ?? ''}
                  onChange={(e) =>
                    updateField('site_description', e.target.value)
                  }
                  placeholder={t('settings.siteDescriptionPlaceholder')}
                  className="min-h-[108px]"
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteDescriptionHint')}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('settings.siteChromeTitle')}</CardTitle>
              <CardDescription>{t('settings.siteChromeHint')}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-5">
              <div className="grid gap-2">
                <Label htmlFor="site-header-brand">
                  {t('settings.siteHeaderBrand')}
                </Label>
                <Input
                  id="site-header-brand"
                  value={form.site_header_brand ?? ''}
                  onChange={(e) =>
                    updateField('site_header_brand', e.target.value)
                  }
                  placeholder={siteName || 'Kite'}
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteHeaderBrandHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-header-github">
                  {t('settings.siteHeaderGitHubUrl')}
                </Label>
                <Input
                  id="site-header-github"
                  value={form.site_header_nav_github_url ?? ''}
                  onChange={(e) =>
                    updateField('site_header_nav_github_url', e.target.value)
                  }
                  placeholder="https://github.com/amigoer/kite"
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteHeaderGitHubUrlHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-footer-text">
                  {t('settings.siteFooterText')}
                </Label>
                <Textarea
                  id="site-footer-text"
                  value={form.site_footer_text ?? ''}
                  onChange={(e) =>
                    updateField('site_footer_text', e.target.value)
                  }
                  placeholder={t('settings.siteFooterTextPlaceholder')}
                  className="min-h-[88px]"
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteFooterTextHint')}
                </p>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="site-footer-copyright">
                  {t('settings.siteFooterCopyright')}
                </Label>
                <Input
                  id="site-footer-copyright"
                  value={form.site_footer_copyright ?? ''}
                  onChange={(e) =>
                    updateField('site_footer_copyright', e.target.value)
                  }
                  placeholder={`© ${new Date().getFullYear()} ${siteName || 'Kite'}`}
                />
                <p className="text-xs text-muted-foreground">
                  {t('settings.siteFooterCopyrightHint')}
                </p>
              </div>
            </CardContent>
            <CardFooter className="justify-end gap-2">
              <Button variant="outline" size="sm" onClick={resetForm}>
                {t('settings.reset')}
              </Button>
              <Button
                size="sm"
                onClick={() => mutation.mutate()}
                disabled={mutation.isPending}
              >
                {saved ? (
                  <>
                    <Check className="size-3.5" />
                    {t('settings.saved')}
                  </>
                ) : mutation.isPending ? (
                  t('settings.saving')
                ) : (
                  t('settings.saveSettings')
                )}
              </Button>
            </CardFooter>
          </Card>
        </div>
      )}

      {tab === 'upload' && (
        <Card>
          <CardHeader>
            <CardTitle>{t('settings.uploadDesc')}</CardTitle>
            <CardDescription>{t('settings.uploadHint')}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-5">
            <div className="grid gap-2 sm:max-w-xs">
              <Label htmlFor="upload-max-file-size">
                {t('settings.uploadMaxFileSize')}
              </Label>
              <div className="flex items-center gap-2">
                <Input
                  id="upload-max-file-size"
                  type="number"
                  min={1}
                  step={1}
                  inputMode="numeric"
                  value={uploadMaxFileSizeMB}
                  onChange={(e) =>
                    updateField('upload.max_file_size_mb', e.target.value)
                  }
                  placeholder={t('settings.uploadMaxFileSizePlaceholder')}
                />
                <span className="text-sm text-muted-foreground">MB</span>
              </div>
              <p className="text-xs text-muted-foreground">
                {t('settings.uploadMaxFileSizeHint')}
              </p>
              <p className="text-xs text-muted-foreground">
                {t('settings.uploadMaxFileSizePreview').replace(
                  '{size}',
                  formatSize(uploadMaxFileSizeBytes)
                )}
              </p>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="upload-path-pattern">
                {t('settings.uploadPathPattern')}
              </Label>
              <Input
                id="upload-path-pattern"
                value={uploadPattern}
                onChange={(e) =>
                  updateField('upload.path_pattern', e.target.value)
                }
                placeholder={t('settings.uploadPathPatternPlaceholder')}
                className="font-mono text-xs"
              />
              <p className="text-xs text-muted-foreground">
                {t('settings.uploadPathPatternHint')}
              </p>
            </div>

            <div className="space-y-2">
              <div className="text-sm font-medium">
                {t('settings.uploadPathPatternVariables')}
              </div>
              <div className="grid gap-2 sm:grid-cols-2">
                {uploadPatternVariables.map((item) => (
                  <div
                    key={item.token}
                    className="flex items-start gap-2 rounded-lg border bg-muted/30 px-3 py-2"
                  >
                    <Badge variant="secondary" className="font-mono">
                      {item.token}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {item.description}
                    </span>
                  </div>
                ))}
              </div>
            </div>

            <div className="space-y-2">
              <div className="text-sm font-medium">
                {t('settings.uploadPathPatternPreview')}
              </div>
              <code className="block overflow-x-auto rounded-lg border bg-muted/30 px-3 py-2 font-mono text-xs">
                {uploadPatternPreview}
              </code>
            </div>
          </CardContent>
          <CardFooter className="justify-end gap-2">
            <Button variant="outline" size="sm" onClick={resetForm}>
              {t('settings.reset')}
            </Button>
            <Button
              size="sm"
              onClick={() => mutation.mutate()}
              disabled={mutation.isPending}
            >
              {saved ? (
                <>
                  <Check className="size-3.5" />
                  {t('settings.saved')}
                </>
              ) : mutation.isPending ? (
                t('settings.saving')
              ) : (
                t('settings.saveSettings')
              )}
            </Button>
          </CardFooter>
        </Card>
      )}

      {/* ── Auth ─────────────────────────────────────── */}
      {tab === 'auth' && (
        <div className="space-y-5">
          <Card>
            <CardHeader>
              <CardTitle>{t('settings.authDesc')}</CardTitle>
            </CardHeader>
            <CardContent className="divide-y">
              <Preference
                label={t('settings.twoFactor')}
                hint={t('settings.twoFactorHint')}
              >
                <Switch
                  checked={boolOf('two_factor_required')}
                  onCheckedChange={() => toggleField('two_factor_required')}
                />
              </Preference>
              <Preference label={t('settings.passwordMinLength')}>
                <Input
                  value={form.password_min_length ?? ''}
                  onChange={(e) =>
                    updateField('password_min_length', e.target.value)
                  }
                  placeholder="10"
                  className="w-20"
                />
              </Preference>
              <Preference
                label={t('settings.sessionTimeout')}
                hint={t('settings.sessionTimeoutHint')}
              >
                <Input
                  value={form.session_timeout ?? ''}
                  onChange={(e) =>
                    updateField('session_timeout', e.target.value)
                  }
                  placeholder="7d"
                  className="w-24"
                />
              </Preference>
              <Preference
                label={t('settings.allowGuestUpload')}
                hint={t('settings.allowGuestUploadHint')}
              >
                <Switch
                  checked={boolOf('allow_guest_upload')}
                  onCheckedChange={() => toggleField('allow_guest_upload')}
                />
              </Preference>
              <Preference
                label={t('settings.allowPublicGallery')}
                hint={t('settings.allowPublicGalleryHint')}
              >
                <Switch
                  checked={boolOf('allow_public_gallery')}
                  onCheckedChange={() => toggleField('allow_public_gallery')}
                />
              </Preference>
            </CardContent>
            <CardFooter className="justify-end gap-2">
              <Button variant="outline" size="sm" onClick={resetForm}>
                {t('settings.reset')}
              </Button>
              <Button
                size="sm"
                onClick={() => mutation.mutate()}
                disabled={mutation.isPending}
              >
                {saved ? (
                  <>
                    <Check className="size-3.5" />
                    {t('settings.saved')}
                  </>
                ) : mutation.isPending ? (
                  t('settings.saving')
                ) : (
                  t('settings.saveSettings')
                )}
              </Button>
            </CardFooter>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('settings.oauthProvidersTitle')}</CardTitle>
              <CardDescription>
                {t('settings.oauthProvidersHint')}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {(providerList ?? []).map((provider) => {
                  const draft = getProviderDraft(provider)
                  const isPending =
                    saveProviderMutation.isPending &&
                    saveProviderMutation.variables?.provider === provider.key
                  const missing =
                    providerTouched[provider.key] &&
                    providerMissingFields(provider, draft)
                  return (
                    <div
                      key={provider.key}
                      className="rounded-xl border bg-background/70 p-4"
                    >
                      <div className="flex items-center justify-between gap-3">
                        <div className="flex min-w-0 items-center gap-3">
                          <SocialProviderLogo
                            provider={provider.icon_key}
                            size={36}
                            rounded="rounded-lg"
                          />
                          <div className="min-w-0">
                            <div className="text-sm font-medium">
                              {provider.label}
                            </div>
                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                              <span>{provider.protocol}</span>
                              <span aria-hidden>·</span>
                              <span>{getProviderStatus(provider, draft)}</span>
                            </div>
                          </div>
                        </div>
                        <Switch
                          checked={draft.enabled}
                          onCheckedChange={(checked) =>
                            updateProviderDraft(provider.key, {
                              enabled: checked,
                            })
                          }
                        />
                      </div>

                      {!provider.site_url_valid && (
                        <div className="mt-4 rounded-lg border border-amber-500/30 bg-amber-500/10 p-3 text-xs text-amber-700 dark:text-amber-300">
                          {t('settings.oauthSiteUrlInvalid')}
                        </div>
                      )}

                      <div className="mt-4 grid gap-4 sm:grid-cols-2">
                        <div className="grid gap-2">
                          <Label>{clientIdLabel(provider.key)}</Label>
                          <Input
                            value={draft.client_id}
                            onChange={(e) =>
                              updateProviderDraft(provider.key, {
                                client_id: e.target.value,
                              })
                            }
                            className={cn(
                              missing &&
                                !draft.client_id.trim() &&
                                'border-red-500'
                            )}
                            placeholder={
                              provider.key === 'wechat'
                                ? 'wx123...'
                                : 'client-id'
                            }
                          />
                        </div>

                        <div className="grid gap-2">
                          <Label>{clientSecretLabel(provider.key)}</Label>
                          <Input
                            type="password"
                            value={draft.client_secret}
                            onChange={(e) =>
                              updateProviderDraft(provider.key, {
                                client_secret: e.target.value,
                              })
                            }
                            className={cn(
                              missing &&
                                !draft.client_secret.trim() &&
                                !provider.has_secret &&
                                'border-red-500'
                            )}
                            placeholder={
                              provider.has_secret
                                ? t('settings.oauthSecretConfigured')
                                : provider.key === 'wechat'
                                  ? 'app-secret'
                                  : 'client-secret'
                            }
                          />
                        </div>

                        <div className="grid gap-2 sm:col-span-2">
                          <Label>{t('settings.oauthCallbackUrl')}</Label>
                          <div className="flex items-center gap-2">
                            <Input
                              value={provider.callback_url}
                              readOnly
                              className="font-mono text-xs"
                            />
                            <Button
                              type="button"
                              variant="outline"
                              size="icon"
                              onClick={() => {
                                navigator.clipboard.writeText(
                                  provider.callback_url
                                )
                                toast.success(t('toast.copied'))
                              }}
                            >
                              <Copy className="size-4" />
                            </Button>
                          </div>
                        </div>
                      </div>

                      <div className="mt-4 flex items-center justify-between gap-3 text-[11px] text-muted-foreground">
                        <div className="min-w-0 truncate">
                          <span className="font-medium">
                            {t('settings.oauthScopes')}
                          </span>
                          <span className="ml-2 font-mono">
                            {provider.scopes.join(' ')}
                          </span>
                        </div>
                        <Button
                          size="sm"
                          onClick={() => handleSaveProvider(provider)}
                          disabled={isPending}
                        >
                          {isPending && (
                            <Loader2 className="size-4 animate-spin" />
                          )}
                          {t('settings.saveProvider')}
                        </Button>
                      </div>
                    </div>
                  )
                })}

                {providerList == null &&
                  Array.from({ length: 3 }).map((_, index) => (
                    <Skeleton key={index} className="h-[180px] rounded-xl" />
                  ))}
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* ── Default storage ──────────────────────────── */}
      {tab === 'storage' && (
        <Card>
          <CardHeader>
            <CardTitle>{t('settings.storageTabDesc')}</CardTitle>
            <CardDescription>{t('settings.storageTabHint')}</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {(storageList ?? []).map((d) => {
              const vendor = resolveLogoVendor(d.provider, d.driver)
              const checked = d.is_default
              return (
                <label
                  key={d.id}
                  className={cn(
                    'flex cursor-pointer items-center gap-3 rounded-lg border p-3 transition-colors',
                    checked
                      ? 'border-foreground/30 bg-muted/30'
                      : 'hover:bg-muted/30'
                  )}
                >
                  <input
                    type="radio"
                    name="default-driver"
                    checked={checked}
                    disabled={!d.is_active || setDefaultMutation.isPending}
                    onChange={() => {
                      if (!checked && d.is_active)
                        setDefaultMutation.mutate(d.id)
                    }}
                    className="size-4 accent-foreground"
                  />
                  <StorageLogo vendor={vendor} size={28} rounded="rounded-md" />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium">
                        {d.name}
                      </span>
                      <Badge
                        variant="outline"
                        className="h-4 px-1.5 font-mono text-[10px] uppercase tracking-wide text-muted-foreground"
                      >
                        {d.driver}
                      </Badge>
                      {!d.is_active && (
                        <Badge
                          variant="outline"
                          className="h-4 px-1.5 text-[10px] uppercase tracking-wide text-muted-foreground"
                        >
                          {t('storage.idleBadge')}
                        </Badge>
                      )}
                    </div>
                    <div className="font-mono text-[10px] text-muted-foreground">
                      P{d.priority}
                    </div>
                  </div>
                  <span className="shrink-0 text-[11px] tabular-nums text-muted-foreground">
                    {formatSize(d.used_bytes)}
                  </span>
                </label>
              )
            })}
            {storageList != null && storageList.length === 0 && (
              <div className="py-8 text-center text-sm text-muted-foreground">
                {t('storage.noStorage')}
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* ── Email ────────────────────────────────────── */}
      {tab === 'email' && (
        <Card>
          <CardHeader>
            <CardTitle>{t('settings.emailDesc')}</CardTitle>
          </CardHeader>
          <CardContent className="divide-y">
            <Preference label={t('settings.smtpServer')}>
              <Input
                value={form.smtp_host ?? ''}
                onChange={(e) => updateField('smtp_host', e.target.value)}
                placeholder="smtp.kite.dev"
                className="w-64"
              />
            </Preference>
            <Preference label={t('settings.smtpPort')}>
              <Input
                value={form.smtp_port ?? ''}
                onChange={(e) => updateField('smtp_port', e.target.value)}
                placeholder="587"
                className="w-24"
              />
            </Preference>
            <Preference label={t('settings.smtpTls')}>
              <Switch
                checked={boolOf('smtp_tls')}
                onCheckedChange={() => toggleField('smtp_tls')}
              />
            </Preference>
            <Preference label={t('settings.smtpFrom')}>
              <Input
                value={form.smtp_from ?? ''}
                onChange={(e) => updateField('smtp_from', e.target.value)}
                placeholder="no-reply@kite.dev"
                className="w-64"
              />
            </Preference>
          </CardContent>
          <CardFooter className="justify-between gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => toast.info(t('settings.testMailSent'))}
            >
              <Mail className="size-3.5" />
              {t('settings.sendTestMail')}
            </Button>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={resetForm}>
                {t('settings.reset')}
              </Button>
              <Button
                size="sm"
                onClick={() => mutation.mutate()}
                disabled={mutation.isPending}
              >
                {saved ? (
                  <>
                    <Check className="size-3.5" />
                    {t('settings.saved')}
                  </>
                ) : mutation.isPending ? (
                  t('settings.saving')
                ) : (
                  t('settings.saveSettings')
                )}
              </Button>
            </div>
          </CardFooter>
        </Card>
      )}
    </div>
  )
}
