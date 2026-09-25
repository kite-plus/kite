import { useNavigate, useRouter } from '@tanstack/react-router'
import { useI18n } from '@/i18n'
import { useDocumentTitle } from '@/hooks/useDocumentTitle'
import { Button } from '@/components/ui/button'

export function NotFoundError() {
  const { t } = useI18n()
  useDocumentTitle(t('error.notFound'))
  const navigate = useNavigate()
  const { history } = useRouter()
  return (
    <div className='h-svh'>
      <div className='m-auto flex h-full w-full flex-col items-center justify-center gap-2'>
        <h1 className='text-[7rem] leading-tight font-bold'>404</h1>
        <span className='font-medium'>{t('error.notFound')}</span>
        <p className='text-center text-muted-foreground'>
          {t('error.notFoundNote')}
        </p>
        <div className='mt-6 flex gap-4'>
          <Button variant='outline' onClick={() => history.go(-1)}>
            {t('error.goBack')}
          </Button>
          <Button onClick={() => navigate({ to: '/' })}>{t('error.home')}</Button>
        </div>
      </div>
    </div>
  )
}
