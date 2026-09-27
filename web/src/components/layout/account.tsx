import { Link } from '@tanstack/react-router'
import { ExternalLink, KeyRound, Languages, LogOut, UserRound } from 'lucide-react'
import { locales, useI18n, type Locale } from '@/i18n'
import { useAccountInfo } from '@/hooks/useAccount'
import { useSite } from '@/hooks/useContents'
import { useSession } from '@/hooks/useSession'
import { useSettings } from '@/hooks/useSettings'
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
 * the profile, else the site's author, since one person writes a Kite site,
 * else the name that signs in. Under the name goes the email, or how the
 * studio is reached: signed in, or on this computer with no password.
 */
export function useAccount() {
  const { t } = useI18n()
  const session = useSession()
  const info = useAccountInfo()
  const settings = useSettings()
  const user = session.data?.required ? session.data.user : undefined
  const profile = info.data?.profile
  const shown = profile?.name || settings.data?.site.author || user

  let note: string
  if (profile?.email) note = profile.email
  else if (user) note = shown === user ? t('session.role') : t('session.signedInAs', { user })
  else note = t('session.local')

  return {
    user,
    name: shown || t('session.role'),
    note,
    avatar: info.data?.avatar,
    // Initials only from a name somebody chose; nobody named gets a figure.
    initials: shown ? getDisplayNameInitials(shown) : undefined,
    tint: shown ? tint(shown) : undefined,
    // Only a server that can write the account offers to set a password.
    canSetPassword: Boolean(info.data && !info.data.protected && info.data.editable),
  }
}

/** UserAvatar is the picture, or initials in the name's own color. */
export function UserAvatar({ className, large }: { className?: string; large?: boolean }) {
  const account = useAccount()

  return (
    <Avatar className={cn(large ? 'size-16' : 'size-8', 'rounded-full', className)}>
      {account.avatar && (
        <AvatarImage src={account.avatar} alt='' className='object-cover' />
      )}
      <AvatarFallback
        className={cn(
          'rounded-full font-medium',
          large ? 'text-xl' : 'text-xs',
          account.tint
        )}
      >
        {account.initials ?? (
          <UserRound
            className={cn(large ? 'size-7' : 'size-4', 'text-muted-foreground')}
          />
        )}
      </AvatarFallback>
    </Avatar>
  )
}

/**
 * AccountMenuItems are the things that belong to the person rather than the
 * site: their account, the way to the site, the studio's language, and the
 * way out -- or, where there is no password, the way to set one. The
 * studio's appearance is in the sidebar with the other settings.
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
