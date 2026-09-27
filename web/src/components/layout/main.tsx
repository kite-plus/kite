import { cn } from '@/lib/utils'
import { StudioFooter } from './studio-footer'

type MainProps = React.HTMLAttributes<HTMLElement> & {
  fixed?: boolean
  fluid?: boolean
  ref?: React.Ref<HTMLElement>
}

// Kite: every page closes with the studio's footer, at the foot of the
// window when the page is short, and under what scrolls when it is fixed.
export function Main({ fixed, className, fluid, children, ...props }: MainProps) {
  return (
    <main
      data-layout={fixed ? 'fixed' : 'auto'}
      className={cn(
        'px-4 py-6',

        // If layout is fixed, make the main container flex and grow
        fixed && 'flex grow flex-col overflow-hidden',

        // If layout is not fluid, set the max-width
        !fluid &&
          '@7xl/content:mx-auto @7xl/content:w-full @7xl/content:max-w-7xl',
        className
      )}
      {...props}
    >
      {children}
      <StudioFooter className={fixed ? 'mt-4 shrink-0' : 'mt-auto'} />
    </main>
  )
}
