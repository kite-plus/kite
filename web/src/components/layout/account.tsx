import { UserRound } from 'lucide-react'
import { useI18n } from '@/i18n'
import { useAccountInfo } from '@/hooks/useAccount'
import { useSession } from '@/hooks/useSession'
import { useSettings } from '@/hooks/useSettings'
import { tint } from '@/lib/tint'
import { cn, getDisplayNameInitials } from '@/lib/utils'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

/**
 * useAccount says who the studio is being used by: the name and picture from
 * the profile, else the site's author, since one person writes a Kite site,
 * else the name that signs in.
 */
export function useAccount() {
  const { t } = useI18n()
  const session = useSession()
  const info = useAccountInfo()
  const settings = useSettings()
  const user = session.data?.required ? session.data.user : undefined
  const profile = info.data?.profile
  const shown = profile?.name || settings.data?.site.author || user

  // Nothing is named while the answers are on their way, rather than a
  // stand-in that a moment later changes.
  const loading = info.isPending || settings.isPending

  return {
    user,
    name: shown || (loading ? '' : t('session.role')),
    email: profile?.email || undefined,
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
