import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'

import { authApi } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { useI18n } from '@/i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'

interface AuthOptions {
  allow_registration: boolean
}

export default function CompleteSocialPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { refreshProfile } = useAuth()
  const { t } = useI18n()
  const ticket = searchParams.get('ticket') ?? ''
  const checkedRef = useRef(false)
  const [form, setForm] = useState({
    username: '',
    email: '',
  })

  const { data: authOptions, isLoading } = useQuery<AuthOptions>({
    queryKey: ['auth', 'options'],
    queryFn: () => authApi.options().then((r) => r.data.data),
    retry: 0,
  })

  useEffect(() => {
    if (checkedRef.current) return
    checkedRef.current = true
    if (!ticket) {
      toast.error(t('auth.errTicketMissing'))
      navigate('/login', { replace: true })
    }
  }, [navigate, ticket, t])

  const onboardMutation = useMutation({
    mutationFn: () =>
      authApi.onboardOAuth({
        ticket,
        username: form.username.trim(),
        email: form.email.trim(),
      }),
    onSuccess: async (res) => {
      // Server set HttpOnly cookies on the onboard response; we just
      // need to load the new user's profile into the auth store.
      await refreshProfile()
      const data = res.data.data
      toast.success(t('auth.completeSuccess'))
      navigate(data.return_to || '/user/dashboard', { replace: true })
    },
    onError: (err: unknown) => {
      const backendMsg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t('auth.completeFailed')
      toast.error(backendMsg)
    },
  })

  if (isLoading) {
    return (
      <div className="flex min-h-[420px] items-center justify-center">
        <Loader2 className="size-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (authOptions?.allow_registration === false) {
    return (
      <div className="space-y-4 text-center">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">
            {t('auth.completeRegistrationClosedTitle')}
          </h2>
          <p className="mt-2 text-sm text-muted-foreground">
            {t('auth.completeRegistrationClosedDesc')}
          </p>
        </div>
        <Button asChild>
          <Link to="/login">{t('auth.backToLogin')}</Link>
        </Button>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="space-y-2 text-center">
        <h2 className="text-2xl font-semibold tracking-tight">
          {t('auth.completeTitle')}
        </h2>
        <p className="text-sm text-muted-foreground">
          {t('auth.completeSubtitle')}
        </p>
      </div>

      <form
        className="grid gap-4"
        onSubmit={(e) => {
          e.preventDefault()
          if (!form.username.trim() || !form.email.trim()) {
            toast.error(t('auth.completeMissingFields'))
            return
          }
          onboardMutation.mutate()
        }}
      >
        <div className="grid gap-2">
          <Label htmlFor="username">{t('auth.username')}</Label>
          <Input
            id="username"
            value={form.username}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, username: e.target.value }))
            }
            minLength={3}
            maxLength={32}
            placeholder="username"
            required
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="email">{t('auth.email')}</Label>
          <Input
            id="email"
            type="email"
            value={form.email}
            onChange={(e) =>
              setForm((prev) => ({ ...prev, email: e.target.value }))
            }
            placeholder={t('auth.emailPlaceholder')}
            required
          />
        </div>

        <Button type="submit" disabled={onboardMutation.isPending}>
          {onboardMutation.isPending && (
            <Loader2 className="size-4 animate-spin" />
          )}
          {t('auth.completeFinish')}
        </Button>
      </form>
    </div>
  )
}
