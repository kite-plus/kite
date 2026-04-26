import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate } from 'react-router-dom'
import { Loader2 } from 'lucide-react'
import { useAuth } from '@/hooks/use-auth'
import { authApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { KiteLogo } from '@/components/kite-logo'
import { useI18n } from '@/i18n'
import { toast } from 'sonner'

export default function RegisterPage() {
  const { register } = useAuth()
  const { t } = useI18n()
  const navigate = useNavigate()
  const [form, setForm] = useState({
    username: '',
    email: '',
    password: '',
    confirmPassword: '',
  })
  const [loading, setLoading] = useState(false)
  const { data: authOptions } = useQuery<{ allow_registration: boolean }>({
    queryKey: ['auth', 'options'],
    queryFn: () => authApi.options().then((r) => r.data.data),
    retry: 0,
  })
  const allowRegistration = authOptions?.allow_registration !== false

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!allowRegistration) {
      toast.error(t('auth.registrationClosed'))
      return
    }

    if (form.password !== form.confirmPassword) {
      toast.error(t('auth.passwordMismatch'))
      return
    }

    setLoading(true)
    try {
      await register(form.username, form.email, form.password)
      toast.success(t('auth.completeSuccess'))
      navigate('/login')
    } catch (err: unknown) {
      const status = (err as { response?: { status?: number } })?.response
        ?.status
      // 403 means the server says registration is off — give the
      // operator-friendly message rather than the raw envelope. For
      // any other error, prefer the backend's localized `message`
      // (catalogue-translated by the locale middleware) and only
      // fall back to the generic "try later" copy when nothing
      // useful comes back.
      const msg =
        (status === 403 ? t('auth.registrationClosedDesc') : undefined) ??
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ??
        t('auth.registrationFailed')
      toast.error(msg)
    } finally {
      setLoading(false)
    }
  }

  const update = (field: string) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((prev) => ({ ...prev, [field]: e.target.value }))

  return (
    <>
      <div className="mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:w-[480px] sm:p-8">
        <div className="mb-4 flex items-center justify-center">
          <KiteLogo className="me-2 size-6" />
          <h1 className="text-xl font-medium">Kite</h1>
        </div>
      </div>

      <div className="mx-auto flex w-full max-w-sm flex-col justify-center">
        <div className="flex flex-col space-y-2 text-start">
          <h2 className="text-2xl font-semibold tracking-tight">
            {t('auth.createAccount')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {t('auth.createAccountDesc')}
          </p>
        </div>

        {allowRegistration ? (
          <form onSubmit={handleSubmit} className="grid gap-3 pt-2">
            <div className="grid gap-2">
              <Label htmlFor="username">{t('auth.username')}</Label>
              <Input
                id="username"
                autoCapitalize="none"
                autoCorrect="off"
                value={form.username}
                onChange={update('username')}
                placeholder={t('auth.chooseUsername')}
                required
                minLength={3}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="email">{t('auth.email')}</Label>
              <Input
                id="email"
                type="email"
                autoCapitalize="none"
                autoComplete="email"
                autoCorrect="off"
                value={form.email}
                onChange={update('email')}
                placeholder={t('auth.emailPlaceholder')}
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="password">{t('auth.password')}</Label>
              <Input
                id="password"
                type="password"
                value={form.password}
                onChange={update('password')}
                placeholder={t('auth.passwordPlaceholder')}
                required
                minLength={6}
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="confirmPassword">
                {t('auth.confirmPassword')}
              </Label>
              <Input
                id="confirmPassword"
                type="password"
                value={form.confirmPassword}
                onChange={update('confirmPassword')}
                placeholder={t('auth.confirmPasswordPlaceholder')}
                required
              />
            </div>

            <Button type="submit" className="mt-2" disabled={loading}>
              {loading && <Loader2 className="size-4 animate-spin" />}
              {loading ? t('auth.creatingAccount') : t('auth.signUp')}
            </Button>
          </form>
        ) : (
          <div className="mt-4 rounded-lg border bg-muted/25 p-4 text-sm">
            <div className="font-medium">{t('auth.registrationClosed')}</div>
            <p className="mt-1 text-muted-foreground">
              {t('auth.registrationClosedDesc')}
            </p>
            <Button asChild variant="outline" className="mt-4 w-full">
              <Link to="/login">{t('auth.backToLogin')}</Link>
            </Button>
          </div>
        )}

        <p className="mt-6 text-center text-sm text-muted-foreground">
          {t('auth.hasAccount')}{' '}
          <Link
            to="/login"
            className="font-medium text-foreground underline-offset-4 hover:underline"
          >
            {t('auth.signInNow')}
          </Link>
        </p>

        <p className="mt-4 text-center text-xs text-muted-foreground">
          {t('auth.termsRegisterPrefix')}{' '}
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
      </div>
    </>
  )
}
