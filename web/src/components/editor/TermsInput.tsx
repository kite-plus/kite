import { useMemo, useState } from "react";
import { Plus, X } from "lucide-react";

import { useI18n } from "@/i18n";
import { useTerms } from "@/hooks/useContents";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";

interface Props {
  id?: string;
  label: string;
  taxonomy: string;
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
}

/**
 * The terms an item carries in one taxonomy.
 *
 * Terms the project already uses are offered, since a tag typed slightly
 * differently is a second tag. Anything else typed becomes a new one.
 */
export function TermsInput({ id, label, taxonomy, value, onChange, placeholder }: Props) {
  const { t } = useI18n();
  const terms = useTerms(taxonomy);
  const [open, setOpen] = useState(false);
  const [query, setQuery] = useState("");

  const typed = query.trim();
  const offered = useMemo(
    () => (terms.data?.items ?? []).filter((item) => !value.includes(item.term)),
    [terms.data, value],
  );
  const known = typed === "" || value.includes(typed) || offered.some((item) => item.term === typed);

  const add = (term: string) => {
    if (!value.includes(term)) onChange([...value, term]);
    setQuery("");
  };

  return (
    <div
      id={id}
      aria-label={label}
      className="flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-input px-2 py-1.5 shadow-xs"
    >
      {value.map((term) => (
        <Badge key={term} variant="secondary" className="gap-1 font-normal">
          {term}
          <button
            type="button"
            aria-label={t("editor.removeTerm", { term })}
            onClick={() => onChange(value.filter((each) => each !== term))}
            className="text-muted-foreground transition-colors hover:text-foreground"
          >
            <X className="size-3" />
          </button>
        </Badge>
      ))}
      <Popover
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) setQuery("");
        }}
      >
        <PopoverTrigger asChild>
          <Button variant="ghost" size="sm" className="h-6 px-2 font-normal text-muted-foreground">
            <Plus className="size-3.5" />
            {placeholder}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-60 p-0" align="start">
          <Command>
            <CommandInput value={query} onValueChange={setQuery} placeholder={placeholder} />
            <CommandList>
              <CommandEmpty>{t("palette.empty")}</CommandEmpty>
              <CommandGroup>
                {!known && (
                  <CommandItem value={`create:${typed}`} onSelect={() => add(typed)}>
                    <Plus />
                    {t("editor.createTerm", { term: typed })}
                  </CommandItem>
                )}
                {offered.map((item) => (
                  <CommandItem key={item.term} value={item.term} onSelect={() => add(item.term)}>
                    <span className="flex-1 truncate">{item.term}</span>
                    <span className="text-xs text-muted-foreground tabular-nums">{item.count}</span>
                  </CommandItem>
                ))}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  );
}
