import { useEffect, useRef } from 'react'
import { Loader2 } from 'lucide-react'
import { useNavigate, useSearchParams } from 'react-router-dom'

import { authApi } from '@/lib/api'
import { useAuth } from '@/hooks/use-auth'
import { useI18n } from '@/i18n'
import { toast } from 'sonner'

export default function LoginCallbackPage() {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { refreshProfile } = useAuth()
  const { t } = useI18n()
  const startedRef = useRef(false)

  useEffect(() => {
    if (startedRef.current) return
    startedRef.current = true

    const ticket = searchParams.get('ticket')
    if (!ticket) {
      toast.error(t('auth.errTicketMissing'))
      navigate('/login', { replace: true })
      return
    }

    authApi
      .exchangeOAuth(ticket)
      .then(async (res) => {
        // Server set HttpOnly access/refresh cookies on the exchange
        // response; we just need to hydrate the in-memory profile.
        await refreshProfile()
        const data = res.data.data
        navigate(data.return_to || '/user/dashboard', { replace: true })
      })
      .catch((err: unknown) => {
        // Prefer the backend-translated message when available — the
        // locale middleware already picked the right language based on
        // the kite_locale cookie. Only fall back to the local copy
        // when the response was empty (network blip, 5xx with no body).
        const backendMsg =
          (err as { response?: { data?: { message?: string } } })?.response
            ?.data?.message ?? t('auth.errOAuthFailed')
        toast.error(backendMsg)
        navigate(`/login?oauth_error=${encodeURIComponent(backendMsg)}`, {
          replace: true,
        })
      })
  }, [refreshProfile, navigate, searchParams, t])

  return (
    <div className="flex min-h-[420px] flex-col items-center justify-center gap-3 text-center">
      <Loader2 className="size-8 animate-spin text-muted-foreground" />
      <div>
        <p className="text-sm font-medium">{t('auth.callbackTitle')}</p>
        <p className="text-xs text-muted-foreground">
          {t('auth.callbackSubtitle')}
        </p>
      </div>
    </div>
  )
}
