import { Languages } from "lucide-react";

import { locales, useI18n, type Locale } from "@/i18n";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

/**
 * The admin speaks the operator's language, which is not the site's: a
 * Chinese author may well publish in English, and the reverse.
 *
 * It is reachable from the sign-in page as well as from inside the studio,
 * because somebody who cannot read the form has no way to get past it.
 */
export function LanguagePicker({ showLabel = false }: { showLabel?: boolean }) {
  const { locale, setLocale, t } = useI18n();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        render={
          <Button
            variant="ghost"
            size={showLabel ? "sm" : "icon-sm"}
            title={t("nav.language")}
          >
            <Languages data-icon="inline-start" />
            {showLabel && locales[locale].label}
          </Button>
        }
      />
      <DropdownMenuContent align="end">
        {/* A radio group rather than a list of commands: one of these is
            already true, and the menu should say which. */}
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
  );
}
