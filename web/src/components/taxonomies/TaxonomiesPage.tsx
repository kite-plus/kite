import { ExternalLink } from "lucide-react";

import type { Taxonomy } from "@/api/client";
import { useI18n } from "@/i18n";
import { useContentTypes, useTaxonomies, useTerms } from "@/hooks/useContents";
import { useTaxonomyLabel } from "@/hooks/useKindLabel";
import { setListState } from "@/lib/listState";
import { navigate } from "@/lib/router";

import { Page } from "@/components/shell/Page";
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
import { Empty, EmptyDescription, EmptyHeader, EmptyTitle } from "@/components/ui/empty";
import { Skeleton } from "@/components/ui/skeleton";

export function TaxonomiesPage() {
  const { t } = useI18n();
  const taxonomies = useTaxonomies();

  return (
    <Page title={t("taxonomies.title")} description={t("taxonomies.description")}>
      {taxonomies.data?.items.length === 0 ? (
        <Empty className="border">
          <EmptyHeader>
            <EmptyTitle>{t("taxonomies.empty")}</EmptyTitle>
            <EmptyDescription>{t("taxonomies.description")}</EmptyDescription>
          </EmptyHeader>
        </Empty>
      ) : (
        <div className="grid gap-3.5 xl:grid-cols-2">
          {taxonomies.data?.items.map((item) => <TaxonomyCard key={item.name} taxonomy={item} />)}
        </div>
      )}
    </Page>
  );
}

function TaxonomyCard({ taxonomy }: { taxonomy: Taxonomy }) {
  const { t } = useI18n();
  const taxonomyLabel = useTaxonomyLabel();
  const types = useContentTypes();
  const terms = useTerms(taxonomy.name);

  // A term opens the listing of whichever kind carries this taxonomy.
  const kind = types.data?.items.find((type) => type.taxonomies?.includes(taxonomy.name))?.kind;

  const open = (term: string) => {
    if (!kind) return;
    setListState(kind, { status: "all", q: "", terms: { [taxonomy.name]: term } });
    navigate({ name: "list", kind });
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm">{taxonomyLabel(taxonomy.name)}</CardTitle>
        <CardDescription>{t("taxonomies.terms", { count: taxonomy.terms })}</CardDescription>
        <CardAction>
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("list.open")}
            nativeButton={false}
            render={<a href={taxonomy.url} target="_blank" rel="noreferrer" />}
          >
            <ExternalLink />
          </Button>
        </CardAction>
      </CardHeader>
      <CardContent className="flex flex-wrap gap-1.5">
        {!terms.data
          ? Array.from({ length: 5 }, (_, i) => <Skeleton key={i} className="h-5 w-16" />)
          : terms.data.items.map((item) => (
              <Badge
                key={item.term}
                variant="secondary"
                render={<button type="button" onClick={() => open(item.term)} />}
              >
                {item.term}
                <span className="tabular-nums opacity-60">{item.count}</span>
              </Badge>
            ))}
      </CardContent>
    </Card>
  );
}
