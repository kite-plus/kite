import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Eye,
  EyeOff,
  Loader2,
  LogIn,
  ShieldCheck,
} from 'lucide-react'

import { useAuth } from '@/hooks/use-auth'
import { authApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { KiteLogo } from '@/components/kite-logo'
import { SocialProviderLogo } from '@/components/social-provider-logo'
import { useI18n } from '@/i18n'
import { toast } from 'sonner'

interface AuthOptions {
  allow_registration: boolean
  social_providers: Array<{
    key: string
    label: string
    icon_key: string
  }>
}

export default function LoginPage() {
  const { login, verifyTotp } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [loading, setLoading] = useState(false)
  const [showPassword, setShowPassword] = useState(false)

  // 2FA challenge state. When set, we render the TOTP step instead of
  // the password form. The challenge token expires server-side after
  // 5 minutes; expiresAt lets us show a countdown if we want later.
  const [challenge, setChallenge] = useState<{
    token: string
    expiresAt: string
  } | null>(null)
  const [totpCode, setTotpCode] = useState('')
  const [totpLoading, setTotpLoading] = useState(false)
  const totpInputRef = useRef<HTMLInputElement | null>(null)

  const { data: authOptions } = useQuery<AuthOptions>({
    queryKey: ['auth', 'options'],
    queryFn: () => authApi.options().then((r) => r.data.data),
    retry: 0,
  })

  const allowRegistration = authOptions?.allow_registration !== false
  const socialProviders = authOptions?.social_providers ?? []
  const redirectTo = useMemo(
    () =>
      (location.state as { from?: string } | null)?.from ?? '/user/dashboard',
    [location.state]
  )

  useEffect(() => {
    const params = new URLSearchParams(location.search)
    const oauthError = params.get('oauth_error')
    if (!oauthError) return
    toast.error(oauthError)
    navigate(location.pathname, { replace: true, state: location.state })
  }, [location.pathname, location.search, location.state, navigate])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    const minDelay = new Promise((r) => setTimeout(r, 600))
    try {
      const [outcome] = await Promise.all([login(username, password), minDelay])
      if (outcome.ok) {
        toast.success(t('auth.signInSuccess'))
        navigate(redirectTo, { replace: true })
        return
      }
      // 2FA required — swap to the TOTP step. Password is cleared
      // so it can't be scraped from memory via devtools while we
      // wait for the code; username is kept visible for context.
      setChallenge({
        token: outcome.challengeToken,
        expiresAt: outcome.expiresAt,
      })
      setPassword('')
      setTotpCode('')
      // autofocus the TOTP input on the next paint
      setTimeout(() => totpInputRef.current?.focus(), 0)
    } catch (err) {
      await minDelay
      const status = (err as { response?: { status?: number } })?.response
        ?.status
      const backendMsg = (err as { response?: { data?: { message?: string } } })
        ?.response?.data?.message
      // Status-aware messages keep the UX friendly when the backend
      // returns a generic envelope. The 401 / 403 / 429 / 5xx branches
      // win over backendMsg because the catalogue copy is purpose-built
      // for those flows; backendMsg is the fallback for everything else
      // (validation errors, business-logic 400s, etc.).
      let msg = t('auth.errLoginFailed')
      if (status === 401) {
        msg = t('auth.invalidCredentials')
      } else if (status === 403) {
        msg = t('auth.errAccountDisabled')
      } else if (status === 429) {
        msg = t('auth.errRateLimited')
      } else if (status && status >= 500) {
        msg = t('auth.errServerUnavailable')
      } else if (backendMsg) {
        msg = backendMsg
      }
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const handleVerifyTotp = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!challenge) return
    setTotpLoading(true)
    try {
      await verifyTotp(challenge.token, totpCode)
      toast.success(t('auth.signInSuccess'))
      navigate(redirectTo, { replace: true })
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response
        ?.status
      const backendMsg = (err as { response?: { data?: { message?: string } } })
        ?.response?.data?.message
      let msg = t('auth.errTotpInvalid')
      if (status === 401) {
        // challenge expired or invalid — return to password step
        msg = t('auth.errTotpExpired')
        setChallenge(null)
      } else if (backendMsg) {
        msg = backendMsg
      }
      toast.error(msg)
      setTotpCode('')
      totpInputRef.current?.focus()
    } finally {
      setTotpLoading(false)
    }
  }

  const cancelTotp = () => {
    setChallenge(null)
    setTotpCode('')
    setPassword('')
  }

  const startSocialLogin = (provider: string) => {
    const params = new URLSearchParams({ return_to: redirectTo })
    window.location.href = `/api/v1/auth/oauth/${provider}/start?${params.toString()}`
  }

  return (
    <>
      <div className="mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:w-[480px] sm:p-8">
        <div className="mb-4 flex items-center justify-center">
          <KiteLogo className="me-2 size-6" />
          <h1 className="text-xl font-medium">Kite</h1>
        </div>
      </div>

      <div className="mx-auto flex w-full max-w-sm flex-col justify-center">
        {challenge ? (
          <div className="flex flex-col space-y-1.5 text-start">
            <div className="mb-2 inline-flex size-10 items-center justify-center rounded-full border bg-muted/50 text-foreground">
              <ShieldCheck className="size-5" />
            </div>
            <h2 className="text-2xl font-semibold tracking-tight">
              {t('auth.totpTitle')}
            </h2>
            {/* Split the locale entry around the {username} placeholder
                so the username keeps its bold styling. The catalogue's
                Chinese form puts the placeholder in the middle of the
                sentence; the English form does too — both flank it with
                fixed pre/post copy. */}
            <p className="text-sm text-muted-foreground">
              {t('auth.totpHint')
                .split('{username}')
                .flatMap((piece, idx, arr) =>
                  idx < arr.length - 1
                    ? [
                        piece,
                        <span key={idx} className="font-medium text-foreground">
                          {username}
                        </span>,
                      ]
                    : [piece]
                )}
            </p>
          </div>
        ) : (
          <div className="flex flex-col space-y-1.5 text-start">
            <h2 className="text-2xl font-semibold tracking-tight">
              {t('auth.welcomeBack')}
            </h2>
            <p className="text-sm text-muted-foreground">
              {t('auth.signInDesc')}
            </p>
          </div>
        )}

        {challenge && (
          <form onSubmit={handleVerifyTotp} className="grid gap-3 pt-4">
            <div className="grid gap-2">
              <Label htmlFor="totp">{t('auth.totpLabel')}</Label>
              <Input
                id="totp"
                ref={totpInputRef}
                inputMode="numeric"
                autoComplete="one-time-code"
                autoFocus
                maxLength={6}
                value={totpCode}
                onChange={(e) =>
                  setTotpCode(e.target.value.replace(/\D/g, '').slice(0, 6))
                }
                placeholder="123456"
                className="text-center text-lg tracking-[0.5em]"
                required
              />
            </div>
            <Button
              type="submit"
              className="mt-2"
              disabled={totpLoading || totpCode.length !== 6}
            >
              {totpLoading ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <ShieldCheck className="size-4" />
              )}
              {totpLoading ? t('auth.verifying') : t('auth.verifyAndLogin')}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={cancelTotp}
              disabled={totpLoading}
            >
              <ArrowLeft className="size-4" />
              {t('auth.backToLogin')}
            </Button>
          </form>
        )}

        {!challenge && (
          <>
            <form onSubmit={handleSubmit} className="grid gap-3 pt-2">
              <div className="grid gap-2">
                <Label htmlFor="username">{t('auth.usernameOrEmail')}</Label>
                <Input
                  id="username"
                  autoCapitalize="none"
                  autoComplete="email"
                  autoCorrect="off"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  placeholder="name@example.com"
                  required
                />
              </div>

              <div className="grid gap-2">
                <div className="flex items-center justify-between">
                  <Label htmlFor="password">{t('auth.password')}</Label>
                  <Link
                    to="/forgot-password"
                    className="text-xs font-medium text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
                  >
                    {t('auth.forgotPassword')}
                  </Link>
                </div>
                <div className="relative">
                  <Input
                    id="password"
                    type={showPassword ? 'text' : 'password'}
                    autoComplete="current-password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="••••••••"
                    className="pr-9"
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setShowPassword(!showPassword)}
                    tabIndex={-1}
                    className="absolute right-1 top-1/2 flex size-7 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                    aria-label={
                      showPassword
                        ? t('auth.hidePassword')
                        : t('auth.showPassword')
                    }
                  >
                    {showPassword ? (
                      <EyeOff className="size-4" />
                    ) : (
                      <Eye className="size-4" />
                    )}
                  </button>
                </div>
              </div>

              <Button type="submit" className="mt-2" disabled={loading}>
                {loading ? (
                  <Loader2 className="size-4 animate-spin" />
                ) : (
                  <LogIn className="size-4" />
                )}
                {loading ? t('auth.signingIn') : t('auth.login')}
              </Button>

              {socialProviders.length > 0 && (
                <>
                  <div className="relative my-2">
                    <div className="absolute inset-0 flex items-center">
                      <span className="w-full border-t" />
                    </div>
                    <div className="relative flex justify-center text-xs uppercase">
                      <span className="bg-background px-2 text-muted-foreground">
                        {t('auth.continueWith')}
                      </span>
                    </div>
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    {socialProviders.map((provider) => (
                      <Button
                        key={provider.key}
                        variant="outline"
                        type="button"
                        disabled={loading}
                        onClick={() => startSocialLogin(provider.key)}
                      >
                        <SocialProviderLogo
                          provider={provider.icon_key}
                          size={16}
                          appearance="plain"
                        />
                        {provider.label}
                      </Button>
                    ))}
                  </div>
                </>
              )}
            </form>

            <p className="mt-6 text-center text-sm text-muted-foreground">
              {allowRegistration ? (
                <>
                  {t('auth.noAccount')}{' '}
                  <Link
                    to="/register"
                    className="font-medium text-foreground underline-offset-4 hover:underline"
                  >
                    {t('auth.signUpNow')}
                  </Link>
                </>
              ) : (
                t('auth.registrationClosed')
              )}
            </p>

            <p className="mt-4 text-center text-xs text-muted-foreground">
              {t('auth.termsLoginPrefix')}{' '}
              <a
                href="#"
                className="underline underline-offset-4 hover:text-foreground"
              >
                {t('auth.termsTOS')}
              </a>{' '}
              {t('auth.termsAnd')}{' '}
              <a
                href="#"
                className="underline underline-offset-4 hover:text-foreground"
              >
                {t('auth.termsPrivacy')}
              </a>
              {t('auth.termsSuffix')}
            </p>
          </>
        )}
      </div>
    </>
  )
}
