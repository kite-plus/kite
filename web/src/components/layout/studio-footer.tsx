import { useI18n } from '@/i18n'
import { useSite } from '@/hooks/useContents'
import { cn } from '@/lib/utils'
import { KiteMark } from '@/components/KiteMark'

const website = 'https://www.kite.plus'
const repository = 'https://github.com/kite-plus/kite'

/**
 * StudioFooter closes a page of the studio: what it runs on, and the way to
 * Kite's documentation, its source and its issue tracker.
 */
export function StudioFooter({ className }: { className?: string }) {
  const { t } = useI18n()
  const site = useSite()
  const version = site.data?.version

  return (
    <footer
      className={cn(
        'flex flex-wrap items-center justify-between gap-x-6 gap-y-2 border-t pt-4 text-xs text-muted-foreground',
        className
      )}
    >
      <p className='flex min-w-0 items-center gap-1.5'>
        <KiteMark className='size-4' />
        <Outside href={website}>{t('footer.poweredBy')}</Outside>
        {version && <span className='truncate font-mono'>{version}</span>}
      </p>
      <nav aria-label={t('footer.links')} className='flex items-center gap-4'>
        <Outside href={`${website}/docs/`}>{t('footer.docs')}</Outside>
        <Outside href={repository} className='flex items-center gap-1'>
          <GitHubMark />
          GitHub
        </Outside>
        <Outside href={`${repository}/issues`}>{t('footer.issues')}</Outside>
      </nav>
    </footer>
  )
}

function Outside({
  href,
  className,
  children,
}: {
  href: string
  className?: string
  children: React.ReactNode
}) {
  return (
    <a
      href={href}
      target='_blank'
      rel='noreferrer'
      className={cn('transition-colors hover:text-foreground', className)}
    >
      {children}
    </a>
  )
}

/** GitHubMark is GitHub's own mark, from its Octicons. */
function GitHubMark() {
  return (
    <svg viewBox='0 0 16 16' className='size-3.5 shrink-0' fill='currentColor' aria-hidden>
      <path d='M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0 0 16 8c0-4.42-3.58-8-8-8z' />
    </svg>
  )
}
