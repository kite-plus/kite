import { Separator } from '@/components/ui/separator'
import { cn } from '@/lib/utils'
import { PublishBar } from './publish-bar'

type ContentSectionProps = {
  title: string
  desc: string
  children: React.JSX.Element
  // Kite: the theme gallery needs the whole column, not a form's width.
  wide?: boolean
}

// Kite: a section heads the page, with what is waiting to be published under
// its title, since the app sidebar is the menu of the sections.
export function ContentSection({ title, desc, children, wide }: ContentSectionProps) {
  return (
    <div className='flex flex-1 flex-col overflow-hidden'>
      <div className='flex-none space-y-0.5'>
        <h1 className='text-2xl font-bold tracking-tight md:text-3xl'>{title}</h1>
        <p className='text-muted-foreground'>{desc}</p>
      </div>
      <Separator className='my-4 flex-none lg:my-6' />
      <PublishBar />
      <div className='faded-bottom h-full w-full overflow-y-auto scroll-smooth pe-4 pb-12'>
        <div className={cn('-mx-1 px-1.5', !wide && 'lg:max-w-2xl')}>{children}</div>
      </div>
    </div>
  )
}
