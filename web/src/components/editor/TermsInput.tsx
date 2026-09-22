import { Fragment, useMemo, useState } from "react";
import { Combobox as ComboboxPrimitive } from "@base-ui/react";
import { Plus } from "lucide-react";
import { cn } from "cn";

import { useI18n } from "@/i18n";
import { useTerms } from "@/hooks/useContents";
import { IconChevronDown } from "@/components/icons";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxItem,
  ComboboxList,
  ComboboxValue,
  useComboboxAnchor,
} from "@/components/ui/combobox";

interface Props {
  id?: string;
  /** label names the field for a screen reader, since the design shows none. */
  label: string;
  taxonomy: string;
  value: string[];
  onChange: (value: string[]) => void;
  /**
   * select draws the field as the design draws a category: a dropdown box
   * with the chosen terms as plain words. chips draws it as its tag box.
   */
  variant?: "chips" | "select";
  placeholder?: string;
}

/**
 * The terms an item carries in one taxonomy.
 *
 * Terms the project already uses are offered, since a tag typed slightly
 * differently is a second tag. Anything else typed becomes a new one.
 */
export function TermsInput({
  id,
  label,
  taxonomy,
  value,
  onChange,
  variant = "chips",
  placeholder,
}: Props) {
  const { t } = useI18n();
  const anchor = useComboboxAnchor();
  const terms = useTerms(taxonomy);
  const [query, setQuery] = useState("");

  const typed = query.trim();
  const known = useMemo(() => {
    const names = new Set(terms.data?.items.map((item) => item.term));
    for (const term of value) names.add(term);
    return names;
  }, [terms.data, value]);

  const items = useMemo(
    () => (typed && !known.has(typed) ? [...known, typed] : [...known]),
    [known, typed],
  );
  const select = variant === "select";

  return (
    <Combobox
      multiple
      autoHighlight
      items={items}
      value={value}
      onValueChange={(next: string[], details) => {
        // Escape on a closed list would empty the field. Here it is on its
        // way to closing whatever holds the form, and must not take the
        // terms with it.
        if (details.reason === "escape-key") {
          details.cancel();
          details.allowPropagation();
          return;
        }
        onChange(next);
        setQuery("");
      }}
      inputValue={query}
      onInputValueChange={setQuery}
    >
      <ComboboxChips
        ref={anchor}
        className={cn(
          "rounded-[7px] border-input bg-background text-xs",
          select
            ? "h-[30px] min-h-0 flex-nowrap gap-0 px-2.5 py-0 has-data-[slot=combobox-chip]:px-2.5"
            : "min-h-9 gap-1.5 px-2 py-[5px] has-data-[slot=combobox-chip]:px-2",
        )}
      >
        <ComboboxValue>
          {(chosen: string[]) => (
            <Fragment>
              {chosen.map((term, i) =>
                select ? (
                  <ComboboxChip
                    key={term}
                    showRemove={false}
                    className="h-auto shrink-0 rounded-none bg-transparent p-0 text-xs font-normal"
                  >
                    {i < chosen.length - 1 ? `${term}、` : term}
                  </ComboboxChip>
                ) : (
                  <ComboboxChip
                    key={term}
                    showRemove={false}
                    className="h-auto gap-[5px] rounded-[5px] bg-muted px-[7px] py-0.5 text-[11.5px] font-normal text-foreground-2"
                  >
                    {term}
                    <ComboboxPrimitive.ChipRemove
                      aria-label={t("editor.removeTerm", { term })}
                      className="cursor-pointer text-subtle transition-colors hover:text-foreground"
                    >
                      ×
                    </ComboboxPrimitive.ChipRemove>
                  </ComboboxChip>
                ),
              )}
              <ComboboxChipsInput
                id={id}
                aria-label={label}
                placeholder={select && chosen.length > 0 ? "" : placeholder}
                className="min-w-[60px] bg-transparent text-xs placeholder:text-subtle"
              />
              {select && (
                <ComboboxPrimitive.Trigger
                  aria-label={label}
                  className="ml-auto flex shrink-0 items-center text-muted-foreground outline-none"
                >
                  <IconChevronDown className="size-3" />
                </ComboboxPrimitive.Trigger>
              )}
            </Fragment>
          )}
        </ComboboxValue>
      </ComboboxChips>
      <ComboboxContent anchor={anchor}>
        <ComboboxList>
          {(term: string) => (
            <ComboboxItem key={term} value={term}>
              {known.has(term) ? (
                term
              ) : (
                <>
                  <Plus />
                  {t("editor.createTerm", { term })}
                </>
              )}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
