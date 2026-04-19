import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Trash2,
  Plus,
  Pencil,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
} from "lucide-react";

import { userApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { PageHeader } from "@/components/page-header";
import { EmptyKite } from "@/components/empty-state";
import { toast } from "sonner";

interface UserItem {
  id: string;
  username: string;
  nickname?: string;
  email: string;
  avatar_url?: string;
  role: string;
  storage_used: number;
  storage_limit: number;
  is_active: boolean;
}

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
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<UserItem | null>(null);
  const [form, setForm] = useState({
    username: "",
    nickname: "",
    email: "",
    password: "",
    role: "user",
    storage_limit: "10737418240",
    is_active: true,
  });
  const [error, setError] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["users", page],
    queryFn: () => userApi.list({ page, size: 20 }).then((r) => r.data.data),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      userApi.create({
        username: form.username,
        nickname: form.nickname,
        email: form.email,
        password: form.password,
        role: form.role,
        storage_limit: parseInt(form.storage_limit),
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      closeDialog();
      toast.success(t("toast.createUser"));
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t("users.saveFailed");
      setError(msg);
    },
  });

  const updateMutation = useMutation({
    mutationFn: () => {
      const payload: Record<string, unknown> = {
        role: form.role,
        is_active: form.is_active,
        storage_limit: parseInt(form.storage_limit),
      };
      if (form.email) payload.email = form.email;
      payload.nickname = form.nickname;
      if (form.password) payload.password = form.password;
      return userApi.update(editingUser!.id, payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      closeDialog();
      toast.success(t("toast.updateUser"));
    },
    onError: (err: unknown) => {
      const msg =
        (err as { response?: { data?: { message?: string } } })?.response?.data
          ?.message ?? t("users.saveFailed");
      setError(msg);
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => userApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users"] });
      toast.success(t("toast.deleteUser"));
    },
    onError: () => toast.error(t("users.deleteFailed")),
  });

  const requestDelete = (user: UserItem) => {
    if (window.confirm(t("users.deleteConfirm"))) {
      deleteMutation.mutate(user.id);
    }
  };

  const openCreate = () => {
    setEditingUser(null);
    setForm({
      username: "",
      nickname: "",
      email: "",
      password: "",
      role: "user",
      storage_limit: "10737418240",
      is_active: true,
    });
    setError("");
    setDialogOpen(true);
  };

  const openEdit = (user: UserItem) => {
    setEditingUser(user);
    setForm({
      username: user.username,
      nickname: user.nickname ?? "",
      email: user.email,
      password: "",
      role: user.role,
      storage_limit: String(user.storage_limit),
      is_active: user.is_active,
    });
    setError("");
    setDialogOpen(true);
  };

  const closeDialog = () => {
    setDialogOpen(false);
    setEditingUser(null);
    setError("");
  };

  const handleSave = () => {
    if (editingUser) {
      updateMutation.mutate();
    } else {
      createMutation.mutate();
    }
  };

  const isPending = createMutation.isPending || updateMutation.isPending;
  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  return (
    <div className="space-y-8">
      <PageHeader
        title={t("users.title")}
        description={t("users.description")}
        actions={
          <Button onClick={openCreate}>
            <Plus className="size-4" />
            {t("users.addUser")}
          </Button>
        }
      />

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 rounded-lg" />
          ))}
        </div>
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden overflow-hidden rounded-xl border sm:block">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/30 hover:bg-muted/30">
                  <TableHead>{t("users.account")}</TableHead>
                  <TableHead>{t("users.email")}</TableHead>
                  <TableHead>{t("users.role")}</TableHead>
                  <TableHead className="min-w-[180px]">
                    {t("users.storageCol")}
                  </TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items?.map((user: UserItem) => {
                  const pct =
                    user.storage_limit > 0
                      ? Math.min(
                          100,
                          (user.storage_used / user.storage_limit) * 100,
                        )
                      : 0;
                  const quotaLabel =
                    user.storage_limit > 0
                      ? `${formatBytes(user.storage_used)} / ${formatBytes(user.storage_limit)}`
                      : `${formatBytes(user.storage_used)} · ${t("users.unlimited")}`;
                  const name = user.nickname?.trim() || user.username;
                  return (
                    <TableRow key={user.id}>
                      <TableCell className="font-medium">
                        <div className="flex items-center gap-3">
                          <Avatar className="size-9 shrink-0">
                            <AvatarImage
                              src={user.avatar_url}
                              alt={name}
                              className="object-cover"
                            />
                            <AvatarFallback className="text-xs font-semibold">
                              {name.charAt(0).toUpperCase()}
                            </AvatarFallback>
                          </Avatar>
                          <div className="min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="truncate font-medium">
                                {name}
                              </span>
                              {!user.is_active && (
                                <Badge
                                  variant="outline"
                                  className="text-[10px]"
                                >
                                  {t("common.disabled")}
                                </Badge>
                              )}
                            </div>
                            <div className="truncate text-xs text-muted-foreground">
                              @{user.username}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {user.email}
                      </TableCell>
                      <TableCell>
                        <Badge
                          variant={
                            user.role === "admin" ? "default" : "secondary"
                          }
                          className="text-[10px]"
                        >
                          {user.role}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <div className="min-w-[160px] space-y-1.5">
                          <div className="flex items-baseline justify-between gap-2 text-xs">
                            <span className="truncate text-muted-foreground">
                              {quotaLabel}
                            </span>
                            {user.storage_limit > 0 && (
                              <span className="shrink-0 font-medium tabular-nums">
                                {pct.toFixed(0)}%
                              </span>
                            )}
                          </div>
                          {user.storage_limit > 0 ? (
                            <Progress value={pct} className="h-1" />
                          ) : (
                            <div className="h-1 rounded-full bg-gradient-to-r from-[color:var(--chart-2)]/30 via-[color:var(--chart-1)]/30 to-[color:var(--chart-4)]/30" />
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end gap-1">
                          <Button
                            size="icon-xs"
                            variant="ghost"
                            onClick={() => openEdit(user)}
                          >
                            <Pencil className="size-3.5" />
                          </Button>
                          <Button
                            size="icon-xs"
                            variant="ghost"
                            onClick={() => requestDelete(user)}
                          >
                            <Trash2 className="size-3.5 text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          </div>

          {/* Mobile cards */}
          <div className="space-y-3 sm:hidden">
            {data?.items?.map((user: UserItem) => {
              const pct =
                user.storage_limit > 0
                  ? Math.min(
                      100,
                      (user.storage_used / user.storage_limit) * 100,
                    )
                  : 0;
              const name = user.nickname?.trim() || user.username;
              return (
                <div key={user.id} className="rounded-xl border bg-card p-4">
                  <div className="flex items-start gap-3">
                    <Avatar className="size-10 shrink-0">
                      <AvatarImage
                        src={user.avatar_url}
                        alt={name}
                        className="object-cover"
                      />
                      <AvatarFallback className="text-sm font-semibold">
                        {name.charAt(0).toUpperCase()}
                      </AvatarFallback>
                    </Avatar>
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-2">
                        <span className="truncate text-sm font-medium">
                          {name}
                        </span>
                        <Badge
                          variant={
                            user.role === "admin" ? "default" : "secondary"
                          }
                          className="text-[10px]"
                        >
                          {user.role}
                        </Badge>
                        {!user.is_active && (
                          <Badge variant="outline" className="text-[10px]">
                            {t("common.disabled")}
                          </Badge>
                        )}
                      </div>
                      <p className="mt-0.5 truncate text-xs text-muted-foreground">
                        @{user.username}
                      </p>
                      <p className="mt-0.5 truncate text-xs text-muted-foreground">
                        {user.email}
                      </p>
                      <div className="mt-2 space-y-1">
                        <div className="flex items-baseline justify-between gap-2 text-[11px]">
                          <span className="truncate text-muted-foreground">
                            {formatBytes(user.storage_used)}
                            {user.storage_limit > 0
                              ? ` / ${formatBytes(user.storage_limit)}`
                              : ` · ${t("users.unlimited")}`}
                          </span>
                          {user.storage_limit > 0 && (
                            <span className="shrink-0 font-medium tabular-nums">
                              {pct.toFixed(0)}%
                            </span>
                          )}
                        </div>
                        {user.storage_limit > 0 && (
                          <Progress value={pct} className="h-1" />
                        )}
                      </div>
                    </div>
                    <div className="flex shrink-0 gap-1">
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        onClick={() => openEdit(user)}
                      >
                        <Pencil className="size-3.5" />
                      </Button>
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        onClick={() => requestDelete(user)}
                      >
                        <Trash2 className="size-3.5 text-destructive" />
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>

          {data?.items?.length === 0 && (
            <EmptyKite
              title={t("users.noUsers")}
              hint={t("users.noUsersHint")}
              action={
                <Button size="sm" onClick={openCreate}>
                  <Plus className="size-3.5" />
                  {t("users.addUser")}
                </Button>
              }
            />
          )}

          {data && data.total > 20 && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="icon-sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <ChevronLeft className="size-4" />
              </Button>
              <span className="min-w-15 text-center text-sm text-muted-foreground">
                {page} / {totalPages}
              </span>
              <Button
                variant="outline"
                size="icon-sm"
                disabled={page >= totalPages}
                onClick={() => setPage((p) => p + 1)}
              >
                <ChevronRight className="size-4" />
              </Button>
            </div>
          )}
        </>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={(open) => !open && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingUser ? t("users.editUser") : t("users.addUser")}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="size-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label>{t("profile.nickname")}</Label>
              <Input
                value={form.nickname}
                onChange={(e) =>
                  setForm((f) => ({ ...f, nickname: e.target.value }))
                }
                placeholder={t("profile.nicknamePlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("users.username")}</Label>
              <Input
                value={form.username}
                onChange={(e) =>
                  setForm((f) => ({ ...f, username: e.target.value }))
                }
                disabled={!!editingUser}
                placeholder={t("auth.chooseUsername")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("users.email")}</Label>
              <Input
                type="email"
                value={form.email}
                onChange={(e) =>
                  setForm((f) => ({ ...f, email: e.target.value }))
                }
                placeholder={t("auth.emailPlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>
                {t("auth.password")}
                {editingUser && (
                  <span className="ml-1 text-xs text-muted-foreground">
                    ({t("users.leaveBlank")})
                  </span>
                )}
              </Label>
              <Input
                type="password"
                value={form.password}
                onChange={(e) =>
                  setForm((f) => ({ ...f, password: e.target.value }))
                }
                placeholder={
                  editingUser ? t("users.leaveBlank") : t("auth.passwordHint")
                }
              />
            </div>
            <div className="flex gap-4">
              <div className="flex-1 space-y-2">
                <Label>{t("users.role")}</Label>
                <div className="flex gap-2">
                  {["user", "admin"].map((r) => (
                    <Button
                      key={r}
                      type="button"
                      size="sm"
                      variant={form.role === r ? "default" : "outline"}
                      onClick={() => setForm((f) => ({ ...f, role: r }))}
                    >
                      {r}
                    </Button>
                  ))}
                </div>
              </div>
              {editingUser && (
                <div className="flex-1 space-y-2">
                  <Label>{t("users.status")}</Label>
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      size="sm"
                      variant={form.is_active ? "default" : "outline"}
                      onClick={() => setForm((f) => ({ ...f, is_active: true }))}
                    >
                      {t("common.active")}
                    </Button>
                    <Button
                      type="button"
                      size="sm"
                      variant={!form.is_active ? "destructive" : "outline"}
                      onClick={() => setForm((f) => ({ ...f, is_active: false }))}
                    >
                      {t("common.disabled")}
                    </Button>
                  </div>
                </div>
              )}
            </div>
            <div className="space-y-2">
              <Label>{t("users.storageLimit")}</Label>
              <Select
                value={form.storage_limit}
                onValueChange={(v) =>
                  setForm((f) => ({ ...f, storage_limit: v }))
                }
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="1073741824">1 GB</SelectItem>
                  <SelectItem value="5368709120">5 GB</SelectItem>
                  <SelectItem value="10737418240">10 GB</SelectItem>
                  <SelectItem value="53687091200">50 GB</SelectItem>
                  <SelectItem value="-1">{t("users.unlimited")}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={handleSave}
              disabled={
                isPending ||
                (!editingUser &&
                  (!form.username || !form.email || !form.password))
              }
            >
              {isPending ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
