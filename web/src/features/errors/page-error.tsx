import type { ErrorComponentProps } from '@tanstack/react-router'
import { AppHeader } from '@/components/layout/app-header'
import { Main } from '@/components/layout/main'
import { GeneralError } from './general-error'

/**
 * PageError is GeneralError inside the studio's layout, as explore's console
 * shows a page that failed: the sidebar and the header stay, to go elsewhere
 * with. Only a page's own route may use it, since it needs that layout.
 */
export function PageError(props: ErrorComponentProps) {
  return (
    <>
      <AppHeader />
      <Main className='flex flex-1'>
        <GeneralError {...props} className='h-auto flex-1' />
      </Main>
    </>
  )
}
