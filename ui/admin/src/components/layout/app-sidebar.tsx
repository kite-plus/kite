import { useLayout } from '@/context/layout-provider'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar'
import { sidebarData } from './data/sidebar-data'
import { NavGroup } from './nav-group'
import { NavUser } from './nav-user'
import { TeamSwitcher } from './team-switcher'
import { useCurrentUser } from '@/hooks/use-auth'

export function AppSidebar() {
  const { collapsible, variant } = useLayout()
  const { data: currentUser } = useCurrentUser()

  // 从 API 获取真实用户数据，回退到默认值
  const user = {
    name: currentUser?.user.displayName || currentUser?.user.username || 'Admin',
    email: currentUser?.user.email || '',
    avatar: currentUser?.user.avatar || '',
  }

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <TeamSwitcher teams={sidebarData.teams} />
      </SidebarHeader>
      <SidebarContent>
        {sidebarData.navGroups.map((props) => (
          <NavGroup key={props.title} {...props} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
