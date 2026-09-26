import { useI18n } from '@/i18n'
import { useSignOut } from '@/hooks/useSession'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface SignOutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SignOutDialog({ open, onOpenChange }: SignOutDialogProps) {
  const { t } = useI18n()
  const signOut = useSignOut()

  // Kite: the layout takes a signed-out browser to the sign-in form, with the
  // way back to this page. Navigating here as well raced it, and the second
  // one nested the sign-in address inside the way back.
  const handleSignOut = () => {
    signOut.mutate(undefined, { onSuccess: () => onOpenChange(false) })
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
