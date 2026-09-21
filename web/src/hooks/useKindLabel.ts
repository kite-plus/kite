import { useI18n } from "@/i18n";
import { useContentTypes } from "@/hooks/useContents";

/**
 * useKindLabel names a content kind in the operator's language.
 *
 * The built-in kinds are translated; any other falls back to the label the
 * project declared for it.
 */
export function useKindLabel() {
  const { t } = useI18n();
  const types = useContentTypes();

  const declared = (kind: string) =>
    types.data?.items.find((type) => type.kind === kind)?.label ?? kind;

  return {
    one: (kind: string) =>
      kind === "post" ? t("kind.post") : kind === "page" ? t("kind.page") : declared(kind),
    many: (kind: string) =>
      kind === "post" ? t("nav.posts") : kind === "page" ? t("nav.pages") : declared(kind),
  };
}

/** useTaxonomyLabel translates the two built-in taxonomies and leaves the rest. */
export function useTaxonomyLabel() {
  const { t } = useI18n();
  return (name: string) =>
    name === "tags" ? t("taxonomy.tags") : name === "categories" ? t("taxonomy.categories") : name;
}
