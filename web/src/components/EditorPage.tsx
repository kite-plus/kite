import { useCallback, useRef, useState } from "react";

import { useContentTypes } from "@/hooks/useContents";
import { useItem } from "@/hooks/useItem";
import { ConflictDialog } from "@/components/ConflictDialog";
import { Editor } from "@/components/Editor";
import { Preview } from "@/components/Preview";
import { SchemaForm } from "@/components/SchemaForm";
import { Failure, Select } from "@/components/ui";
import { cn } from "@/lib/cn";

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
    return <div className="p-10 text-center text-sm text-[var(--muted-foreground)]">Loading</div>;
  }
  if (!item.draft) {
    return (
      <div className="p-6">
        <Failure error={item.error ?? "nothing to edit"} />
      </div>
    );
  }

  const draft = item.draft;

  return (
    <div className="flex h-dvh flex-col">
      <header className="flex flex-wrap items-center gap-3 border-b border-[var(--border)] px-4 py-2.5">
        <button
          type="button"
          onClick={onClose}
          className="rounded-md border border-[var(--border)] px-2 py-1 text-sm hover:bg-[var(--accent)]"
        >
          Back
        </button>

        <input
          value={draft.title}
          onChange={(e) => item.edit({ title: e.target.value })}
          placeholder="Title"
          className="min-w-40 flex-1 bg-transparent text-lg font-semibold outline-none placeholder:text-[var(--muted-foreground)]"
        />

        <Select
          label="Status"
          value={draft.status}
          onChange={(e) => item.edit({ status: e.target.value })}
        >
          {statuses.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </Select>

        <span className="text-xs text-[var(--muted-foreground)]">
          {uploading > 0
            ? `uploading ${uploading}`
            : item.status === "saving"
              ? "saving"
              : item.dirty
                ? "unsaved"
                : "saved"}
        </span>

        <button
          type="button"
          disabled={!item.dirty || item.status === "saving"}
          onClick={async () => {
            const saved = await item.save();
            // A new item has no id until the server gives it one, and the
            // editor needs it before a file can be attached to it.
            if (saved && !id) onCreated(saved);
          }}
          className="rounded-md bg-brand px-3 py-1.5 text-sm text-white disabled:opacity-40"
        >
          Save
        </button>
      </header>

      {(item.error || uploadError) && (
        <div className="border-b border-[var(--border)] bg-red-500/5 px-4 py-2 text-sm text-red-700 dark:text-red-400">
          {item.error ?? uploadError}
        </div>
      )}

      <div className="flex min-h-0 flex-1">
        <aside className="hidden w-64 shrink-0 overflow-auto border-r border-[var(--border)] p-4 lg:block">
          <Field label="Slug">
            <input
              value={draft.slug ?? ""}
              onChange={(e) => item.edit({ slug: e.target.value })}
              placeholder="derived from the title"
              className="w-full rounded-md border border-[var(--border)] bg-[var(--background)] px-2 py-1.5 text-sm outline-none focus:border-brand"
            />
          </Field>

          <Field label="Published">
            <input
              type="datetime-local"
              value={toLocalInput(draft.published_at)}
              onChange={(e) =>
                item.edit({
                  published_at: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                })
              }
              className="w-full rounded-md border border-[var(--border)] bg-[var(--background)] px-2 py-1.5 text-sm outline-none focus:border-brand"
            />
          </Field>

          {type?.taxonomies?.map((taxonomy) => (
            <Field key={taxonomy} label={taxonomy}>
              <input
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
                className="w-full rounded-md border border-[var(--border)] bg-[var(--background)] px-2 py-1.5 text-sm outline-none focus:border-brand"
              />
            </Field>
          ))}

          {type?.fields && type.fields.length > 0 && (
            <div className="mt-5 border-t border-[var(--border)] pt-4">
              <SchemaForm
                fields={type.fields}
                values={draft.meta ?? {}}
                onChange={(meta) => item.edit({ meta })}
              />
            </div>
          )}

          {id && (
            <button
              type="button"
              onClick={async () => {
                if (confirm("Delete this item and everything in its folder?")) {
                  if (await item.remove()) onClose();
                }
              }}
              className="mt-6 w-full rounded-md border border-red-500/30 px-2 py-1.5 text-sm text-red-600 hover:bg-red-500/5 dark:text-red-400"
            >
              Delete
            </button>
          )}
        </aside>

        <div className={cn("min-w-0 flex-1 border-r border-[var(--border)]")}>
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
    <label className="mb-3 block">
      <span className="mb-1 block text-xs font-medium text-[var(--muted-foreground)]">{label}</span>
      {children}
    </label>
  );
}

/** toLocalInput formats an instant for a datetime-local control. */
function toLocalInput(iso?: string): string {
  if (!iso) return "";
  const at = new Date(iso);
  const local = new Date(at.getTime() - at.getTimezoneOffset() * 60_000);
  return local.toISOString().slice(0, 16);
}
