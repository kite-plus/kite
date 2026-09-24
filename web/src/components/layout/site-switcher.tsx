import { Link } from '@tanstack/react-router'
import { ChevronsUpDown, ExternalLink, SlidersHorizontal } from 'lucide-react'
import { useI18n } from '@/i18n'
import { useSite } from '@/hooks/useContents'
import { siteHome } from '@/lib/links'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { KiteMark } from '@/components/KiteMark'

/**
 * SiteSwitcher heads the rail with the site this studio edits. It takes the
 * place shadcn-admin gives a team switcher: Kite has one site per studio, so
 * the menu leads to that site rather than to others.
 */
export function SiteSwitcher() {
  const { t } = useI18n()
  const { isMobile, setOpenMobile } = useSidebar()
  const site = useSite()

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size='lg'
              className='data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
            >
              <div className='flex aspect-square size-8 items-center justify-center rounded-lg border bg-background'>
                <KiteMark className='size-6' />
              </div>
              <div className='grid flex-1 text-start text-sm leading-tight'>
                <span className='truncate font-semibold'>
                  {site.data?.title ?? 'Kite'}
                </span>
                <span className='truncate text-xs text-muted-foreground'>
                  Kite {site.data?.version ?? ''}
                </span>
              </div>
              <ChevronsUpDown className='ms-auto' />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className='w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg'
            align='start'
            side={isMobile ? 'bottom' : 'right'}
            sideOffset={4}
          >
            <DropdownMenuLabel className='text-xs text-muted-foreground'>
              {site.data?.base_url}
            </DropdownMenuLabel>
            <DropdownMenuItem asChild className='gap-2 p-2'>
              <a href={siteHome(site.data)} target='_blank' rel='noreferrer'>
                <ExternalLink />
                {t('nav.viewSite')}
              </a>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild className='gap-2 p-2'>
              <Link to='/settings' onClick={() => setOpenMobile(false)}>
                <SlidersHorizontal />
                {t('settings.site')}
              </Link>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
