import { Link } from "@tanstack/react-router";
import { ExternalLink, Plus } from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { useContentTypes, useSite, useTaxonomies } from "@/hooks/useContents";
import { useKindLabel, useTaxonomyLabel } from "@/hooks/useKindLabel";
import { useSession } from "@/hooks/useSession";
import { siteHome } from "@/lib/links";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { RefreshButton } from "@/components/refresh-button";
import { DeliveryCard, LatestPosts, TopTerms } from "./components/cards";
import { Overview } from "./components/overview";
import { StatCards } from "./components/stat-cards";

export function Dashboard() {
  const { t } = useI18n();
  const site = useSite();
  const session = useSession();
  const types = useContentTypes();
  const taxonomies = useTaxonomies();
  const kindLabel = useKindLabel();
  const taxonomyLabel = useTaxonomyLabel();

  const kinds = types.data?.items.map((type) => type.kind) ?? ["post", "page"];
  const primary = kinds[0] ?? "post";
  const tagged = types.data?.items.find((type) => type.kind === primary)?.taxonomies ?? [];

  const hour = new Date().getHours();
  const part = hour < 5 ? "night" : hour < 12 ? "morning" : hour < 18 ? "afternoon" : "evening";
  const greeting = t(`dashboard.${part}` as Key);
  const name = session.data?.required ? session.data.user : undefined;

  // The site in figures, as explore's console heads its own: each kind, then
  // each taxonomy that sorts them.
  const counts = site.data?.counts;
  const figures = [
    site.data?.title,
    ...kinds.map((kind) =>
      counts?.[kind] !== undefined ? `${kindLabel.many(kind)} ${counts[kind]}` : undefined,
    ),
    ...(taxonomies.data?.items ?? []).map((item) => `${taxonomyLabel(item.name)} ${item.terms}`),
  ]
    .filter(Boolean)
    .join(" · ");

  return (
    <>
      <AppHeader />

      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle
          title={name ? t("dashboard.greeting", { greeting, name }) : greeting}
          description={figures || " "}
        >
          <RefreshButton />
          <Button variant="outline" asChild>
            <a href={siteHome(site.data)} target="_blank" rel="noreferrer">
              <ExternalLink />
              {t("nav.viewSite")}
            </a>
          </Button>
          <Button asChild>
            <Link to="/content/$kind/$id" params={{ kind: primary, id: "new" }}>
              <Plus />
              {t("dashboard.write", { kind: kindLabel.one(primary) })}
            </Link>
          </Button>
        </PageTitle>

        <StatCards kind={primary} />

        <div className="grid grid-cols-1 gap-4 lg:grid-cols-7">
          <Card className="col-span-1 lg:col-span-4">
            <CardHeader>
              <CardTitle>{t("dashboard.overview")}</CardTitle>
              <CardDescription>
                {t("dashboard.overviewNote", { kind: kindLabel.many(primary) })}
              </CardDescription>
            </CardHeader>
            <CardContent className="ps-2">
              <Overview kind={primary} />
            </CardContent>
          </Card>
          <LatestPosts kind={primary} />
        </div>

        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          <DeliveryCard />
          {tagged.length > 0 && (
            <TopTerms kind={primary} taxonomy={tagged.includes("tags") ? "tags" : tagged[0]} />
          )}
        </div>
      </Main>
    </>
  );
}
