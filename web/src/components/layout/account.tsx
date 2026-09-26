import { Link } from '@tanstack/react-router'
import {
  ExternalLink,
  KeyRound,
  Languages,
  Laptop,
  LogOut,
  MonitorCog,
  UserRound,
} from 'lucide-react'
import { locales, useI18n, type Locale } from '@/i18n'
import { useAccountInfo } from '@/hooks/useAccount'
import { useSite } from '@/hooks/useContents'
import { useSession } from '@/hooks/useSession'
import { siteHome } from '@/lib/links'
import { tint } from '@/lib/tint'
import { cn, getDisplayNameInitials } from '@/lib/utils'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
} from '@/components/ui/dropdown-menu'

/**
 * useAccount says who the studio is being used by: the name and picture from
 * the profile, else the name that signs in. A project with no password and
 * no profile has nobody to name, and says it is in local mode rather than
 * inventing somebody.
 */
export function useAccount() {
  const { t } = useI18n()
  const session = useSession()
  const info = useAccountInfo()
  const user = session.data?.required ? session.data.user : undefined
  const profile = info.data?.profile
  const shown = profile?.name || user

  let note: string
  if (profile?.email) note = profile.email
  else if (user) note = profile?.name ? t('session.signedInAs', { user }) : t('session.role')
  else note = profile?.name ? t('session.localMode') : t('session.localNote')

  return {
    user,
    name: shown ?? t('session.local'),
    note,
    avatar: info.data?.avatar,
    // Initials only from a name somebody chose; local mode gets a computer.
    initials: shown ? getDisplayNameInitials(shown) : undefined,
    tint: shown ? tint(shown) : undefined,
    // Only a server that can write the account offers to set a password.
    canSetPassword: Boolean(info.data && !info.data.protected && info.data.editable),
  }
}

/** UserAvatar is the picture, or initials in the name's own color. */
export function UserAvatar({
  className,
  square,
  large,
}: {
  className?: string
  square?: boolean
  large?: boolean
}) {
  const account = useAccount()
  const shape = square ? 'rounded-lg' : 'rounded-full'

  return (
    <Avatar className={cn(large ? 'size-16' : 'size-8', shape, className)}>
      {account.avatar && (
        <AvatarImage src={account.avatar} alt='' className='object-cover' />
      )}
      <AvatarFallback
        className={cn(
          shape,
          large ? 'text-xl' : 'text-xs',
          'font-medium',
          account.tint
        )}
      >
        {account.initials ?? (
          <Laptop
            className={cn(large ? 'size-7' : 'size-4', 'text-muted-foreground')}
          />
        )}
      </AvatarFallback>
    </Avatar>
  )
}

/**
 * AccountMenuItems are the things that belong to the person rather than the
 * site: their account, the way to the site, the studio's language and
 * appearance, and the way out -- or, where there is no password, the way to
 * set one.
 */
export function AccountMenuItems({ onSignOut }: { onSignOut: () => void }) {
  const { t, locale, setLocale } = useI18n()
  const site = useSite()
  const { user, canSetPassword } = useAccount()

  return (
    <>
      <DropdownMenuGroup>
        <DropdownMenuItem asChild>
          <Link to='/settings/account'>
            <UserRound />
            {t('nav.account')}
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <a href={siteHome(site.data)} target='_blank' rel='noreferrer'>
            <ExternalLink />
            {t('nav.viewSite')}
          </a>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link to='/settings/appearance'>
            <MonitorCog />
            {t('settings.interface')}
          </Link>
        </DropdownMenuItem>
        <DropdownMenuSub>
          <DropdownMenuSubTrigger>
            <Languages />
            {t('nav.language')}
          </DropdownMenuSubTrigger>
          <DropdownMenuSubContent>
            <DropdownMenuRadioGroup
              value={locale}
              onValueChange={(next) => setLocale(next as Locale)}
            >
              {(Object.keys(locales) as Locale[]).map((code) => (
                <DropdownMenuRadioItem key={code} value={code}>
                  {locales[code].label}
                </DropdownMenuRadioItem>
              ))}
            </DropdownMenuRadioGroup>
          </DropdownMenuSubContent>
        </DropdownMenuSub>
      </DropdownMenuGroup>
      {user ? (
        <>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant='destructive' onClick={onSignOut}>
            <LogOut />
            {t('session.signOut')}
          </DropdownMenuItem>
        </>
      ) : (
        canSetPassword && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link to='/settings/account' hash='sign-in'>
                <KeyRound />
                {t('session.setPassword')}
              </Link>
            </DropdownMenuItem>
          </>
        )
      )}
    </>
  )
}
