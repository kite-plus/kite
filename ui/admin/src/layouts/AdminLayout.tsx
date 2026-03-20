import { Outlet } from 'react-router'
import { getCookie } from '@/lib/cookies'
import { cn } from '@/lib/utils'
import { LayoutProvider } from '@/context/layout-provider'
import { SearchProvider } from '@/context/search-provider'
import { SidebarInset, SidebarProvider } from '@/components/ui/sidebar'
import { AppSidebar } from '@/components/layout/app-sidebar'

/**
 * 管理后台主布局 — shadcn-admin 风格
 * SidebarProvider + AppSidebar + SidebarInset
 */
export function AdminLayout() {
  const defaultOpen = getCookie('sidebar_state') !== 'false'
  return (
    <SearchProvider>
      <LayoutProvider>
        <SidebarProvider defaultOpen={defaultOpen}>
          <AppSidebar />
          <SidebarInset
            className={cn(
              '@container/content',
              'has-data-[layout=fixed]:h-svh',
              'peer-data-[variant=inset]:has-data-[layout=fixed]:h-[calc(100svh-(var(--spacing)*4))]'
            )}
          >
            <Outlet />
            <footer className='mt-auto border-t px-4 py-4 text-center text-xs text-muted-foreground'>
              © {new Date().getFullYear()}{' '}
              <a href='https://github.com/amigoer/kite' target='_blank' rel='noopener noreferrer' className='hover:text-primary underline-offset-4 hover:underline'>
                Kite
              </a>{' '}
              · 轻量级博客引擎
            </footer>
          </SidebarInset>
        </SidebarProvider>
      </LayoutProvider>
    </SearchProvider>
  )
}
