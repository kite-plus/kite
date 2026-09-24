import { Languages } from 'lucide-react'
import { locales, useI18n, type Locale } from '@/i18n'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

/**
 * The admin speaks the operator's language, which is not the site's: a
 * Chinese author may well publish in English, and the reverse.
 *
 * It is reachable from the sign-in page as well as from inside the studio,
 * because somebody who cannot read the form has no way to get past it.
 */
export function LanguagePicker() {
  const { locale, setLocale, t } = useI18n()

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant='ghost' size='sm' title={t('nav.language')}>
          <Languages />
          {locales[locale].label}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align='end'>
        <DropdownMenuRadioGroup
          value={locale}
          onValueChange={(next) => setLocale(next as Locale)}
        >
          {(Object.keys(locales) as Locale[]).map((code) => (
            <DropdownMenuRadioItem key={code} value={code}>
              {locales[code].label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
