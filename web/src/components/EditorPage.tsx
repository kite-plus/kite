import { useCallback, useRef, useState } from "react";
import { ArrowLeft, Trash2 } from "lucide-react";

import { useContentTypes } from "@/hooks/useContents";
import { useItem } from "@/hooks/useItem";
import { Alert } from "@/components/Alert";
import { ConflictDialog } from "@/components/ConflictDialog";
import { Editor } from "@/components/Editor";
import { Preview } from "@/components/Preview";
import { PublishPanel } from "@/components/PublishPanel";
import { SchemaForm } from "@/components/SchemaForm";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";

interface Props {
  id: string | null;
  kind: string;
  onClose: () => void;
  onCreated: (id: string) => void;
}

const statuses = ["draft", "published", "scheduled", "archived"];

export function EditorPage({ id, kind, onClose, onCreated }: Props) {
  const types = useContentTypes();
  const item = useItem(id, kind);
  const [uploading, setUploading] = useState(0);
  const [uploadError, setUploadError] = useState<string | null>(null);

  const insert = useRef<(text: string) => void>(null);
  const type = types.data?.items.find((t) => t.kind === (item.draft?.kind ?? kind));

  const drop = useCallback(
    async (files: File[]) => {
      setUploadError(null);
      setUploading((n) => n + files.length);
      try {
        for (const file of files) {
          const link = await item.attach(file);
          const alt = file.name.replace(/\.[^.]+$/, "");
          insert.current?.(`\n![${alt}](${link})\n`);
        }
      } catch (err) {
        setUploadError(err instanceof Error ? err.message : String(err));
      } finally {
        setUploading(0);
      }
    },
    [item],
  );

  if (item.status === "loading") {
    return (
      <div className="p-10 text-center text-sm text-muted-foreground">Loading</div>
    );
  }
  if (!item.draft) {
    return (
      <div className="p-6">
        <Alert tone="stop" title="Nothing to edit">
          {item.error}
        </Alert>
      </div>
    );
  }

  const draft = item.draft;
  const state =
    uploading > 0
      ? `uploading ${uploading}`
      : item.status === "saving"
        ? "saving"
        : item.dirty
          ? "unsaved"
          : "saved";

  return (
    <div className="flex h-svh flex-col bg-background">
      <header className="flex min-h-14 flex-wrap items-center gap-2 border-b px-3 py-2 sm:px-4">
        <Button variant="ghost" size="icon" onClick={onClose} title="Back">
          <ArrowLeft className="size-4" />
        </Button>

        <Input
          value={draft.title}
          onChange={(e) => item.edit({ title: e.target.value })}
          placeholder="Title"
          className="min-w-40 flex-1 border-0 bg-transparent px-1 text-base font-semibold shadow-none focus-visible:ring-0 md:text-base"
        />

        <Select
          value={draft.status}
          onValueChange={(v) => v && item.edit({ status: v })}
        >
          <SelectTrigger size="sm" className="w-32">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {statuses.map((s) => (
              <SelectItem key={s} value={s}>
                {s}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <span className="w-20 text-right text-xs text-muted-foreground">{state}</span>

        <Button
          size="sm"
          disabled={!item.dirty || item.status === "saving"}
          onClick={async () => {
            const saved = await item.save();
            // A new item has no id until the server gives it one, and the
            // editor needs it before a file can be attached to it.
            if (saved && !id) onCreated(saved);
          }}
        >
          Save
        </Button>
      </header>

      {(item.error || uploadError) && (
        <div className="border-b px-4 py-2">
          <Alert tone="stop">{item.error ?? uploadError}</Alert>
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        <aside className="hidden w-72 shrink-0 overflow-auto border-r p-4 lg:block">
          <div className="space-y-3">
            <Field label="Slug">
              <Input
                value={draft.slug ?? ""}
                onChange={(e) => item.edit({ slug: e.target.value })}
                placeholder="derived from the title"
              />
            </Field>

            <Field label="Published">
              <Input
                type="datetime-local"
                value={toLocalInput(draft.published_at)}
                onChange={(e) =>
                  item.edit({
                    published_at: e.target.value
                      ? new Date(e.target.value).toISOString()
                      : undefined,
                  })
                }
              />
            </Field>

            {type?.taxonomies?.map((taxonomy) => (
              <Field key={taxonomy} label={taxonomy}>
                <Input
                  value={(draft.taxonomies?.[taxonomy] ?? []).join(", ")}
                  onChange={(e) =>
                    item.edit({
                      taxonomies: {
                        ...draft.taxonomies,
                        [taxonomy]: e.target.value
                          .split(",")
                          .map((s) => s.trim())
                          .filter(Boolean),
                      },
                    })
                  }
                  placeholder="comma separated"
                />
              </Field>
            ))}
          </div>

          {type?.fields && type.fields.length > 0 && (
            <>
              <Separator className="my-4" />
              <SchemaForm
                fields={type.fields}
                values={draft.meta ?? {}}
                onChange={(meta) => item.edit({ meta })}
              />
            </>
          )}

          {id && (
            <>
              <Separator className="my-4" />
              <PublishPanel ids={[id]} />
              <Separator className="my-4" />
              <Button
                variant="ghost"
                size="sm"
                className="w-full text-destructive hover:bg-destructive/10 hover:text-destructive"
                onClick={async () => {
                  if (confirm("Delete this item and everything in its folder?")) {
                    if (await item.remove()) onClose();
                  }
                }}
              >
                <Trash2 className="size-4" />
                Delete
              </Button>
            </>
          )}
        </aside>

        <div className="min-w-0 flex-1 border-r">
          <Editor
            value={draft.body}
            onChange={(body) => item.edit({ body })}
            onDropFiles={id ? drop : undefined}
            onReady={(fn) => {
              insert.current = fn;
            }}
          />
        </div>

        <div className="hidden min-w-0 flex-1 md:block">
          <Preview draft={draft} id={id} />
        </div>
      </div>

      {item.conflict && (
        <ConflictDialog
          conflict={item.conflict}
          ours={draft}
          onTakeTheirs={item.takeTheirs}
          onKeepOurs={item.keepOurs}
          onCancel={() => item.edit({})}
        />
      )}
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="space-y-1.5">
      <Label className="text-xs font-medium text-muted-foreground">{label}</Label>
      {children}
    </div>
  );
}

/** toLocalInput formats an instant for a datetime-local control. */
function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const at = new Date(iso);
  const local = new Date(at.getTime() - at.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
