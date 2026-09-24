import {
  ChevronLeftIcon,
  ChevronRightIcon,
  ChevronsLeftIcon,
} from 'lucide-react'
import { useI18n } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

type DataTablePaginationProps = {
  /** page is 1-based: how many pages this listing has walked through. */
  page: number
  pageSize: number
  /** total is every row the filters match, when the server counted them. */
  total?: number
  /** summary is said beside the page size, such as how many rows there are. */
  summary?: React.ReactNode
  hasPrevious: boolean
  hasNext: boolean
  onFirst: () => void
  onPrevious: () => void
  onNext: () => void
  onPageSizeChange: (pageSize: number) => void
  className?: string
}

/**
 * Kite: pages walked by cursor rather than numbered. The API has no offset,
 * because over a set that is being edited an offset repeats and skips rows,
 * so there is no jumping to page seven; first, back and on are what exist.
 */
export function DataTablePagination({
  page,
  pageSize,
  total,
  summary,
  hasPrevious,
  hasNext,
  onFirst,
  onPrevious,
  onNext,
  onPageSizeChange,
  className,
}: DataTablePaginationProps) {
  const { t } = useI18n()
  const pages = total === undefined ? undefined : Math.max(1, Math.ceil(total / pageSize))

  return (
    <div
      className={cn(
        'flex items-center justify-between overflow-clip px-2',
        '@max-2xl/content:flex-col-reverse @max-2xl/content:gap-4',
        className
      )}
      style={{ overflowClipMargin: 1 }}
    >
      <div className='flex w-full items-center justify-between'>
        <div className='text-sm text-muted-foreground'>{summary}</div>
        <div className='flex items-center gap-2 @max-2xl/content:flex-row-reverse'>
          <Select
            value={`${pageSize}`}
            onValueChange={(value) => onPageSizeChange(Number(value))}
          >
            <SelectTrigger className='h-8 w-17.5'>
              <SelectValue placeholder={pageSize} />
            </SelectTrigger>
            <SelectContent side='top'>
              {[10, 20, 30, 50].map((size) => (
                <SelectItem key={size} value={`${size}`}>
                  {size}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <p className='hidden text-sm font-medium sm:block'>
            {t('list.perPage')}
          </p>
        </div>
      </div>

      <div className='flex items-center sm:space-x-6 lg:space-x-8'>
        <div className='flex min-w-25 items-center justify-center text-sm font-medium'>
          {pages === undefined
            ? t('list.pageN', { page })
            : t('list.page', { page, pages })}
        </div>
        <div className='flex items-center space-x-2'>
          <Button
            variant='outline'
            className='size-8 p-0 @max-md/content:hidden'
            onClick={onFirst}
            disabled={!hasPrevious}
          >
            <span className='sr-only'>{t('list.first')}</span>
            <ChevronsLeftIcon className='h-4 w-4' />
          </Button>
          <Button
            variant='outline'
            className='size-8 p-0'
            onClick={onPrevious}
            disabled={!hasPrevious}
          >
            <span className='sr-only'>{t('list.previous')}</span>
            <ChevronLeftIcon className='h-4 w-4' />
          </Button>
          <Button
            variant='outline'
            className='size-8 p-0'
            onClick={onNext}
            disabled={!hasNext}
          >
            <span className='sr-only'>{t('list.next')}</span>
            <ChevronRightIcon className='h-4 w-4' />
          </Button>
        </div>
      </div>
    </div>
  )
}
