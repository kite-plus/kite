import { Link } from "@tanstack/react-router";
import { ExternalLink } from "lucide-react";

import type { Taxonomy } from "@/api/client";
import { useI18n } from "@/i18n";
import { useContentTypes, useTaxonomies, useTerms } from "@/hooks/useContents";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardAction,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";
import { AppHeader } from "@/components/layout/app-header";
import { Main } from "@/components/layout/main";
import { PageTitle } from "@/components/layout/page-title";
import { QueryError } from "@/components/query-error";

export function Taxonomies() {
  const { t } = useI18n();
  const taxonomies = useTaxonomies();

  return (
    <>
      <AppHeader />
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
        <PageTitle title={t("taxonomies.title")} description={t("taxonomies.description")} />

        {taxonomies.error && !taxonomies.data ? (
          <QueryError error={taxonomies.error} onRetry={() => void taxonomies.refetch()} />
        ) : taxonomies.data?.items.length === 0 ? (
          <div className="rounded-md border p-10 text-center text-muted-foreground">
            {t("taxonomies.empty")}
          </div>
        ) : (
          <div className="grid gap-4 xl:grid-cols-2">
            {taxonomies.isPending
              ? Array.from({ length: 2 }, (_, i) => <Skeleton key={i} className="h-40" />)
              : taxonomies.data?.items.map((item) => <TaxonomyCard key={item.name} taxonomy={item} />)}
          </div>
        )}
      </Main>
    </>
  );
}

function TaxonomyCard({ taxonomy }: { taxonomy: Taxonomy }) {
  const { t } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();
  const types = useContentTypes();
  const terms = useTerms(taxonomy.name);

  // A term opens the listing of whichever kind carries this taxonomy.
  const kind = types.data?.items.find((type) => type.taxonomies?.includes(taxonomy.name))?.kind;

  return (
    <Card>
      <CardHeader>
        <CardTitle>{taxonomyLabel(taxonomy.name)}</CardTitle>
        <CardDescription>{t("taxonomies.terms", { count: taxonomy.terms })}</CardDescription>
        <CardAction>
          <Button variant="ghost" size="icon" asChild>
            <a href={taxonomy.url} target="_blank" rel="noreferrer" aria-label={t("list.open")}>
              <ExternalLink />
            </a>
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-wrap gap-2">
        {!terms.data
          ? Array.from({ length: 5 }, (_, i) => <Skeleton key={i} className="h-5 w-16" />)
          : terms.data.items.map((item) =>
              kind ? (
                <Link
                  key={item.term}
                  to="/content/$kind"
                  params={{ kind }}
                  search={{ terms: [`${taxonomy.name}:${item.term}`] }}
                >
                  <Badge variant="secondary" className="gap-1.5 font-normal hover:bg-secondary/70">
                    {item.term}
                    <span className="text-muted-foreground tabular-nums">{item.count}</span>
                  </Badge>
                </Link>
              ) : (
                <Badge key={item.term} variant="secondary" className="gap-1.5 font-normal">
                  {item.term}
                  <span className="text-muted-foreground tabular-nums">{item.count}</span>
                </Badge>
              ),
            )}
      </CardContent>
    </Card>
  );
}
