import type { Settings } from "@/api/client";
import { useI18n, type Key } from "@/i18n";
import { useWritable } from "@/hooks/useContents";
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
  timezone: "site.timezone",
  keywords: "site.keywords",
  noindex: "site.noindex",
  page_size: "build.pageSize",
  feed_limit: "build.feedLimit",
  default_category: "content.defaultCategory",
  head_html: "site.headHTML",
  footer_html: "site.footerHTML",
};

// A setting left empty is sent as null, which takes it out of kite.yaml
// rather than leaving a key that says nothing. The default category is the
// exception: empty says a new post starts in none.
const read = ({ site, build, content }: Settings) => ({
  title: site.title,
  description: site.description || null,
  base_url: site.base_url,
  language: site.language ?? "",
  timezone: site.timezone || null,
  keywords: site.keywords?.length ? site.keywords : null,
  noindex: site.noindex ?? false,
  page_size: build.page_size || null,
  feed_limit: build.feed_limit || null,
  default_category: content.default_category,
  head_html: site.head_html || null,
  footer_html: site.footer_html || null,
});

type Values = ReturnType<typeof read>;

export function SiteSettings() {
  const { t } = useI18n();
  const form = useSettingsForm(read, (key) => paths[key]);
  const writable = useWritable();
  const values = form.values as Partial<Values>;
  const set = (key: keyof Values, value: unknown) => form.change({ ...values, [key]: value });
  const text = (key: keyof Values) => (value: string) => set(key, value === "" ? null : value);
  const count = (key: keyof Values) => (value: string) => set(key, value === "" ? null : Number(value));

  return (
    <ContentSection title={t("settings.site")} desc={t("settings.siteNote")} writes>
      <div>
        <FormProblem form={form} />
        {!form.loaded ? (
          <Skeleton className="h-72 w-full" />
        ) : (
          <fieldset disabled={!writable} className="grid gap-10">
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
                <Row
                  id="page_size"
                  label="settings.pageSize"
                  help="settings.pageSizeHelp"
                  note={pagedOtherwise(form.settings?.build.pagination, t)}
                >
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

            <Group title={t("settings.groupWriting")}>
              <Row id="default_category" label="settings.defaultCategory" help="settings.defaultCategoryHelp">
                <Input
                  id="default_category"
                  placeholder={t("settings.defaultCategoryNone")}
                  value={values.default_category ?? ""}
                  onChange={(event) => set("default_category", event.target.value)}
                />
              </Row>
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
          </fieldset>
        )}
        <LeaveGuard form={form} />
      </div>
    </ContentSection>
  );
}

const listings = ["home", "list", "term"] as const;

/**
 * pagedOtherwise names the listings that do not page by the page size, as
 * the theme or build.pagination in kite.yaml has them, so the number is not
 * taken to rule them too. It is null when every listing pages by it.
 */
function pagedOtherwise(
  pagination: Record<string, number> | undefined,
  t: ReturnType<typeof useI18n>["t"],
): string | null {
  const own = listings.filter((kind) => pagination?.[kind] !== undefined);
  if (!pagination || own.length === 0) return null;
  const items = own.map((kind) =>
    t("settings.pagedAs", {
      listing: t(`settings.paged.${kind}` as Key),
      size: pagination[kind] === 0 ? t("settings.pagedAll") : t("settings.pagedBy", { n: pagination[kind] }),
    }),
  );
  return t("settings.pagedOtherwise", { listings: items.join(t("settings.pagedSep")) });
}
