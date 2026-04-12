import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { HardDrive, Trash2, Check } from "lucide-react";
import { storageApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

export default function StoragePage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();

  const { data, isLoading } = useQuery({
    queryKey: ["storage"],
    queryFn: () => storageApi.list().then((r) => r.data.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => storageApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["storage"] }),
  });

  const testMutation = useMutation({
    mutationFn: (id: string) => storageApi.test(id),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("storage.title")}</h1>
        <p className="text-sm text-muted-foreground">
          {t("storage.description")}
        </p>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-lg" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          {data?.map(
            (cfg: {
              id: string;
              name: string;
              driver: string;
              is_default: boolean;
              is_active: boolean;
            }) => (
              <div
                key={cfg.id}
                className="flex items-center justify-between rounded-lg border bg-card p-4"
              >
                <div className="flex items-center gap-3">
                  <HardDrive className="size-5 text-muted-foreground" />
                  <div>
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium">{cfg.name}</p>
                      <Badge variant="outline" className="text-[10px]">
                        {cfg.driver}
                      </Badge>
                      {cfg.is_default && (
                        <Badge variant="secondary" className="text-[10px]">
                          {t("common.default")}
                        </Badge>
                      )}
                    </div>
                    <p className="text-xs text-muted-foreground">
                      {cfg.is_active ? t("common.active") : t("common.inactive")}
                    </p>
                  </div>
                </div>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => testMutation.mutate(cfg.id)}
                    disabled={testMutation.isPending}
                  >
                    {testMutation.isSuccess ? (
                      <Check className="size-4 text-green-600" />
                    ) : (
                      t("common.test")
                    )}
                  </Button>
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    onClick={() => deleteMutation.mutate(cfg.id)}
                    disabled={cfg.is_default}
                  >
                    <Trash2 className="size-4 text-destructive" />
                  </Button>
                </div>
              </div>
            )
          )}

          {data?.length === 0 && (
            <div className="flex flex-col items-center py-16 text-center">
              <HardDrive className="mb-3 size-12 text-muted-foreground/30" />
              <p className="text-sm text-muted-foreground">
                {t("storage.noStorage")}
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
