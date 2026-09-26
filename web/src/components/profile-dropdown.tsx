import { useI18n } from '@/i18n'
import useDialogState from '@/hooks/use-dialog-state'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  AccountMenuItems,
  UserAvatar,
  useAccount,
} from '@/components/layout/account'
import { SignOutDialog } from '@/components/sign-out-dialog'

export function ProfileDropdown() {
  const { t } = useI18n()
  const [open, setOpen] = useDialogState()
  const account = useAccount()

  return (
    <>
      <DropdownMenu modal={false}>
        <DropdownMenuTrigger asChild>
          <Button
            variant='ghost'
            className='relative h-8 w-8 rounded-full'
            aria-label={t('nav.account')}
          >
            <UserAvatar />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent className='w-56' align='end' forceMount>
          <DropdownMenuLabel className='font-normal'>
            <div className='flex flex-col gap-1.5'>
              <p className='text-sm leading-none font-medium'>{account.name}</p>
              <p className='text-xs leading-none text-muted-foreground'>
                {account.note}
              </p>
            </div>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <AccountMenuItems onSignOut={() => setOpen(true)} />
        </DropdownMenuContent>
      </DropdownMenu>

      <SignOutDialog open={!!open} onOpenChange={setOpen} />
    </>
  )
}
