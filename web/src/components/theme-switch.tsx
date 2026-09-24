import { Monitor, Moon, Sun } from 'lucide-react'
import { useI18n } from '@/i18n'
import { useTheme } from '@/lib/theme'
import { Button } from '@/components/ui/button'

// Kite: explore's toggle, one click to the next mode instead of a menu.
const modes = {
  system: { icon: Monitor, label: 'mode.system', next: 'dark' },
  dark: { icon: Moon, label: 'mode.dark', next: 'light' },
  light: { icon: Sun, label: 'mode.light', next: 'system' },
} as const

/** ThemeSwitch cycles automatic, dark and light mode, showing the current one. */
export function ThemeSwitch() {
  const { t } = useI18n()
  const { theme, setTheme } = useTheme()
  const { icon: Icon, label, next } = modes[theme]
  const hint = t('mode.cycle', { current: t(label), next: t(modes[next].label) })

  return (
    <Button
      variant='ghost'
      size='icon'
      className='scale-95 rounded-full'
      aria-label={hint}
      title={hint}
      onClick={() => setTheme(next)}
    >
      <Icon className='size-[1.2rem]' />
    </Button>
  )
}
