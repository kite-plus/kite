import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";

import { api, unwrap, type Settings } from "@/api/client";
import { Alert } from "@/components/Alert";
import { Page } from "@/components/Shell";
import { SchemaForm } from "@/components/SchemaForm";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

/** sitePaths maps a field of SiteSettings to the config path that holds it. */
const sitePaths = {
  title: "site.title",
  description: "site.description",
  base_url: "site.baseURL",
  language: "site.language",
} as const;

const labels: Record<keyof typeof sitePaths, string> = {
  title: "Title",
  description: "Description",
  base_url: "Base URL",
  language: "Language",
};

/**
 * The site's settings, and the theme's.
 *
 * The theme's half is generated from the schema the theme declares, which is
 * the third place one renderer serves: an item's own fields, a theme's
 * settings, and a plugin's when those arrive. Writing a page per theme is the
 * reason configuring one is normally a development task.
 */
export function SettingsPage() {
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

  if (error && !settings) {
    return (
      <Page title="Settings">
        <Alert tone="stop" title="Could not load the settings">
          {error}
        </Alert>
      </Page>
    );
  }
  if (!settings) {
    return (
      <Page title="Settings">
        <p className="text-sm text-muted-foreground">Loading</p>
      </Page>
    );
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

  return (
    <Page
      title="Settings"
      description="Written to kite.yaml, leaving everything else in it alone."
      actions={
        <Button size="sm" disabled={!dirty || saving} onClick={save}>
          {saving ? "Saving" : "Save"}
        </Button>
      }
    >
      <div className="max-w-2xl space-y-4">
        {error && <Alert tone="stop">{error}</Alert>}

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Site</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {(Object.keys(sitePaths) as (keyof typeof sitePaths)[]).map((key) => (
              <div key={key} className="space-y-1.5">
                <Label htmlFor={key}>{labels[key]}</Label>
                <Input
                  id={key}
                  value={String(site[key] ?? "")}
                  onChange={(e) => {
                    setSite({ ...site, [key]: e.target.value });
                    setDirty(true);
                  }}
                />
              </div>
            ))}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Theme</CardTitle>
            <CardDescription>{settings.theme.name} declares these itself.</CardDescription>
          </CardHeader>
          <CardContent>
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
              <p className="text-sm text-muted-foreground">
                This theme declares no settings.
              </p>
            )}
          </CardContent>
        </Card>
      </div>
    </Page>
  );
}
