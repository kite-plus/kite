import { Link } from '@tanstack/react-router'
import { useSite } from '@/hooks/useContents'
import { ConfigDrawer } from '@/components/config-drawer'
import { KiteMark } from '@/components/KiteMark'
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
      <PhoneMark />
      {children}
      <Search className='me-auto' />
      <VisitSite />
      <ThemeSwitch />
      <ConfigDrawer />
    </Header>
  )
}

/**
 * PhoneMark puts the mark in the bar on a phone, where the sidebar that
 * carries it is folded away, and leads to the dashboard as the sidebar's does.
 */
function PhoneMark() {
  const site = useSite()
  return (
    <Link to='/' aria-label={site.data?.title ?? 'Kite'} className='-mx-1 shrink-0 md:hidden'>
      <KiteMark className='size-8' />
    </Link>
  )
}
