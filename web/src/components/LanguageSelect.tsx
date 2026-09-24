import { useMemo } from "react";

import { useI18n } from "@/i18n";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

/**
 * The languages offered for a site.
 *
 * A curated list rather than every tag that exists: the point of a list is
 * that it is short enough to read. A project already set to something not
 * here keeps it, so nobody is talked out of a language by a form.
 */
const offered = [
  "en", "en-GB", "zh-CN", "zh-TW", "zh-HK", "ja", "ko",
  "fr", "de", "es", "es-MX", "pt", "pt-BR", "it", "nl", "ru", "uk", "pl",
  "tr", "ar", "he", "fa", "hi", "bn", "id", "ms", "vi", "th",
  "sv", "da", "nb", "fi", "cs", "sk", "hu", "ro", "bg", "el", "hr", "ca",
];

interface Props {
  id?: string;
  value: string;
  onChange: (value: string) => void;
}

/**
 * A site's language is a BCP 47 tag, and it is not something to type.
 *
 * It sets the lang attribute on every page, the prefix in every localized
 * URL and how dates are written. A typo in it does not fail; it quietly
 * produces a site that says it is written in a language that does not exist,
 * and an empty one is worse still.
 */
export function LanguageSelect({ id, value, onChange }: Props) {
  const { locale } = useI18n();

  const options = useMemo(() => {
    // A tag the project already uses is always offered, so opening this form
    // can never be the thing that changes it.
    const tags = offered.includes(value) || !value ? offered : [value, ...offered];

    const inUI = new Intl.DisplayNames([locale], { type: "language" });
    return tags.map((tag) => {
      // Two names: what speakers of the language call it, and what the
      // operator's own language calls it. Either alone leaves somebody
      // reading a word they do not know.
      const endonym = new Intl.DisplayNames([tag], { type: "language" }).of(tag);
      const known = inUI.of(tag);
      return {
        tag,
        label: endonym === known ? (endonym ?? tag) : `${endonym} · ${known}`,
      };
    });
  }, [locale, value]);

  return (
    <Select value={value} onValueChange={(v) => v && onChange(v)}>
      <SelectTrigger id={id} className="w-full">
        <SelectValue />
      </SelectTrigger>
      <SelectContent className="max-h-72">
        <SelectGroup>
          {options.map((o) => (
            // Radix shows the chosen item's text in the trigger, so the tag
            // beside the name appears there too.
            <SelectItem key={o.tag} value={o.tag}>
              <span className="flex min-w-0 items-baseline gap-2">
                <span className="truncate">{o.label}</span>
                <span className="font-mono text-xs text-muted-foreground">{o.tag}</span>
              </span>
            </SelectItem>
          ))}
        </SelectGroup>
      </SelectContent>
    </Select>
  );
}
