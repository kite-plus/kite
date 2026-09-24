import { useI18n } from '@/i18n'
import { KiteMark } from '@/components/KiteMark'
import { LanguagePicker } from '@/components/language-picker'

type AuthLayoutProps = {
  children: React.ReactNode
}

/**
 * The door, and what it says: no site title and nothing about what is
 * behind it. Everything a studio knows comes from an API that will not answer
 * until somebody has signed in.
 */
export function AuthLayout({ children }: AuthLayoutProps) {
  return (
    <div className='container grid h-svh max-w-none items-center justify-center'>
      <div className='mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:p-8'>
        <div className='mb-4 flex items-center justify-center'>
          <KiteMark className='me-2 size-7' />
          <h1 className='text-xl font-medium'>Kite</h1>
        </div>
        {children}
        <AuthFooter />
      </div>
    </div>
  )
}

/** AuthFooter is the way back out, and the language to read the form in. */
export function AuthFooter() {
  const { t } = useI18n()
  return (
    <div className='flex items-center justify-center gap-2 pt-2 text-xs text-muted-foreground'>
      {/* The server sends the bare host to the site's home page. */}
      <a href='/' className='transition-colors hover:text-foreground'>
        {t('login.backToSite')}
      </a>
      <span aria-hidden>·</span>
      <LanguagePicker />
    </div>
  )
}
