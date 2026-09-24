import { useNavigate, useRouter, type ErrorComponentProps } from '@tanstack/react-router'
import { ApiError } from '@/api/client'
import { useI18n, useProblem } from '@/i18n'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'

type GeneralErrorProps = Partial<ErrorComponentProps> & {
  className?: string
  minimal?: boolean
}

/**
 * GeneralError is what a screen shows when it could not be drawn. The most
 * common cause is a server that stopped answering, so it says what went
 * wrong and offers to ask again rather than only to go elsewhere.
 */
export function GeneralError({
  className,
  minimal = false,
  error,
  reset,
}: GeneralErrorProps) {
  const { t } = useI18n()
  const problem = useProblem()
  const navigate = useNavigate()
  const router = useRouter()

  const retry = () => {
    reset?.()
    void router.invalidate()
  }

  return (
    <div className={cn('h-svh w-full', className)}>
      <div className='m-auto flex h-full w-full flex-col items-center justify-center gap-2 px-4'>
        {!minimal && (
          <h1 className='text-[7rem] leading-tight font-bold'>500</h1>
        )}
        <span className='font-medium'>{t('error.general')}</span>
        <p className='text-center text-muted-foreground'>
          {t('error.generalNote')}
        </p>
        {error instanceof ApiError ? (
          <p className='max-w-lg text-center text-sm text-muted-foreground'>
            {problem(error.code, error.message).title}
          </p>
        ) : (
          error instanceof Error && (
            <p className='max-w-lg text-center font-mono text-xs break-all text-muted-foreground'>
              {error.message}
            </p>
          )
        )}
        {!minimal && (
          <div className='mt-6 flex gap-4'>
            <Button variant='outline' onClick={retry}>
              {t('session.retry')}
            </Button>
            <Button onClick={() => navigate({ to: '/' })}>{t('error.home')}</Button>
          </div>
        )}
      </div>
    </div>
  )
}
