import { Moon, Sun, Monitor } from 'lucide-react'
import { useTheme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'
import { useI18n } from '@/i18n'

type Props = {
  className?: string
}

export function ThemeToggle({ className }: Props) {
  const { theme, setTheme } = useTheme()
  const { t } = useI18n()
  return (
    <Button
      variant="ghost"
      size="icon-sm"
      onClick={() => {
        if (theme === 'light') setTheme('dark')
        else if (theme === 'dark') setTheme('system')
        else setTheme('light')
      }}
      // The aria-label flips with locale so screen-reader users don't
      // get a Chinese hint while reading an English UI.
      aria-label={t('auth.themeToggleAria')}
      className={cn('rounded-full', className)}
    >
      {theme === 'light' && <Sun className="size-4" />}
      {theme === 'dark' && <Moon className="size-4" />}
      {theme === 'system' && <Monitor className="size-4" />}
      <span className="sr-only">{t('auth.themeToggleAria')}</span>
    </Button>
  )
}
