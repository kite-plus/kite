import { ConfigDrawer } from '@/components/config-drawer'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'
import { VisitSite } from '@/components/visit-site'
import { Header } from './header'

/**
 * AppHeader is the bar every screen opens with. shadcn-admin assembles it on
 * each page; Kite's screens all want the same one, so it is assembled here,
 * with room before the search for what a screen adds.
 */
export function AppHeader({
  fixed = true,
  children,
}: {
  fixed?: boolean
  children?: React.ReactNode
}) {
  return (
    <Header fixed={fixed}>
      {children}
      <Search className='me-auto' />
      <VisitSite />
      <ThemeSwitch />
      <ConfigDrawer />
    </Header>
  )
}
