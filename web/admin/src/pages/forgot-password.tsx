import { useEffect, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import {
  ArrowLeft,
  Eye,
  EyeOff,
  KeyRound,
  Loader2,
  MailCheck,
  ShieldCheck,
} from 'lucide-react'
import { toast } from 'sonner'

import { authApi } from '@/lib/api'
import { useI18n } from '@/i18n'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { KiteLogo } from '@/components/kite-logo'

// Client-side resend cooldown in seconds. The backend uses the same
// value (60s) as its server-side throttle; duplicating it here is
// strictly a UX signal — every actual send gets re-validated server
// side regardless.
const RESEND_SECONDS = 60

export default function ForgotPasswordPage() {
  const { t } = useI18n()
  const navigate = useNavigate()

  const [step, setStep] = useState<'request' | 'verify'>('request')
  const [identifier, setIdentifier] = useState('')
  const [code, setCode] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [showPassword, setShowPassword] = useState(false)
  const [requesting, setRequesting] = useState(false)
  const [confirming, setConfirming] = useState(false)
  const [resendTimer, setResendTimer] = useState(0)
  const codeInputRef = useRef<HTMLInputElement | null>(null)

  // Countdown ticker. We intentionally run it in its own effect so
  // the timer keeps ticking even if the step component re-renders
  // for unrelated reasons (typing in the code field, etc).
  useEffect(() => {
    if (resendTimer <= 0) return
    const id = setInterval(() => {
      setResendTimer((prev) => (prev > 0 ? prev - 1 : 0))
    }, 1000)
    return () => clearInterval(id)
  }, [resendTimer])

  const startResendCooldown = () => setResendTimer(RESEND_SECONDS)

  const handleRequest = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!identifier.trim()) return
    setRequesting(true)
    try {
      await authApi.requestPasswordReset(identifier.trim())
      // Backend returns 200 whether or not the account exists, so we
      // show a generic success toast. Real UX confirmation is the
      // email the user will (or won't) see land in their inbox.
      toast.success(t('forgotPassword.sentIfExists'))
      setStep('verify')
      startResendCooldown()
      setTimeout(() => codeInputRef.current?.focus(), 0)
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response
        ?.status
      const backendMsg = (err as { response?: { data?: { message?: string } } })
        ?.response?.data?.message
      let msg = t('forgotPassword.sendFailed')
      if (status === 424) {
        msg = backendMsg ?? t('forgotPassword.smtpUnavailable')
      } else if (status === 429) {
        msg = t('auth.tooManyRequests')
      } else if (backendMsg) {
        msg = backendMsg
      }
      toast.error(msg)
    } finally {
      setRequesting(false)
    }
  }

  const handleResend = async () => {
    if (resendTimer > 0 || requesting) return
    setRequesting(true)
    try {
      await authApi.requestPasswordReset(identifier.trim())
      toast.success(t('forgotPassword.resent'))
      startResendCooldown()
    } catch {
      // Silent: if it fails we keep the user on the verify step
      // since the previous code (if real) may still be valid.
      toast.error(t('forgotPassword.sendFailed'))
    } finally {
      setRequesting(false)
    }
  }

  const handleConfirm = async (e: React.FormEvent) => {
    e.preventDefault()
    if (code.length < 4 || !newPassword) return
    if (newPassword !== confirmPassword) {
      toast.error(t('auth.passwordMismatch'))
      return
    }
    setConfirming(true)
    try {
      await authApi.confirmPasswordReset(identifier.trim(), code, newPassword)
      toast.success(t('forgotPassword.resetSuccess'))
      navigate('/login', { replace: true })
    } catch (err) {
      const status = (err as { response?: { status?: number } })?.response
        ?.status
      const backendMsg = (err as { response?: { data?: { message?: string } } })
        ?.response?.data?.message
      let msg = t('forgotPassword.resetFailed')
      if (status === 404) {
        msg = t('forgotPassword.codeNotFound')
      } else if (status === 410) {
        msg = t('forgotPassword.codeExpired')
      } else if (status === 400 && backendMsg) {
        msg = backendMsg
      }
      toast.error(msg)
      setCode('')
      codeInputRef.current?.focus()
    } finally {
      setConfirming(false)
    }
  }

  const backToRequest = () => {
    setStep('request')
    setCode('')
    setNewPassword('')
    setConfirmPassword('')
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
        <div className="flex flex-col space-y-1.5 text-start">
          <div className="mb-2 inline-flex size-10 items-center justify-center rounded-full border bg-muted/50 text-foreground">
            {step === 'request' ? (
              <KeyRound className="size-5" />
            ) : (
              <MailCheck className="size-5" />
            )}
          </div>
          <h2 className="text-2xl font-semibold tracking-tight">
            {step === 'request'
              ? t('forgotPassword.title')
              : t('forgotPassword.verifyTitle')}
          </h2>
          <p className="text-sm text-muted-foreground">
            {step === 'request'
              ? t('forgotPassword.subtitle')
              : t('forgotPassword.verifySubtitle')}
          </p>
        </div>

        {step === 'request' && (
          <form onSubmit={handleRequest} className="grid gap-3 pt-4">
            <div className="grid gap-2">
              <Label htmlFor="identifier">
                {t('forgotPassword.identifier')}
              </Label>
              <Input
                id="identifier"
                autoFocus
                autoCapitalize="none"
                autoComplete="username"
                value={identifier}
                onChange={(e) => setIdentifier(e.target.value)}
                placeholder={t('forgotPassword.identifierPlaceholder')}
                required
              />
            </div>
            <Button
              type="submit"
              className="mt-2"
              disabled={requesting || !identifier.trim()}
            >
              {requesting ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <MailCheck className="size-4" />
              )}
              {requesting
                ? t('forgotPassword.sending')
                : t('forgotPassword.sendCode')}
            </Button>
            <Button asChild type="button" variant="ghost">
              <Link to="/login">
                <ArrowLeft className="size-4" />
                {t('forgotPassword.backToLogin')}
              </Link>
            </Button>
          </form>
        )}

        {step === 'verify' && (
          <form onSubmit={handleConfirm} className="grid gap-3 pt-4">
            <div className="grid gap-2">
              <Label htmlFor="code">{t('forgotPassword.code')}</Label>
              <Input
                id="code"
                ref={codeInputRef}
                inputMode="numeric"
                autoComplete="one-time-code"
                maxLength={6}
                value={code}
                onChange={(e) =>
                  setCode(e.target.value.replace(/\D/g, '').slice(0, 6))
                }
                placeholder="123456"
                className="text-center text-lg tracking-[0.5em]"
                required
              />
              <div className="flex items-center justify-between text-[11px] text-muted-foreground">
                <span>{t('forgotPassword.codeHint')}</span>
                <button
                  type="button"
                  className="font-medium text-foreground underline-offset-4 hover:underline disabled:cursor-not-allowed disabled:text-muted-foreground disabled:no-underline"
                  onClick={handleResend}
                  disabled={resendTimer > 0 || requesting}
                >
                  {resendTimer > 0
                    ? t('forgotPassword.resendIn').replace(
                        '{s}',
                        String(resendTimer)
                      )
                    : t('forgotPassword.resend')}
                </button>
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="new-password">
                {t('forgotPassword.newPassword')}
              </Label>
              <div className="relative">
                <Input
                  id="new-password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete="new-password"
                  minLength={6}
                  maxLength={64}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  placeholder={t('auth.passwordHint')}
                  className="pr-9"
                  required
                />
                <button
                  type="button"
                  onClick={() => setShowPassword(!showPassword)}
                  tabIndex={-1}
                  className="absolute right-1 top-1/2 flex size-7 -translate-y-1/2 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                >
                  {showPassword ? (
                    <EyeOff className="size-4" />
                  ) : (
                    <Eye className="size-4" />
                  )}
                </button>
              </div>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="confirm-password">
                {t('auth.confirmPassword')}
              </Label>
              <Input
                id="confirm-password"
                type={showPassword ? 'text' : 'password'}
                autoComplete="new-password"
                minLength={6}
                maxLength={64}
                value={confirmPassword}
                onChange={(e) => setConfirmPassword(e.target.value)}
                placeholder={t('auth.confirmPasswordPlaceholder')}
                required
              />
            </div>

            <Button
              type="submit"
              className="mt-2"
              disabled={
                confirming ||
                code.length < 4 ||
                !newPassword ||
                newPassword !== confirmPassword
              }
            >
              {confirming ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <ShieldCheck className="size-4" />
              )}
              {confirming
                ? t('forgotPassword.resetting')
                : t('forgotPassword.resetPassword')}
            </Button>
            <Button
              type="button"
              variant="ghost"
              onClick={backToRequest}
              disabled={confirming}
            >
              <ArrowLeft className="size-4" />
              {t('forgotPassword.changeIdentifier')}
            </Button>
          </form>
        )}
      </div>
    </>
  )
}
