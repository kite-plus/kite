import type { ReactNode } from "react";

import { useI18n, type Key } from "@/i18n";
import { Label } from "@/components/ui/label";

/** Group is a titled part of a settings form, drawn as theme settings draw a section. */
export function Group({
  id,
  title,
  note,
  children,
}: {
  id?: string;
  title: string;
  note?: string;
  children: ReactNode;
}) {
  return (
    <fieldset id={id} className="grid scroll-mt-4 gap-6">
      <legend className="mb-6 w-full border-b pb-2">
        <span className="text-sm font-semibold">{title}</span>
        {note && <p className="mt-0.5 text-sm text-muted-foreground">{note}</p>}
      </legend>
      {children}
    </fieldset>
  );
}

/** Row is one labelled field, with a line of help under it when it needs one. */
export function Row({ id, label, help, children }: { id: string; label: Key; help?: Key; children: ReactNode }) {
  const { t } = useI18n();
  return (
    <div className="grid gap-2">
      <Label htmlFor={id}>{t(label)}</Label>
      {children}
      {help && <p className="text-sm text-muted-foreground">{t(help)}</p>}
    </div>
  );
}
