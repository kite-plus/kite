import { Link } from '@tanstack/react-router'
import { useSite } from '@/hooks/useContents'
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
 * sit on the dashboard and under settings.
 */
export function SiteBrand() {
  const { setOpenMobile } = useSidebar()
  const site = useSite()
  const title = site.data?.title ?? 'Kite'

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size='lg' tooltip={title} asChild>
          <Link to='/' onClick={() => setOpenMobile(false)}>
            {/* Wrapped, as the button would size a bare svg down to 16px. The
                mark's own margin is pulled in to line up with the rail's icons. */}
            <span className='-ms-1.5 flex shrink-0 group-data-[collapsible=icon]:ms-0'>
              <KiteMark className='size-10 group-data-[collapsible=icon]:size-8' />
            </span>
            <div className='grid flex-1 text-start leading-tight'>
              <span className='truncate text-base font-semibold'>{title}</span>
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
