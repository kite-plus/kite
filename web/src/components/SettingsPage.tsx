import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { api, unwrap, type Settings } from "@/api/client";
import { SchemaForm } from "@/components/SchemaForm";
import { Failure, Panel } from "@/components/ui";

/**
 * The site's settings, and the theme's.
 *
 * The theme's half is generated from the schema the theme declares, which is
 * the third place one renderer serves: an item's own fields, a theme's
 * settings, and a plugin's when those arrive. Writing a page per theme is the
 * reason configuring one is normally a development task.
 */
export function SettingsPage({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient();

  const [settings, setSettings] = useState<Settings | null>(null);
  const [site, setSite] = useState<Record<string, unknown>>({});
  const [theme, setTheme] = useState<Record<string, unknown>>({});
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [dirty, setDirty] = useState(false);

  useEffect(() => {
    (async () => {
      try {
        const data = unwrap(await api.GET("/settings", {}));
        setSettings(data);
        setSite({ ...data.site });
        setTheme({ ...(data.theme.values ?? {}) });
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, []);

  if (error && !settings) return <div className="p-6"><Failure error={error} /></div>;
  if (!settings) {
    return <div className="p-10 text-center text-sm text-[var(--muted-foreground)]">Loading</div>;
  }

  // Only what changed is sent, so a file is not rewritten for values nobody
  // touched.
  const changes = () => {
    const out: Record<string, unknown> = {};
    for (const [key, path] of Object.entries(sitePaths)) {
      const before = (settings.site as Record<string, unknown>)[key];
      if (site[key] !== before) out[path] = site[key] ?? "";
    }
    for (const [key, value] of Object.entries(theme)) {
      if (value !== (settings.theme.values ?? {})[key]) out[`theme.settings.${key}`] = value;
    }
    return out;
  };

  const save = async () => {
    setSaving(true);
    setError(null);
    const { data, error } = await api.PUT("/settings", { body: changes() });
    setSaving(false);
    if (error) {
      setError(error.error.message);
      return;
    }
    setSettings(data);
    setSite({ ...data.site });
    setTheme({ ...(data.theme.values ?? {}) });
    setDirty(false);
    await queryClient.invalidateQueries({ queryKey: ["site"] });
  };

  const field = (key: keyof typeof sitePaths, label: string) => (
    <label className="block">
      <span className="mb-1 block text-xs font-medium text-[var(--muted-foreground)]">{label}</span>
      <input
        value={String(site[key] ?? "")}
        onChange={(e) => {
          setSite({ ...site, [key]: e.target.value });
          setDirty(true);
        }}
        className="w-full rounded-md border border-[var(--border)] bg-[var(--background)] px-2 py-1.5 text-sm outline-none focus:border-brand focus:ring-2 focus:ring-brand/20"
      />
    </label>
  );

  return (
    <div className="mx-auto max-w-3xl px-4 py-8 sm:px-6">
      <header className="mb-6 flex items-center justify-between gap-3">
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-[var(--border)] px-3 py-1.5 text-sm hover:bg-[var(--accent)]"
          >
            Back
          </button>
          <button
            type="button"
            disabled={!dirty || saving}
            onClick={save}
            className="rounded-md bg-brand px-3 py-1.5 text-sm text-white disabled:opacity-40"
          >
            {saving ? "Saving" : "Save"}
          </button>
        </div>
      </header>

      {error && <div className="mb-4"><Failure error={error} /></div>}

      <Panel className="mb-6 p-5">
        <h2 className="mb-4 text-sm font-semibold">Site</h2>
        <div className="space-y-3">
          {field("title", "Title")}
          {field("description", "Description")}
          {field("base_url", "Base URL")}
          {field("language", "Language")}
        </div>
      </Panel>

      <Panel className="p-5">
        <h2 className="mb-1 text-sm font-semibold">Theme</h2>
        <p className="mb-4 text-xs text-[var(--muted-foreground)]">
          {settings.theme.name} declares these itself.
        </p>
        {settings.theme.schema && settings.theme.schema.length > 0 ? (
          <SchemaForm
            fields={settings.theme.schema}
            values={theme}
            onChange={(values) => {
              setTheme(values);
              setDirty(true);
            }}
          />
        ) : (
          <p className="text-sm text-[var(--muted-foreground)]">
            This theme declares no settings.
          </p>
        )}
      </Panel>
    </div>
  );
}

/** sitePaths maps a field of SiteSettings to the config path that holds it. */
const sitePaths = {
  title: "site.title",
  description: "site.description",
  base_url: "site.baseURL",
  language: "site.language",
} as const;
