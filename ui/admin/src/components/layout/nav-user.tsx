import {
  LogOut,
  User,
} from 'lucide-react'
import { Link } from 'react-router'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import {
  SidebarMenu,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { useLogout } from '@/hooks/use-auth'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

type NavUserProps = {
  user: {
    name: string
    email: string
    avatar: string
  }
}

export function NavUser({ user }: NavUserProps) {
  const logoutMutation = useLogout()
  const { state } = useSidebar()
  const isCollapsed = state === 'collapsed'

  /** 获取名字首字母作为 Avatar fallback */
  const initials = user.name.slice(0, 2).toUpperCase()

  function handleLogout() {
    logoutMutation.mutate(undefined, {
      onSuccess: () => {
        window.location.href = '/admin/login'
      },
    })
  }

  // 侧边栏折叠：只显示居中头像
  if (isCollapsed) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <div className="flex items-center justify-center py-1.5">
            <Tooltip>
              <TooltipTrigger asChild>
                <Link to="/profile">
                  <Avatar className="h-8 w-8 rounded-lg shrink-0">
                    {user.avatar && <AvatarImage src={user.avatar} alt={user.name} className="rounded-lg" />}
                    <AvatarFallback className="rounded-lg bg-primary/10 text-primary font-medium">
                      {initials}
                    </AvatarFallback>
                  </Avatar>
                </Link>
              </TooltipTrigger>
              <TooltipContent side="right" className="text-xs">{user.name}</TooltipContent>
            </Tooltip>
          </div>
        </SidebarMenuItem>
      </SidebarMenu>
    )
  }

  // 侧边栏展开：完整显示
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <div className="flex items-center gap-2 px-2 py-1.5 rounded-md hover:bg-sidebar-accent/50 transition-colors">
          <Link to="/profile" className="flex items-center gap-2 flex-1 min-w-0 group">
            <Avatar className="h-8 w-8 rounded-lg shrink-0">
              {user.avatar && <AvatarImage src={user.avatar} alt={user.name} className="rounded-lg" />}
              <AvatarFallback className="rounded-lg bg-primary/10 text-primary font-medium">
                {initials}
              </AvatarFallback>
            </Avatar>
            <span className="truncate text-sm font-semibold group-hover:text-sidebar-accent-foreground transition-colors">
              {user.name}
            </span>
          </Link>

          <div className="flex items-center gap-0.5 shrink-0">
            <Tooltip>
              <TooltipTrigger asChild>
                <Link
                  to="/profile"
                  className="w-7 h-7 flex items-center justify-center rounded-md text-muted-foreground hover:text-foreground hover:bg-sidebar-accent transition-colors"
                >
                  <User className="w-4 h-4" />
                </Link>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs">个人资料</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  onClick={handleLogout}
                  disabled={logoutMutation.isPending}
                  className="w-7 h-7 flex items-center justify-center rounded-md text-muted-foreground hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors cursor-pointer disabled:opacity-50"
                >
                  <LogOut className="w-4 h-4" />
                </button>
              </TooltipTrigger>
              <TooltipContent side="top" className="text-xs">退出登录</TooltipContent>
            </Tooltip>
          </div>
        </div>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}

