import { Link } from '@tanstack/react-router'
import { ExternalLink, Languages, LogOut, UserCog } from 'lucide-react'
import { locales, useI18n, type Locale } from '@/i18n'
import { useSite } from '@/hooks/useContents'
import { useSession } from '@/hooks/useSession'
import { siteHome } from '@/lib/links'
import { getDisplayNameInitials } from '@/lib/utils'
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
 * useAccount says who the studio is being used by. A project with no password
 * has no user, and says so rather than inventing one.
 */
export function useAccount() {
  const { t } = useI18n()
  const session = useSession()
  const user = session.data?.required ? session.data.user : undefined
  const name = user ?? t('session.local')
  return {
    user,
    name,
    initials: getDisplayNameInitials(name),
    note: user ? t('session.role') : t('session.localNote'),
  }
}

/**
 * AccountMenuItems are the things that belong to the person rather than the
 * site: the way to the site, the studio's language and appearance, and the
 * way out, which a project with no password does not offer.
 */
export function AccountMenuItems({ onSignOut }: { onSignOut: () => void }) {
  const { t, locale, setLocale } = useI18n()
  const site = useSite()
  const { user } = useAccount()

  return (
    <>
      <DropdownMenuGroup>
        <DropdownMenuItem asChild>
          <a href={siteHome(site.data)} target='_blank' rel='noreferrer'>
            <ExternalLink />
            {t('nav.viewSite')}
          </a>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link to='/settings/appearance'>
            <UserCog />
            {t('nav.appearance')}
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
      {user && (
        <>
          <DropdownMenuSeparator />
          <DropdownMenuItem variant='destructive' onClick={onSignOut}>
            <LogOut />
            {t('session.signOut')}
          </DropdownMenuItem>
        </>
      )}
    </>
  )
}
