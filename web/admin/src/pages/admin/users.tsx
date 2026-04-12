import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Users, Trash2 } from "lucide-react";
import { userApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";

function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

export default function UsersPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);

  const { data, isLoading } = useQuery({
    queryKey: ["users", page],
    queryFn: () => userApi.list({ page, size: 20 }).then((r) => r.data.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => userApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["users"] }),
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">{t("users.title")}</h1>
        <p className="text-sm text-muted-foreground">{t("users.description")}</p>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : (
        <>
          <div className="rounded-lg border">
            <div className="grid grid-cols-[1fr_1fr_auto_auto_auto] gap-4 border-b px-4 py-2 text-xs font-medium text-muted-foreground">
              <span>{t("users.username")}</span>
              <span>{t("users.email")}</span>
              <span>{t("users.role")}</span>
              <span>{t("users.storageCol")}</span>
              <span />
            </div>
            {data?.items?.map(
              (user: {
                id: string;
                username: string;
                email: string;
                role: string;
                storage_used: number;
                storage_limit: number;
                is_active: boolean;
              }) => (
                <div
                  key={user.id}
                  className="grid grid-cols-[1fr_1fr_auto_auto_auto] items-center gap-4 border-b px-4 py-3 text-sm last:border-0"
                >
                  <div className="flex items-center gap-2">
                    <span className="font-medium">{user.username}</span>
                    {!user.is_active && (
                      <Badge variant="outline" className="text-[10px]">
                        {t("common.disabled")}
                      </Badge>
                    )}
                  </div>
                  <span className="text-muted-foreground truncate">
                    {user.email}
                  </span>
                  <Badge
                    variant={user.role === "admin" ? "default" : "secondary"}
                    className="text-[10px]"
                  >
                    {user.role}
                  </Badge>
                  <span className="text-xs text-muted-foreground whitespace-nowrap">
                    {formatBytes(user.storage_used)}
                    {user.storage_limit > 0 &&
                      ` / ${formatBytes(user.storage_limit)}`}
                  </span>
                  <Button
                    size="icon-xs"
                    variant="ghost"
                    onClick={() => deleteMutation.mutate(user.id)}
                  >
                    <Trash2 className="size-3.5 text-destructive" />
                  </Button>
                </div>
              )
            )}
          </div>

          {data?.items?.length === 0 && (
            <div className="flex flex-col items-center py-16 text-center">
              <Users className="mb-3 size-12 text-muted-foreground/30" />
              <p className="text-sm text-muted-foreground">{t("users.noUsers")}</p>
            </div>
          )}

          {data && data.total > 20 && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                {t("common.previous")}
              </Button>
              <span className="text-sm text-muted-foreground">
                {t("common.page")} {page} {t("common.of")} {Math.ceil(data.total / 20)}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= Math.ceil(data.total / 20)}
                onClick={() => setPage((p) => p + 1)}
              >
                {t("common.next")}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
