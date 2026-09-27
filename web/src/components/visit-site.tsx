import { ExternalLink } from 'lucide-react'
import { useI18n } from '@/i18n'
import { useSite } from '@/hooks/useContents'
import { siteHome } from '@/lib/links'
import { Button } from '@/components/ui/button'

/** VisitSite opens the site in a new tab, from every screen's header. */
export function VisitSite() {
  const { t } = useI18n()
  const site = useSite()
  const label = t('nav.viewSite')

  return (
    <Button
      variant='ghost'
      size='icon'
      className='scale-95 rounded-full'
      aria-label={label}
      title={label}
      asChild
    >
      <a href={siteHome(site.data)} target='_blank' rel='noreferrer'>
        <ExternalLink className='size-[1.2rem]' />
      </a>
    </Button>
  )
}
