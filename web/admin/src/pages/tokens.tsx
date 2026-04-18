import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, Key, Copy, Check } from "lucide-react";

import { tokenApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/page-header";
import { toast } from "sonner";

export default function TokensPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [newToken, setNewToken] = useState("");
  const [copied, setCopied] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["tokens"],
    queryFn: () => tokenApi.list().then((r) => r.data.data),
  });

  const createMutation = useMutation({
    mutationFn: () => tokenApi.create(name),
    onSuccess: (res) => {
      setNewToken(res.data.data.token);
      setName("");
      queryClient.invalidateQueries({ queryKey: ["tokens"] });
      toast.success("API Token 创建成功");
    },
    onError: () => toast.error("创建失败"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => tokenApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tokens"] });
      toast.success("删除成功");
    },
    onError: () => toast.error("删除失败"),
  });

  const copyToken = () => {
    navigator.clipboard.writeText(newToken);
    toast.success("已复制到剪贴板！请妥善保存该 Token");
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-8">
      <PageHeader
        title={t("tokens.title")}
        description={t("tokens.description")}
        actions={
          <Dialog
            open={createOpen}
            onOpenChange={(open) => {
              setCreateOpen(open);
              if (!open) setNewToken("");
            }}
          >
            <DialogTrigger asChild>
              <Button>
                <Plus className="size-4" />
                {t("tokens.newToken")}
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>
                  {newToken ? t("tokens.tokenCreated") : t("tokens.createToken")}
                </DialogTitle>
              </DialogHeader>
              {newToken ? (
                <div className="grid gap-3">
                  <p className="text-sm text-muted-foreground">
                    {t("tokens.copyWarning")}
                  </p>
                  <div className="flex gap-2">
                    <Input value={newToken} readOnly className="font-mono" />
                    <Button variant="outline" onClick={copyToken}>
                      {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <div className="grid gap-2">
                    <Label>{t("tokens.tokenName")}</Label>
                    <Input
                      value={name}
                      onChange={(e) => setName(e.target.value)}
                      placeholder={t("tokens.tokenNamePlaceholder")}
                    />
                  </div>
                  <DialogFooter>
                    <Button
                      onClick={() => createMutation.mutate()}
                      disabled={!name || createMutation.isPending}
                    >
                      {createMutation.isPending ? t("tokens.creating") : t("common.create")}
                    </Button>
                  </DialogFooter>
                </>
              )}
            </DialogContent>
          </Dialog>
        }
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : (
        <div className="space-y-2">
          {data?.map(
            (token: {
              id: string;
              name: string;
              last_used?: string;
              expires_at?: string;
              created_at: string;
            }) => (
              <div
                key={token.id}
                className="flex items-center justify-between gap-3 rounded-lg border bg-card px-4 py-3 transition-colors hover:border-foreground/20"
              >
                <div className="flex min-w-0 items-center gap-3">
                  <div className="flex size-9 shrink-0 items-center justify-center rounded-md bg-muted">
                    <Key className="size-4 text-muted-foreground" />
                  </div>
                  <div className="min-w-0">
                    <p className="truncate text-sm font-medium">{token.name}</p>
                    <p className="truncate text-xs text-muted-foreground">
                      {t("tokens.created")}{" "}
                      {new Date(token.created_at).toLocaleDateString()}
                      {token.last_used &&
                        ` · ${t("tokens.lastUsed")} ${new Date(token.last_used).toLocaleDateString()}`}
                    </p>
                  </div>
                </div>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  className="shrink-0 text-muted-foreground hover:text-destructive"
                  onClick={() => deleteMutation.mutate(token.id)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            )
          )}

          {data?.length === 0 && (
            <div className="flex flex-col items-center rounded-xl border border-dashed py-16 text-center">
              <div className="flex size-14 items-center justify-center rounded-full bg-muted">
                <Key className="size-6 text-muted-foreground" />
              </div>
              <p className="mt-4 text-sm font-medium text-muted-foreground">{t("tokens.noTokens")}</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
