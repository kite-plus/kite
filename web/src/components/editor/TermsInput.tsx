import { Fragment, useMemo, useState } from "react";
import { Plus } from "lucide-react";

import { useI18n } from "@/i18n";
import { useTerms } from "@/hooks/useContents";
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
  taxonomy: string;
  value: string[];
  onChange: (value: string[]) => void;
}

/**
 * The terms an item carries in one taxonomy.
 *
 * Terms the project already uses are offered, since a tag typed slightly
 * differently is a second tag. Anything else typed becomes a new one.
 */
export function TermsInput({ id, taxonomy, value, onChange }: Props) {
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
      <ComboboxChips ref={anchor}>
        <ComboboxValue>
          {(chosen: string[]) => (
            <Fragment>
              {chosen.map((term) => (
                <ComboboxChip key={term}>{term}</ComboboxChip>
              ))}
              <ComboboxChipsInput
                id={id}
                placeholder={chosen.length === 0 ? t("editor.addTerm") : ""}
              />
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
