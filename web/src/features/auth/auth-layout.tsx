import { useI18n } from '@/i18n'
import { KiteMark } from '@/components/KiteMark'
import { LanguagePicker } from '@/components/language-picker'

type AuthLayoutProps = {
  title: React.ReactNode
  description: React.ReactNode
  children: React.ReactNode
  // Kite: a site being created has no pages to go back to yet.
  noSite?: boolean
}

/**
 * The door, and what it says: no site title and nothing about what is
 * behind it. Everything a studio knows comes from an API that will not answer
 * until somebody has signed in.
 *
 * It is shadcn-admin's second sign-in page without its picture: the form
 * alone, in the middle of the page.
 */
export function AuthLayout({
  title,
  description,
  children,
  noSite,
}: AuthLayoutProps) {
  return (
    // Kite: min-h-svh rather than h-svh, so a form taller than a landscape
    // phone scrolls instead of losing its top.
    <div className='relative container grid min-h-svh flex-col items-center justify-center'>
      <div className='lg:p-8'>
        <div className='mx-auto flex w-full flex-col justify-center space-y-2 py-8 sm:w-120 sm:p-8'>
          <div className='mb-4 flex items-center justify-center'>
            <KiteMark className='me-2 size-6' />
            <h1 className='text-xl font-medium'>Kite</h1>
          </div>
        </div>
        <div className='mx-auto flex w-full max-w-sm flex-col justify-center space-y-2'>
          <div className='flex flex-col space-y-2 text-start'>
            <h2 className='text-lg font-semibold tracking-tight'>{title}</h2>
            <p className='text-sm text-muted-foreground'>{description}</p>
          </div>
          {children}
          <AuthFooter noSite={noSite} />
        </div>
      </div>
    </div>
  )
}

/** AuthFooter is the way back out, and the language to read the form in. */
function AuthFooter({ noSite }: { noSite?: boolean }) {
  const { t } = useI18n()
  return (
    <div className='flex items-center justify-center gap-2 pt-2 text-xs text-muted-foreground'>
      {!noSite && (
        <>
          {/* The server sends the bare host to the site's home page. */}
          <a href='/' className='transition-colors hover:text-foreground'>
            {t('login.backToSite')}
          </a>
          <span aria-hidden>·</span>
        </>
      )}
      <LanguagePicker />
    </div>
  )
}
