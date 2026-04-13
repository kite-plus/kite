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
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => tokenApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["tokens"] }),
  });

  const copyToken = () => {
    navigator.clipboard.writeText(newToken);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("tokens.title")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("tokens.description")}
          </p>
        </div>
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
              <div className="space-y-3">
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
                <div className="space-y-2">
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
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
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
                className="flex items-center justify-between gap-3 rounded-lg border bg-card p-4"
              >
                <div className="flex items-center gap-3 min-w-0">
                  <Key className="size-4 text-muted-foreground shrink-0" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium truncate">{token.name}</p>
                    <p className="text-xs text-muted-foreground truncate">
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
                  className="shrink-0"
                  onClick={() => deleteMutation.mutate(token.id)}
                >
                  <Trash2 className="size-4 text-destructive" />
                </Button>
              </div>
            )
          )}

          {data?.length === 0 && (
            <div className="flex flex-col items-center py-16 text-center">
              <Key className="mb-3 size-12 text-muted-foreground/30" />
              <p className="text-sm text-muted-foreground">{t("tokens.noTokens")}</p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
