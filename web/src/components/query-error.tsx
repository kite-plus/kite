import { ApiError } from '@/api/client'
import { useI18n, useProblem } from '@/i18n'
import { Button } from '@/components/ui/button'

/**
 * QueryError stands in for a card or a list whose data failed to load, where
 * a skeleton would otherwise wait for an answer that is not coming.
 */
export function QueryError({ error, onRetry }: { error: unknown; onRetry: () => void }) {
  const { t } = useI18n()
  const problem = useProblem()
  const said =
    error instanceof ApiError
      ? problem(error.code, error.message).title
      : error instanceof Error
        ? error.message
        : String(error)

  return (
    <div className='flex flex-col items-center gap-3 rounded-md border border-dashed p-6 text-center'>
      <p className='font-medium'>{t('error.loadFailed')}</p>
      <p className='text-sm text-muted-foreground'>{said}</p>
      <Button variant='outline' size='sm' onClick={onRetry}>
        {t('session.retry')}
      </Button>
    </div>
  )
}
