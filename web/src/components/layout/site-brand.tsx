import { Link } from '@tanstack/react-router'
import { useI18n } from '@/i18n'
import { useSite, useWritable } from '@/hooks/useContents'
import { useSession } from '@/hooks/useSession'
import { Badge } from '@/components/ui/badge'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { KiteMark } from '@/components/KiteMark'

/**
 * SiteBrand heads the rail with the site this studio edits and leads back to
 * the dashboard. It takes the place of shadcn-admin's team switcher: a studio
 * has one site, so there is nothing to switch to, and the site's own links
 * sit on the dashboard and under settings. A studio nobody signs in to says
 * so beside the site's name, as does one that takes no changes.
 */
export function SiteBrand() {
  const { t } = useI18n()
  const { setOpenMobile } = useSidebar()
  const site = useSite()
  const session = useSession()
  const title = site.data?.title ?? 'Kite'
  const local = session.data?.required === false
  const readOnly = !useWritable()
  const marks = [local && t('session.local'), readOnly && t('session.readOnly')].filter(Boolean)

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          size='lg'
          tooltip={[title, ...marks].join(' · ')}
          asChild
        >
          <Link to='/' onClick={() => setOpenMobile(false)}>
            {/* Wrapped, as the button would size a bare svg down to 16px. The
                mark's own margin is pulled in to line up with the rail's icons. */}
            <span className='-ms-1.5 flex shrink-0 group-data-[collapsible=icon]:ms-0'>
              <KiteMark className='size-10 group-data-[collapsible=icon]:size-8' />
            </span>
            <div className='grid flex-1 text-start leading-tight'>
              <span className='flex min-w-0 items-center gap-1.5'>
                <span className='truncate text-base font-semibold'>{title}</span>
                {marks.map((mark) => (
                  <Badge key={String(mark)} variant='secondary' className='shrink-0 px-1.5 py-0 text-[11px] font-normal'>
                    {mark}
                  </Badge>
                ))}
              </span>
              <span className='truncate text-xs text-muted-foreground'>
                Kite {site.data?.version ?? ''}
              </span>
            </div>
          </Link>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
