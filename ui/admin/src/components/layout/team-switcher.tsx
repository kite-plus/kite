import * as React from 'react'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar'

type TeamSwitcherProps = {
  teams: {
    name: string
    logo: React.ElementType
    plan: string
  }[]
}

export function TeamSwitcher({ teams }: TeamSwitcherProps) {
  const [activeTeam] = React.useState(teams[0])

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          size='lg'
          className='hover:bg-transparent hover:text-sidebar-accent-foreground'
          asChild
        >
          <a href='https://www.kite.plus' target='_blank' rel='noopener noreferrer'>
          <div className='flex aspect-square size-12 items-center justify-center'>
            <activeTeam.logo className='size-12' />
          </div>
          <div className='grid flex-1 text-start leading-tight'>
            <span className='truncate text-lg font-bold tracking-tight'>
              {activeTeam.name}
            </span>
            <span className='truncate text-[11px] text-muted-foreground'>{activeTeam.plan}</span>
          </div>
          </a>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
