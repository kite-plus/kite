import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus,
  Trash2,
  Key,
  Copy,
  Check,
  Upload,
  Eye,
  Sparkles,
  Activity,
  ShieldCheck,
} from "lucide-react";

import { tokenApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/page-header";
import { EmptyKite } from "@/components/empty-state";
import { toast } from "sonner";

interface TokenItem {
  id: string;
  name: string;
  last_used?: string;
  expires_at?: string;
  created_at: string;
}

interface TokenStatProps {
  icon: React.ElementType;
  label: string;
  value: string | number;
  hint: string;
  tint: "indigo" | "emerald" | "amber";
}

const TINTS: Record<
  TokenStatProps["tint"],
  { bg: string; text: string; ring: string }
> = {
  indigo: {
    bg: "bg-[color:var(--chart-1)]/10",
    text: "text-[color:var(--chart-1)]",
    ring: "ring-[color:var(--chart-1)]/25",
  },
  emerald: {
    bg: "bg-[color:var(--chart-2)]/10",
    text: "text-[color:var(--chart-2)]",
    ring: "ring-[color:var(--chart-2)]/25",
  },
  amber: {
    bg: "bg-[color:var(--chart-4)]/10",
    text: "text-[color:var(--chart-4)]",
    ring: "ring-[color:var(--chart-4)]/25",
  },
};

function TokenStat({ icon: Icon, label, value, hint, tint }: TokenStatProps) {
  const palette = TINTS[tint];
  return (
    <div className="glow-card relative overflow-hidden rounded-xl border bg-card p-5">
      <div className="dot-grid pointer-events-none absolute inset-0 opacity-30" />
      <div className="relative flex items-start gap-4">
        <div
          className={`flex size-11 shrink-0 items-center justify-center rounded-lg ring-1 ${palette.bg} ${palette.text} ${palette.ring}`}
        >
          <Icon className="size-5" />
        </div>
        <div className="min-w-0 flex-1">
          <p className="truncate text-xs font-medium text-muted-foreground">
            {label}
          </p>
          <p className="mt-1 text-2xl font-semibold tracking-tight tabular-nums">
            {value}
          </p>
          <p className="mt-1 truncate text-xs text-muted-foreground">{hint}</p>
        </div>
      </div>
    </div>
  );
}

function maskToken(id: string) {
  const suffix = id.replace(/-/g, "").slice(-4).toUpperCase();
  return `kite_••••${suffix}`;
}

export default function TokensPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [name, setName] = useState("");
  const [scopeUpload, setScopeUpload] = useState(true);
  const [scopeRead, setScopeRead] = useState(true);
  const [newToken, setNewToken] = useState("");
  const [copied, setCopied] = useState(false);
  const [rowCopiedId, setRowCopiedId] = useState<string | null>(null);

  const { data, isLoading } = useQuery({
    queryKey: ["tokens"],
    queryFn: () => tokenApi.list().then((r) => r.data.data),
  });

  const tokens: TokenItem[] = useMemo(() => data ?? [], [data]);

  // Keep a "now" in state so relative-time labels refresh periodically and
  // Date.now() stays outside of render / memo bodies (purity rule).
  const [now, setNow] = useState<number>(() => Date.now());
  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), 60_000);
    return () => clearInterval(id);
  }, []);

  const stats = useMemo(() => {
    const sevenDays = 7 * 24 * 60 * 60 * 1000;
    const oneDay = 24 * 60 * 60 * 1000;
    const createdThisWeek = tokens.filter(
      (tk) => now - new Date(tk.created_at).getTime() < sevenDays,
    ).length;
    const recentlyActive = tokens.filter(
      (tk) => tk.last_used && now - new Date(tk.last_used).getTime() < oneDay,
    ).length;
    return {
      total: tokens.length,
      createdThisWeek,
      recentlyActive,
    };
  }, [tokens, now]);

  const createMutation = useMutation({
    mutationFn: () => tokenApi.create(name),
    onSuccess: (res) => {
      setNewToken(res.data.data.token);
      setName("");
      queryClient.invalidateQueries({ queryKey: ["tokens"] });
      toast.success(t("tokens.createSuccess"));
    },
    onError: () => toast.error(t("tokens.createFailed")),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => tokenApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["tokens"] });
      toast.success(t("tokens.deleteSuccess"));
    },
    onError: () => toast.error(t("tokens.deleteFailed")),
  });

  const copyToken = () => {
    navigator.clipboard.writeText(newToken);
    toast.success(t("tokens.copyHint"));
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const copyRow = (token: TokenItem) => {
    navigator.clipboard.writeText(maskToken(token.id));
    setRowCopiedId(token.id);
    setTimeout(() => setRowCopiedId(null), 1500);
  };

  const requestDelete = (token: TokenItem) => {
    if (window.confirm(t("tokens.deleteConfirm"))) {
      deleteMutation.mutate(token.id);
    }
  };

  const formatRelative = (iso?: string) => {
    if (!iso) return t("tokens.neverUsed");
    const diff = now - new Date(iso).getTime();
    const m = Math.floor(diff / 60_000);
    const h = Math.floor(diff / 3_600_000);
    const d = Math.floor(diff / 86_400_000);
    if (m < 1) return t("tokens.justNow");
    if (m < 60) return t("tokens.minutesAgo").replace("{n}", String(m));
    if (h < 24) return t("tokens.hoursAgo").replace("{n}", String(h));
    if (d < 30) return t("tokens.daysAgo").replace("{n}", String(d));
    return new Date(iso).toLocaleDateString();
  };

  const resetCreateForm = () => {
    setName("");
    setScopeUpload(true);
    setScopeRead(true);
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
              if (!open) {
                setNewToken("");
                resetCreateForm();
              }
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
                      {copied ? (
                        <Check className="size-4" />
                      ) : (
                        <Copy className="size-4" />
                      )}
                    </Button>
                  </div>
                  <div className="flex items-start gap-2 rounded-md border bg-muted/40 p-3 text-xs text-muted-foreground">
                    <ShieldCheck className="mt-0.5 size-3.5 shrink-0" />
                    <span>{t("tokens.maskedHint")}</span>
                  </div>
                </div>
              ) : (
                <>
                  <div className="grid gap-4">
                    <div className="grid gap-2">
                      <Label>{t("tokens.tokenName")}</Label>
                      <Input
                        value={name}
                        onChange={(e) => setName(e.target.value)}
                        placeholder={t("tokens.tokenNamePlaceholder")}
                        autoFocus
                      />
                    </div>

                    <div className="grid gap-2">
                      <Label>{t("tokens.scope")}</Label>
                      <p className="text-xs text-muted-foreground">
                        {t("tokens.scopeHint")}
                      </p>
                      <div className="grid gap-2 rounded-lg border bg-muted/30 p-1">
                        <label
                          htmlFor="scope-upload"
                          className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2.5 hover:bg-muted/70"
                        >
                          <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-[color:var(--chart-1)]/15 text-[color:var(--chart-1)]">
                            <Upload className="size-4" />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="text-sm font-medium">
                              {t("tokens.scopeUpload")}
                            </div>
                            <div className="truncate text-xs text-muted-foreground">
                              PicGo · uPic · ShareX
                            </div>
                          </div>
                          <Switch
                            id="scope-upload"
                            checked={scopeUpload}
                            onCheckedChange={setScopeUpload}
                          />
                        </label>
                        <label
                          htmlFor="scope-read"
                          className="flex cursor-pointer items-center gap-3 rounded-md px-3 py-2.5 hover:bg-muted/70"
                        >
                          <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-[color:var(--chart-2)]/15 text-[color:var(--chart-2)]">
                            <Eye className="size-4" />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="text-sm font-medium">
                              {t("tokens.scopeRead")}
                            </div>
                            <div className="truncate text-xs text-muted-foreground">
                              list · download · metadata
                            </div>
                          </div>
                          <Switch
                            id="scope-read"
                            checked={scopeRead}
                            onCheckedChange={setScopeRead}
                          />
                        </label>
                      </div>
                    </div>
                  </div>
                  <DialogFooter>
                    <Button
                      onClick={() => createMutation.mutate()}
                      disabled={
                        !name ||
                        createMutation.isPending ||
                        (!scopeUpload && !scopeRead)
                      }
                    >
                      {createMutation.isPending
                        ? t("tokens.creating")
                        : t("common.create")}
                    </Button>
                  </DialogFooter>
                </>
              )}
            </DialogContent>
          </Dialog>
        }
      />

      {/* Stat tiles */}
      <div className="grid gap-3 sm:grid-cols-3">
        {isLoading ? (
          Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-24 rounded-xl" />
          ))
        ) : (
          <>
            <TokenStat
              icon={Key}
              label={t("tokens.totalTokens")}
              value={stats.total}
              hint={t("tokens.totalTokensDesc")}
              tint="indigo"
            />
            <TokenStat
              icon={Sparkles}
              label={t("tokens.thisWeek")}
              value={stats.createdThisWeek}
              hint={t("tokens.thisWeekDesc")}
              tint="emerald"
            />
            <TokenStat
              icon={Activity}
              label={t("tokens.recentlyActive")}
              value={stats.recentlyActive}
              hint={t("tokens.recentlyActiveDesc")}
              tint="amber"
            />
          </>
        )}
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      ) : tokens.length === 0 ? (
        <EmptyKite
          title={t("tokens.noTokens")}
          hint={t("tokens.noTokensHint")}
          action={
            <Button size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="size-3.5" />
              {t("tokens.newToken")}
            </Button>
          }
        />
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden overflow-hidden rounded-xl border sm:block">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                  <TableHead>{t("tokens.columnName")}</TableHead>
                  <TableHead>{t("tokens.columnValue")}</TableHead>
                  <TableHead>{t("tokens.columnScope")}</TableHead>
                  <TableHead>{t("tokens.columnCreated")}</TableHead>
                  <TableHead>{t("tokens.columnLastUsed")}</TableHead>
                  <TableHead className="w-16" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className="font-medium">
                      <div className="flex items-center gap-2.5">
                        <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted">
                          <Key className="size-3.5 text-muted-foreground" />
                        </div>
                        <span className="truncate">{token.name}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <button
                        type="button"
                        onClick={() => copyRow(token)}
                        className="group inline-flex items-center gap-2 rounded-md border bg-muted/40 px-2 py-1 font-mono text-xs text-muted-foreground transition hover:border-foreground/25 hover:text-foreground"
                      >
                        <span>{maskToken(token.id)}</span>
                        {rowCopiedId === token.id ? (
                          <Check className="size-3 text-[color:var(--chart-2)]" />
                        ) : (
                          <Copy className="size-3 opacity-60 group-hover:opacity-100" />
                        )}
                      </button>
                    </TableCell>
                    <TableCell>
                      <div className="flex gap-1">
                        <Badge
                          variant="secondary"
                          className="h-5 gap-1 px-1.5 text-[10px] font-medium"
                        >
                          <Upload className="size-2.5" />
                          {t("tokens.scopeUpload")}
                        </Badge>
                        <Badge
                          variant="outline"
                          className="h-5 gap-1 px-1.5 text-[10px] font-medium"
                        >
                          <Eye className="size-2.5" />
                          {t("tokens.scopeRead")}
                        </Badge>
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground tabular-nums">
                      {formatRelative(token.created_at)}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground tabular-nums">
                      {token.last_used
                        ? formatRelative(token.last_used)
                        : t("tokens.neverUsed")}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Button
                          size="icon-sm"
                          variant="ghost"
                          className="text-muted-foreground hover:text-destructive"
                          onClick={() => requestDelete(token)}
                          disabled={deleteMutation.isPending}
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {/* Mobile cards */}
          <div className="space-y-2 sm:hidden">
            {tokens.map((token) => (
              <div
                key={token.id}
                className="glow-card rounded-lg border bg-card px-4 py-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <Key className="size-3.5 shrink-0 text-muted-foreground" />
                      <p className="truncate text-sm font-medium">
                        {token.name}
                      </p>
                    </div>
                    <button
                      type="button"
                      onClick={() => copyRow(token)}
                      className="mt-2 inline-flex items-center gap-1.5 rounded-md border bg-muted/40 px-2 py-0.5 font-mono text-[11px] text-muted-foreground"
                    >
                      <span>{maskToken(token.id)}</span>
                      {rowCopiedId === token.id ? (
                        <Check className="size-3 text-[color:var(--chart-2)]" />
                      ) : (
                        <Copy className="size-3" />
                      )}
                    </button>
                    <div className="mt-2 flex flex-wrap gap-1">
                      <Badge
                        variant="secondary"
                        className="h-4 gap-1 px-1.5 text-[10px] font-medium"
                      >
                        <Upload className="size-2.5" />
                        {t("tokens.scopeUpload")}
                      </Badge>
                      <Badge
                        variant="outline"
                        className="h-4 gap-1 px-1.5 text-[10px] font-medium"
                      >
                        <Eye className="size-2.5" />
                        {t("tokens.scopeRead")}
                      </Badge>
                    </div>
                    <p className="mt-2 text-[11px] text-muted-foreground">
                      {t("tokens.created")} {formatRelative(token.created_at)}
                      <span className="mx-1.5 text-border">·</span>
                      {token.last_used
                        ? `${t("tokens.lastUsed")} ${formatRelative(token.last_used)}`
                        : t("tokens.neverUsed")}
                    </p>
                  </div>
                  <Button
                    size="icon-sm"
                    variant="ghost"
                    className="shrink-0 text-muted-foreground hover:text-destructive"
                    onClick={() => requestDelete(token)}
                    disabled={deleteMutation.isPending}
                  >
                    <Trash2 className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
