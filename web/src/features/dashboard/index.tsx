import { Link } from "@tanstack/react-router";
import { ExternalLink, Plus } from "lucide-react";

import { useI18n, type Key } from "@/i18n";
import { useContentTypes, useSite } from "@/hooks/useContents";
import { useKindLabel } from "@/hooks/useKindLabel";
import { useSession } from "@/hooks/useSession";
import { longDay } from "@/lib/dates";
import { siteHome } from "@/lib/links";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { IndexProblems } from "@/components/IndexProblems";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { DeliveryCard, LatestPosts, PendingCard, TopTerms } from "./components/cards";
import { Overview } from "./components/overview";
import { StatCards } from "./components/stat-cards";

export function Dashboard() {
  const { t, locale } = useI18n();
  const site = useSite();
  const session = useSession();
  const types = useContentTypes();
  const kindLabel = useKindLabel();

  const kinds = types.data?.items.map((type) => type.kind) ?? ["post", "page"];
  const primary = kinds[0] ?? "post";
  const tagged = types.data?.items.find((type) => type.kind === primary)?.taxonomies ?? [];

  const hour = new Date().getHours();
  const part = hour < 5 ? "night" : hour < 12 ? "morning" : hour < 18 ? "afternoon" : "evening";
  const greeting = t(`dashboard.${part}` as Key);
  const name = session.data?.required ? session.data.user : undefined;

  return (
    <>
      <AppHeader />

      <Main>
        <div className="mb-4 flex flex-wrap items-end justify-between gap-2">
          <div>
            <h1 className="text-2xl font-bold tracking-tight">
              {name ? t("dashboard.greeting", { greeting, name }) : greeting}
            </h1>
            <p className="text-muted-foreground">
              {site.data?.title ? `${site.data.title} · ` : ""}
              {longDay(locale)}
            </p>
          </div>
          <div className="flex items-center gap-2">
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
          </div>
        </div>

        <div className="space-y-4">
          <IndexProblems />
          <StatCards primary={primary} kinds={kinds} />

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

          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            <DeliveryCard />
            <PendingCard kind={primary} />
            {tagged.length > 0 && (
              <TopTerms kind={primary} taxonomy={tagged.includes("tags") ? "tags" : tagged[0]} />
            )}
          </div>
        </div>
      </Main>
    </>
  );
}
