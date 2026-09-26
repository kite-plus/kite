import type { Settings } from "@/api/client";
import { useI18n } from "@/i18n";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { CodeField } from "@/components/CodeField";
import { KeywordsInput } from "@/components/KeywordsInput";
import { LanguageSelect } from "@/components/LanguageSelect";
import { TimezoneSelect } from "@/components/TimezoneSelect";
import { ContentSection } from "../components/content-section";
import { Group, Row } from "../components/form-group";
import { FormProblem, LeaveGuard, SaveButton, useSettingsForm } from "../use-settings-form";

/** paths maps a field of the form to the config path that holds it. */
const paths: Record<string, string> = {
  title: "site.title",
  description: "site.description",
  base_url: "site.baseURL",
  language: "site.language",
  author: "site.author",
  timezone: "site.timezone",
  keywords: "site.keywords",
  noindex: "site.noindex",
  page_size: "build.pageSize",
  feed_limit: "build.feedLimit",
  head_html: "site.headHTML",
  footer_html: "site.footerHTML",
};

// A setting left empty is sent as null, which takes it out of kite.yaml
// rather than leaving a key that says nothing.
const read = ({ site, build }: Settings) => ({
  title: site.title,
  description: site.description || null,
  base_url: site.base_url,
  language: site.language ?? "",
  author: site.author || null,
  timezone: site.timezone || null,
  keywords: site.keywords?.length ? site.keywords : null,
  noindex: site.noindex ?? false,
  page_size: build.page_size || null,
  feed_limit: build.feed_limit || null,
  head_html: site.head_html || null,
  footer_html: site.footer_html || null,
});

type Values = ReturnType<typeof read>;

export function SiteSettings() {
  const { t } = useI18n();
  const form = useSettingsForm(read, (key) => paths[key]);
  const values = form.values as Partial<Values>;
  const set = (key: keyof Values, value: unknown) => form.change({ ...values, [key]: value });
  const text = (key: keyof Values) => (value: string) => set(key, value === "" ? null : value);
  const count = (key: keyof Values) => (value: string) => set(key, value === "" ? null : Number(value));

  return (
    <ContentSection title={t("settings.site")} desc={t("settings.siteNote")}>
      <div>
        <FormProblem form={form} />
        {!form.loaded ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <div className="grid gap-10">
            <Group title={t("settings.groupBasics")}>
              <Row id="title" label="settings.siteTitle">
                <Input id="title" value={values.title ?? ""} onChange={(event) => set("title", event.target.value)} />
              </Row>
              <Row id="description" label="settings.siteDescription" help="settings.siteDescriptionHelp">
                <Textarea
                  id="description"
                  rows={3}
                  value={values.description ?? ""}
                  onChange={(event) => text("description")(event.target.value)}
                />
              </Row>
              <Row id="base_url" label="settings.siteBaseURL" help="settings.siteBaseURLHelp">
                <Input
                  id="base_url"
                  type="url"
                  spellCheck={false}
                  value={values.base_url ?? ""}
                  onChange={(event) => set("base_url", event.target.value)}
                />
              </Row>
              <Row id="language" label="settings.siteLanguage">
                <LanguageSelect id="language" value={values.language ?? ""} onChange={(v) => set("language", v)} />
              </Row>
              <Row id="author" label="settings.siteAuthor" help="settings.siteAuthorHelp">
                <Input id="author" value={values.author ?? ""} onChange={(event) => text("author")(event.target.value)} />
              </Row>
              <Row id="timezone" label="settings.siteTimezone" help="settings.siteTimezoneHelp">
                <TimezoneSelect id="timezone" value={values.timezone ?? null} onChange={(v) => set("timezone", v)} />
              </Row>
            </Group>

            <Group title={t("settings.groupSearch")}>
              <Row id="keywords" label="settings.siteKeywords" help="settings.siteKeywordsHelp">
                <KeywordsInput
                  id="keywords"
                  value={values.keywords ?? []}
                  placeholder={t("settings.siteKeywordsPlaceholder")}
                  onChange={(words) => set("keywords", words.length ? words : null)}
                />
              </Row>
              <div className="flex items-start justify-between gap-6">
                <div className="grid gap-1">
                  <Label htmlFor="indexed">{t("settings.siteIndexed")}</Label>
                  <p className="text-sm text-muted-foreground">{t("settings.siteIndexedHelp")}</p>
                </div>
                <Switch
                  id="indexed"
                  checked={!values.noindex}
                  onCheckedChange={(indexed) => set("noindex", !indexed)}
                />
              </div>
            </Group>

            <Group title={t("settings.groupReading")}>
              <div className="grid gap-6 sm:grid-cols-2">
                <Row id="page_size" label="settings.pageSize" help="settings.pageSizeHelp">
                  <Input
                    id="page_size"
                    type="number"
                    min={1}
                    max={100}
                    placeholder="10"
                    value={values.page_size ?? ""}
                    onChange={(event) => count("page_size")(event.target.value)}
                  />
                </Row>
                <Row id="feed_limit" label="settings.feedLimit" help="settings.feedLimitHelp">
                  <Input
                    id="feed_limit"
                    type="number"
                    min={1}
                    max={1000}
                    placeholder="20"
                    value={values.feed_limit ?? ""}
                    onChange={(event) => count("feed_limit")(event.target.value)}
                  />
                </Row>
              </div>
            </Group>

            <Group title={t("settings.groupCode")} note={t("settings.groupCodeNote")}>
              <Row id="head_html" label="settings.headHTML" help="settings.headHTMLHelp">
                <CodeField
                  id="head_html"
                  language="html"
                  value={values.head_html ?? ""}
                  placeholder={'<script async src="https://analytics.example.com/script.js"></script>'}
                  onChange={(value) => text("head_html")(value)}
                />
              </Row>
              <Row id="footer_html" label="settings.footerHTML" help="settings.footerHTMLHelp">
                <CodeField
                  id="footer_html"
                  language="html"
                  value={values.footer_html ?? ""}
                  onChange={(value) => text("footer_html")(value)}
                />
              </Row>
            </Group>

            <div>
              <SaveButton form={form} />
            </div>
          </div>
        )}
        <LeaveGuard form={form} />
      </div>
    </ContentSection>
  );
}
