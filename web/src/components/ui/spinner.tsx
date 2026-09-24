import { Loader2 } from 'lucide-react'
import { cn } from '@/lib/utils'

// Kite: shadcn-admin writes the spinning Loader2 inline; the studio's own
// screens use it often enough to name it.
function Spinner({ className, ...props }: React.ComponentProps<typeof Loader2>) {
  return (
    <Loader2
      role='status'
      aria-label='Loading'
      className={cn('size-4 animate-spin', className)}
      {...props}
    />
  )
}

export { Spinner }
