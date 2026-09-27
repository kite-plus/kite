import { Link, useLocation } from '@tanstack/react-router'
import { CircleUserRound, LogOut } from 'lucide-react'
import { useI18n } from '@/i18n'
import useDialogState from '@/hooks/use-dialog-state'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { SignOutDialog } from '@/components/sign-out-dialog'
import { UserAvatar, useAccount } from './account'

/**
 * NavUser is the person at the foot of the sidebar, with two buttons beside
 * them and no menu: their profile, and the way out. Where nobody signs in
 * there is no way out, and the button says so instead.
 */
export function NavUser() {
  const { t } = useI18n()
  const { state, isMobile, setOpenMobile } = useSidebar()
  const [open, setOpen] = useDialogState()
  const account = useAccount()
  const path = useLocation({ select: (location) => location.pathname })
  const onProfile = path === '/settings/account'
  const signOutLabel = account.user ? t('session.signOut') : t('session.noSignOut')

  if (state === 'collapsed' && !isMobile) {
    return (
      <>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton asChild tooltip={t('session.profile')} isActive={onProfile}>
              <Link to='/settings/account'>
                <CircleUserRound />
                <span>{t('session.profile')}</span>
              </Link>
            </SidebarMenuButton>
          </SidebarMenuItem>
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip={signOutLabel}
              aria-disabled={!account.user}
              className='aria-disabled:opacity-50'
              onClick={() => account.user && setOpen(true)}
            >
              <LogOut />
              <span>{t('session.signOut')}</span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
        <SignOutDialog open={!!open} onOpenChange={setOpen} />
      </>
    )
  }

  return (
    <>
      <div className='flex h-12 items-center gap-2 px-2'>
        <UserAvatar />
        <div className='grid min-w-0 flex-1 text-start text-sm leading-tight'>
          {account.name ? (
            <span className='truncate font-medium'>{account.name}</span>
          ) : (
            <Skeleton className='h-4 w-20' />
          )}
          {account.email && (
            <span className='truncate text-xs text-muted-foreground'>{account.email}</span>
          )}
        </div>
        <Tooltip>
          <TooltipTrigger asChild>
            <Button
              variant='ghost'
              size='icon'
              className={cn(
                'size-8 text-muted-foreground hover:text-foreground',
                onProfile && 'bg-sidebar-accent text-foreground'
              )}
              aria-label={t('session.profile')}
              asChild
            >
              <Link to='/settings/account' onClick={() => setOpenMobile(false)}>
                <CircleUserRound />
              </Link>
            </Button>
          </TooltipTrigger>
          <TooltipContent side='top'>{t('session.profile')}</TooltipContent>
        </Tooltip>
        <Tooltip>
          {/* A disabled button takes no pointer, so its hint hangs on a wrapper. */}
          <TooltipTrigger asChild>
            <span tabIndex={account.user ? undefined : 0}>
              <Button
                variant='ghost'
                size='icon'
                className='size-8 text-muted-foreground hover:text-foreground'
                aria-label={signOutLabel}
                disabled={!account.user}
                onClick={() => setOpen(true)}
              >
                <LogOut />
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent side='top' className='max-w-56'>
            {signOutLabel}
          </TooltipContent>
        </Tooltip>
      </div>

      <SignOutDialog open={!!open} onOpenChange={setOpen} />
    </>
  )
}
