import { Link, useLocation } from '@tanstack/react-router'
import { KeyRound, Laptop, LogOut, ShieldCheck, UserRound } from 'lucide-react'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import useDialogState from '@/hooks/use-dialog-state'
import { Avatar, AvatarImage } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { SignOutDialog } from '@/components/sign-out-dialog'
import { useAccount } from './account'

interface Action {
  label: string
  icon: React.ElementType
  to?: '/settings/account'
  hash?: string
  onClick?: () => void
}

/**
 * NavUser is the person at the foot of the sidebar, as Halo's console has
 * it: their name and what they are to the site, and beside them the way to
 * their account and the way out, or with no password yet the way to set
 * one. Nothing is folded into a menu.
 */
export function NavUser() {
  const { t } = useI18n()
  const { state, isMobile, setOpenMobile } = useSidebar()
  const [open, setOpen] = useDialogState()
  const account = useAccount()
  const path = useLocation({ select: (location) => location.pathname })
  const onAccount = path === '/settings/account'

  const actions: Action[] = [
    { label: t('nav.account'), icon: UserRound, to: '/settings/account' },
  ]
  if (account.user) {
    actions.push({ label: t('session.signOut'), icon: LogOut, onClick: () => setOpen(true) })
  } else if (account.canSetPassword) {
    actions.push({
      label: t('session.setPassword'),
      icon: KeyRound,
      to: '/settings/account',
      hash: 'sign-in',
    })
  }
  const leave = () => setOpenMobile(false)

  return (
    <>
      {state === 'collapsed' && !isMobile ? (
        <SidebarMenu>
          {actions.map((action) => (
            <SidebarMenuItem key={action.label}>
              <SidebarMenuButton
                asChild={Boolean(action.to)}
                tooltip={action.label}
                isActive={action.to !== undefined && !action.hash && onAccount}
                onClick={action.onClick}
              >
                {action.to ? (
                  <Link to={action.to} hash={action.hash}>
                    <action.icon />
                    <span>{action.label}</span>
                  </Link>
                ) : (
                  <>
                    <action.icon />
                    <span>{action.label}</span>
                  </>
                )}
              </SidebarMenuButton>
            </SidebarMenuItem>
          ))}
        </SidebarMenu>
      ) : (
        <div className='flex items-center gap-2 p-2'>
          {account.avatar && (
            <Avatar className='size-9 rounded-full'>
              <AvatarImage src={account.avatar} alt='' className='object-cover' />
            </Avatar>
          )}
          <div className='grid min-w-0 flex-1 justify-items-start gap-1'>
            {account.name ? (
              <span className='max-w-full truncate text-sm font-semibold'>{account.name}</span>
            ) : (
              <Skeleton className='h-5 w-20' />
            )}
            <Badge variant='outline' className='gap-1 font-normal text-muted-foreground'>
              {account.user ? <ShieldCheck /> : <Laptop />}
              {account.user ? t('session.role') : t('session.local')}
            </Badge>
          </div>
          <div className='flex shrink-0 items-center'>
            {actions.map((action) => (
              <Tooltip key={action.label}>
                <TooltipTrigger asChild>
                  <Button
                    variant='ghost'
                    size='icon'
                    className={cn(
                      'size-8 text-muted-foreground hover:text-foreground',
                      action.to && !action.hash && onAccount && 'bg-sidebar-accent text-foreground'
                    )}
                    aria-label={action.label}
                    asChild={Boolean(action.to)}
                    onClick={action.onClick}
                  >
                    {action.to ? (
                      <Link to={action.to} hash={action.hash} onClick={leave}>
                        <action.icon />
                      </Link>
                    ) : (
                      <action.icon />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent side='top'>{action.label}</TooltipContent>
              </Tooltip>
            ))}
          </div>
        </div>
      )}

      <SignOutDialog open={!!open} onOpenChange={setOpen} />
    </>
  )
}
