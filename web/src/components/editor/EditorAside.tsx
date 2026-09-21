import { Trash2 } from "lucide-react";

import type { ContentType, Draft } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { STATUSES } from "@/hooks/useContents";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { canPublish, type DeliveryState, type usePublish } from "@/hooks/usePublish";

import { SchemaForm, type Uploads } from "@/components/SchemaForm";
import { TermsInput } from "@/components/editor/TermsInput";
import { DeliveryStages, PublishProblems } from "@/components/publish/Delivery";
import { Button } from "@/components/ui/button";
import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldLegend,
  FieldSeparator,
  FieldSet,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { InputGroup, InputGroupAddon, InputGroupInput, InputGroupText } from "@/components/ui/input-group";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface Props {
  draft: Draft;
  type?: ContentType;
  onEdit: (patch: Partial<Draft>) => void;
  delivery?: DeliveryState;
  publish: ReturnType<typeof usePublish>;
  /** uploads is absent until the item is saved and has a folder of its own. */
  uploads?: Uploads;
  onDelete?: () => void;
}

/** Everything about an item that is not its text. */
export function EditorAside({ draft, type, onEdit, delivery, publish, uploads, onDelete }: Props) {
  const { t } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();

  // Categories lead, as they do in the listing.
  const taxonomies = [...(type?.taxonomies ?? [])].sort(
    (a, b) => Number(b === "categories") - Number(a === "categories"),
  );

  // "/posts/:slug" reads as "/posts/" in front of the field.
  const prefix = type?.route.includes(":slug") ? type.route.split(":slug")[0] : "/";

  return (
    <FieldGroup className="gap-6 p-4">
      <FieldSet className="gap-3">
        <FieldLegend variant="label">{t("publish.title")}</FieldLegend>
        <FieldGroup className="gap-2.5">
          <Field orientation="horizontal">
            <FieldLabel htmlFor="status" className="font-normal text-muted-foreground">
              {t("list.status")}
            </FieldLabel>
            <Select value={draft.status} onValueChange={(status) => status && onEdit({ status })}>
              <SelectTrigger id="status" size="sm" className="w-40">
                {/* The stored value is English; a person reads their own language. */}
                <SelectValue>{(status: string) => t(`status.${status}` as Key)}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectGroup>
                  {STATUSES.map((status) => (
                    <SelectItem key={status} value={status}>
                      {t(`status.${status}` as Key)}
                    </SelectItem>
                  ))}
                </SelectGroup>
              </SelectContent>
            </Select>
          </Field>

          <Field orientation="horizontal">
            <FieldLabel htmlFor="published_at" className="font-normal text-muted-foreground">
              {t("editor.publishedAt")}
            </FieldLabel>
            <Input
              id="published_at"
              type="datetime-local"
              className="h-7 w-40 px-2 text-xs md:text-xs"
              value={toLocalInput(draft.published_at)}
              onChange={(event) =>
                onEdit({
                  published_at: event.target.value
                    ? new Date(event.target.value).toISOString()
                    : undefined,
                })
              }
            />
          </Field>
        </FieldGroup>

        {canPublish(delivery) && (
          <>
            <FieldSeparator />
            <DeliveryStages delivery={delivery} />
          </>
        )}
        <PublishProblems publish={publish} />
      </FieldSet>

      {taxonomies.length > 0 && (
        <FieldSet className="gap-3">
          <FieldLegend variant="label">{t("editor.terms")}</FieldLegend>
          <FieldGroup className="gap-3">
            {taxonomies.map((taxonomy) => (
              <Field key={taxonomy}>
                <FieldLabel htmlFor={`terms-${taxonomy}`}>{taxonomyLabel(taxonomy)}</FieldLabel>
                <TermsInput
                  id={`terms-${taxonomy}`}
                  taxonomy={taxonomy}
                  value={draft.taxonomies?.[taxonomy] ?? []}
                  onChange={(terms) =>
                    onEdit({ taxonomies: { ...draft.taxonomies, [taxonomy]: terms } })
                  }
                />
              </Field>
            ))}
          </FieldGroup>
        </FieldSet>
      )}

      <FieldSet className="gap-3">
        <FieldLegend variant="label">{t("editor.details")}</FieldLegend>
        <FieldGroup className="gap-4">
          <Field>
            <FieldLabel htmlFor="slug">{t("editor.slug")}</FieldLabel>
            <InputGroup>
              <InputGroupAddon>
                <InputGroupText className="font-mono text-xs">{prefix}</InputGroupText>
              </InputGroupAddon>
              <InputGroupInput
                id="slug"
                className="font-mono text-xs md:text-xs"
                value={draft.slug ?? ""}
                onChange={(event) => onEdit({ slug: event.target.value })}
                placeholder={t("editor.slugPlaceholder")}
              />
            </InputGroup>
          </Field>

          {type?.fields && type.fields.length > 0 && (
            <SchemaForm
              fields={type.fields}
              values={draft.meta ?? {}}
              onChange={(meta) => onEdit({ meta })}
              uploads={uploads}
            />
          )}
        </FieldGroup>
      </FieldSet>

      {onDelete && (
        <Button variant="destructive" onClick={onDelete}>
          <Trash2 data-icon="inline-start" />
          {t("editor.delete")}
        </Button>
      )}
    </FieldGroup>
  );
}

/** toLocalInput formats an instant for a datetime-local control. */
function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const at = new Date(iso);
  const local = new Date(at.getTime() - at.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
