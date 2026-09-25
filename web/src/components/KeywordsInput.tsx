import { useState, type ClipboardEvent, type KeyboardEvent } from "react";
import { X } from "lucide-react";

import { useI18n } from "@/i18n";
import { Badge } from "@/components/ui/badge";

interface Props {
  id?: string;
  value: string[];
  onChange: (value: string[]) => void;
  placeholder?: string;
}

// Commas in either script, and the enumeration comma, separate keywords.
const separators = /[,，、]/;

/** KeywordsInput edits a list of words, each shown as a chip. */
export function KeywordsInput({ id, value, onChange, placeholder }: Props) {
  const { t } = useI18n();
  const [typed, setTyped] = useState("");

  const add = (text: string) => {
    const words = text
      .split(separators)
      .map((word) => word.trim())
      .filter((word) => word !== "");
    const next = [...value];
    for (const word of words) if (!next.includes(word)) next.push(word);
    if (next.length !== value.length) onChange(next);
    setTyped("");
  };

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    // A word being composed with an input method is not finished yet.
    if (event.nativeEvent.isComposing) return;
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      add(typed);
    } else if (event.key === "Backspace" && typed === "" && value.length > 0) {
      onChange(value.slice(0, -1));
    }
  };

  const onPaste = (event: ClipboardEvent<HTMLInputElement>) => {
    const text = event.clipboardData.getData("text");
    if (!separators.test(text)) return;
    event.preventDefault();
    add(typed + text);
  };

  return (
    <div className="flex min-h-9 flex-wrap items-center gap-1.5 rounded-md border border-input px-2 py-1.5 shadow-xs focus-within:border-ring focus-within:ring-[3px] focus-within:ring-ring/50">
      {value.map((word) => (
        <Badge key={word} variant="secondary" className="gap-1 font-normal">
          {word}
          <button
            type="button"
            aria-label={t("form.removeKeyword", { word })}
            onClick={() => onChange(value.filter((each) => each !== word))}
            className="text-muted-foreground transition-colors hover:text-foreground"
          >
            <X className="size-3" />
          </button>
        </Badge>
      ))}
      <input
        id={id}
        className="h-6 min-w-24 flex-1 bg-transparent px-1 text-sm outline-none placeholder:text-muted-foreground"
        value={typed}
        placeholder={value.length === 0 ? placeholder : undefined}
        onChange={(event) => {
          const text = event.target.value;
          if (separators.test(text)) add(text);
          else setTyped(text);
        }}
        onKeyDown={onKeyDown}
        onPaste={onPaste}
        onBlur={() => typed.trim() && add(typed)}
      />
    </div>
  );
}
