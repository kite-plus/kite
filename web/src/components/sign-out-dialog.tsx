import { useLocation, useNavigate } from '@tanstack/react-router'
import { useI18n } from '@/i18n'
import { useSignOut } from '@/hooks/useSession'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface SignOutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SignOutDialog({ open, onOpenChange }: SignOutDialogProps) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const location = useLocation()
  const signOut = useSignOut()

  const handleSignOut = () => {
    signOut.mutate(undefined, {
      onSuccess: () => {
        onOpenChange(false)
        // Signing in again comes back to where this left off.
        void navigate({
          to: '/sign-in',
          search: { redirect: location.href },
          replace: true,
        })
      },
    })
  }

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('session.signOut')}
      desc={t('session.signOutNote')}
      cancelBtnText={t('common.cancel')}
      confirmText={t('session.signOut')}
      destructive
      isLoading={signOut.isPending}
      handleConfirm={handleSignOut}
      className='sm:max-w-sm'
    />
  )
}
