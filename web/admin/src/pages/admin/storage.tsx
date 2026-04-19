import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  HardDrive,
  Trash2,
  Check,
  Plus,
  AlertCircle,
  Layers,
  Repeat,
  Copy as CopyIcon,
  Infinity as InfinityIcon,
  GripVertical,
  Star,
  MoreHorizontal,
  Loader2,
  Settings as SettingsIcon,
  RefreshCw,
} from "lucide-react";
import {
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  closestCenter,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { sortableKeyboardCoordinates } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { adminStatsApi, settingsApi, storageApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { formatSize, cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { PageHeader, Section } from "@/components/page-header";
import { EmptyKite } from "@/components/empty-state";
import { BrandIcon, getBrandInfo } from "@/components/storage-brand";
import { StorageLogo, resolveLogoVendor } from "@/components/storage-logo";
import { toast } from "sonner";

interface StorageSegment {
  kind: string;
  label: string;
  count: number;
  bytes: number;
  color: string;
}

type Driver = "local" | "s3" | "oss" | "cos" | "ftp";
type Unit = "MB" | "GB" | "TB";

interface StorageListItem {
  id: string;
  name: string;
  driver: Driver;
  provider: string;
  capacity_limit_bytes: number;
  used_bytes: number;
  files_count?: number;
  priority: number;
  is_default: boolean;
  is_active: boolean;
}

interface StorageDetail extends StorageListItem {
  config: Record<string, unknown> | null;
}

interface StorageForm {
  name: string;
  driver: Driver;
  capacityValue: string;
  capacityUnit: Unit;
  config: {
    base_path?: string;
    base_url?: string;
    endpoint?: string;
    region?: string;
    bucket?: string;
    access_key_id?: string;
    secret_access_key?: string;
    force_path_style?: boolean;
    host?: string;
    port?: number;
    username?: string;
    password?: string;
  };
}

interface AdminStats {
  total_files: number;
  total_size: number;
  images: number;
  videos: number;
  audios: number;
  others: number;
}

const UNIT_BYTES: Record<Unit, number> = {
  MB: 1024 ** 2,
  GB: 1024 ** 3,
  TB: 1024 ** 4,
};

function bytesToUnit(bytes: number): { value: string; unit: Unit } {
  if (bytes <= 0) return { value: "", unit: "GB" };
  if (bytes % UNIT_BYTES.TB === 0) return { value: String(bytes / UNIT_BYTES.TB), unit: "TB" };
  if (bytes >= UNIT_BYTES.TB) return { value: (bytes / UNIT_BYTES.TB).toFixed(2), unit: "TB" };
  if (bytes >= UNIT_BYTES.GB) return { value: (bytes / UNIT_BYTES.GB).toFixed(2), unit: "GB" };
  return { value: (bytes / UNIT_BYTES.MB).toFixed(0), unit: "MB" };
}

function unitToBytes(value: string, unit: Unit): number {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return 0;
  return Math.floor(n * UNIT_BYTES[unit]);
}

function defaultConfigForDriver(driver: Driver): StorageForm["config"] {
  switch (driver) {
    case "local":
      return { base_path: "./uploads", base_url: "" };
    case "ftp":
      return { host: "", port: 21, username: "", password: "", base_path: "/" };
    default:
      return {};
  }
}

const emptyForm: StorageForm = {
  name: "",
  driver: "local",
  capacityValue: "",
  capacityUnit: "GB",
  config: defaultConfigForDriver("local"),
};

interface DriverOption {
  value: Driver;
  labelKey: string;
  descKey: string;
  provider: string;
}

const DRIVER_OPTIONS: DriverOption[] = [
  { value: "local", labelKey: "storage.driverLocal", descKey: "storage.driverLocalDesc", provider: "local" },
  { value: "s3", labelKey: "storage.driverS3", descKey: "storage.driverS3Desc", provider: "s3" },
  { value: "oss", labelKey: "storage.driverOss", descKey: "storage.driverOssDesc", provider: "aliyun-oss" },
  { value: "cos", labelKey: "storage.driverCos", descKey: "storage.driverCosDesc", provider: "tencent-cos" },
  { value: "ftp", labelKey: "storage.driverFtp", descKey: "storage.driverFtpDesc", provider: "ftp" },
];

const UPLOAD_POLICY_OPTIONS = [
  { value: "single", icon: HardDrive },
  { value: "primary_fallback", icon: Layers },
  { value: "round_robin", icon: Repeat },
  { value: "mirror", icon: CopyIcon },
] as const;

function mapStorageError(err: unknown, fallback: string): string {
  const status = (err as { response?: { status?: number } })?.response?.status;
  const msg =
    (err as { response?: { data?: { message?: string } } })?.response?.data?.message ?? "";

  if (status === 401) return "登录已过期，请重新登录";
  if (status === 403) return "没有权限执行此操作";
  if (status === 409) return "存储名称已存在";
  if (status && status >= 500) return "服务暂时不可用，请稍后再试";

  if (msg.includes("base_path is required")) return "根目录不能为空";
  if (msg.includes("base_url is required")) return "访问 URL 不能为空";
  if (msg.includes("resolve base_path")) return "根目录路径无效，请检查";
  if (msg.includes("create base_path")) return "无法创建根目录，请检查路径权限";
  if (msg.includes("bucket is required")) return "Bucket 不能为空";
  if (msg.includes("endpoint is required")) return "Endpoint 不能为空";
  if (msg.includes("access_key_id") || msg.includes("secret_access_key"))
    return "Access Key 和 Secret Key 不能为空";
  if (msg.includes("ftp driver: host")) return "FTP 主机不能为空";
  if (msg.includes("ftp driver: username")) return "FTP 用户名不能为空";
  if (msg.includes("ftp dial")) return "FTP 无法连接，请检查主机和端口";
  if (msg.includes("ftp login")) return "FTP 登录失败，请检查用户名和密码";
  if (msg.includes("s3 config is nil")) return "S3 配置不完整";
  if (msg.includes("local config is nil")) return "本地存储配置不完整";
  if (msg.includes("ftp config is nil")) return "FTP 配置不完整";
  if (msg.includes("unknown driver")) return "不支持的存储驱动";
  if (msg.startsWith("invalid storage config")) return "表单填写不完整，请检查";

  return fallback;
}

export default function StoragePage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<StorageForm>({ ...emptyForm });
  const [testResult, setTestResult] = useState<Record<string, "ok" | "fail" | "testing">>({});
  const [uploadPolicy, setUploadPolicy] = useState("single");

  const { data: settings } = useQuery<Record<string, string>>({
    queryKey: ["settings"],
    queryFn: () => settingsApi.get().then((r) => r.data.data),
  });

  useEffect(() => {
    if (settings?.["storage.upload_policy"]) {
      setUploadPolicy(settings["storage.upload_policy"]);
    }
  }, [settings]);

  const saveUploadPolicyMutation = useMutation({
    mutationFn: (policy: string) =>
      settingsApi.update({ "storage.upload_policy": policy }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      toast.success(t("storage.savePolicySuccess"));
    },
    onError: () => toast.error(t("storage.savePolicyFailed")),
  });

  const handlePolicyChange = (value: string) => {
    setUploadPolicy(value);
    saveUploadPolicyMutation.mutate(value);
  };

  const { data, isLoading: isStorageLoading } = useQuery<StorageListItem[]>({
    queryKey: ["storage"],
    queryFn: () => storageApi.list().then((r) => r.data.data),
  });

  const { data: adminStats } = useQuery<AdminStats>({
    queryKey: ["admin-stats"],
    queryFn: () => adminStatsApi.get().then((r) => r.data.data),
  });

  const [orderedIds, setOrderedIds] = useState<string[]>([]);
  useEffect(() => {
    if (data) setOrderedIds(data.map((c) => c.id));
  }, [data]);

  const itemsById = useMemo(() => {
    const map = new Map<string, StorageListItem>();
    (data ?? []).forEach((c) => map.set(c.id, c));
    return map;
  }, [data]);

  const orderedItems = useMemo(
    () => orderedIds.map((id) => itemsById.get(id)).filter(Boolean) as StorageListItem[],
    [orderedIds, itemsById],
  );

  const saveMutation = useMutation({
    mutationFn: () => {
      const payload = {
        name: form.name,
        driver: form.driver,
        config: normalizeConfig(form),
        capacity_limit_bytes: unitToBytes(form.capacityValue, form.capacityUnit),
      };
      return editingId ? storageApi.update(editingId, payload) : storageApi.create(payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage"] });
      closeDialog();
      toast.success(t("storage.savedSuccess"));
    },
    onError: (err) => toast.error(mapStorageError(err, t("storage.saveConfigFailed"))),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => storageApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage"] });
      toast.success(t("storage.deleteSuccess"));
    },
    onError: (err) => toast.error(mapStorageError(err, t("storage.deleteFailed"))),
  });

  const reorderMutation = useMutation({
    mutationFn: (ids: string[]) => storageApi.reorder(ids),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage"] });
      toast.success(t("storage.reorderSuccess"));
    },
    onError: (err) => {
      if (data) setOrderedIds(data.map((c) => c.id));
      toast.error(mapStorageError(err, t("storage.reorderFailed")));
    },
  });

  const setDefaultMutation = useMutation({
    mutationFn: (id: string) => storageApi.setDefault(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage"] });
      toast.success(t("storage.setDefaultSuccess"));
    },
    onError: (err) => toast.error(mapStorageError(err, t("storage.setDefaultFailed"))),
  });

  const toggleActiveMutation = useMutation({
    mutationFn: async ({ id, active }: { id: string; active: boolean }) => {
      const detail = await storageApi.get(id);
      const cfg = detail.data.data as StorageDetail;
      return storageApi.update(id, {
        name: cfg.name,
        driver: cfg.driver,
        config: (cfg.config as Record<string, unknown>) ?? {},
        capacity_limit_bytes: cfg.capacity_limit_bytes,
        priority: cfg.priority,
        is_default: cfg.is_default,
        is_active: active,
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage"] });
    },
    onError: (err) => toast.error(mapStorageError(err, t("storage.toggleActiveFailed"))),
  });

  const handleTest = async (id: string) => {
    setTestResult((prev) => ({ ...prev, [id]: "testing" }));
    try {
      await storageApi.test(id);
      setTestResult((prev) => ({ ...prev, [id]: "ok" }));
      toast.success(t("storage.testSuccess"));
      setTimeout(() => setTestResult((prev) => ({ ...prev, [id]: undefined! })), 3000);
    } catch (err) {
      setTestResult((prev) => ({ ...prev, [id]: "fail" }));
      toast.error(mapStorageError(err, t("storage.testFailedGeneric")));
      setTimeout(() => setTestResult((prev) => ({ ...prev, [id]: undefined! })), 3000);
    }
  };

  const openCreate = () => {
    setEditingId(null);
    setForm({ ...emptyForm });
    setDialogOpen(true);
  };

  const openEdit = async (id: string) => {
    try {
      const res = await storageApi.get(id);
      const cfg: StorageDetail = res.data.data;
      const { value, unit } = bytesToUnit(cfg.capacity_limit_bytes);
      setEditingId(cfg.id);
      setForm({
        name: cfg.name,
        driver: cfg.driver,
        capacityValue: value,
        capacityUnit: unit,
        config: (cfg.config as StorageForm["config"]) ?? defaultConfigForDriver(cfg.driver),
      });
      setDialogOpen(true);
    } catch (err) {
      toast.error(mapStorageError(err, t("storage.detailFailed")));
    }
  };

  const closeDialog = () => {
    setDialogOpen(false);
    setEditingId(null);
    setForm({ ...emptyForm });
  };

  const updateConfig = (key: string, value: string | number) =>
    setForm((prev) => ({ ...prev, config: { ...prev.config, [key]: value } }));

  const changeDriver = (driver: Driver) =>
    setForm((prev) => ({ ...prev, driver, config: defaultConfigForDriver(driver) }));

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const onDragEnd = (e: DragEndEvent) => {
    const { active, over } = e;
    if (!over || active.id === over.id) return;
    const oldIdx = orderedIds.indexOf(String(active.id));
    const newIdx = orderedIds.indexOf(String(over.id));
    if (oldIdx < 0 || newIdx < 0) return;
    const next = arrayMove(orderedIds, oldIdx, newIdx);
    setOrderedIds(next);
    reorderMutation.mutate(next);
  };

  /* ── Global breakdown (by file type) ─────────────────────── */
  const breakdownSegs: StorageSegment[] = useMemo(() => {
    const s = adminStats ?? {
      total_files: 0,
      total_size: 0,
      images: 0,
      videos: 0,
      audios: 0,
      others: 0,
    };
    const total = Math.max(1, s.total_files);
    const byKind = (count: number) =>
      total > 0 ? Math.round((count / total) * s.total_size) : 0;
    return [
      { kind: "image", label: t("storage.fileKindImage"), count: s.images, bytes: byKind(s.images), color: "hsl(var(--chart-3))" },
      { kind: "video", label: t("storage.fileKindVideo"), count: s.videos, bytes: byKind(s.videos), color: "hsl(var(--chart-2))" },
      { kind: "audio", label: t("storage.fileKindAudio"), count: s.audios, bytes: byKind(s.audios), color: "hsl(var(--chart-1))" },
      { kind: "other", label: t("storage.fileKindOther"), count: s.others, bytes: byKind(s.others), color: "hsl(var(--chart-4))" },
    ];
  }, [adminStats, t]);

  const hasBreakdown = (adminStats?.total_files ?? 0) > 0;

  return (
    <div className="space-y-10">
      <PageHeader
        title={t("storage.title")}
        description={t("storage.description")}
        actions={
          <Button onClick={openCreate}>
            <Plus className="size-4" />
            {t("storage.addStorage")}
          </Button>
        }
      />

      <Section
        title={t("storage.title")}
        description={t("storage.priorityHint")}
        actions={
          <div className="flex items-center gap-2">
            <span className="hidden text-xs font-medium text-muted-foreground sm:inline">
              {t("storage.uploadPolicy")}
            </span>
            <Select value={uploadPolicy} onValueChange={handlePolicyChange}>
              <SelectTrigger
                aria-label={t("storage.uploadPolicy")}
                className="h-8 w-[180px] gap-2 text-xs [&_[data-role=policy-desc]]:hidden [&_[data-role=policy-icon]]:hidden [&_[data-role=policy-label]]:text-xs [&_[data-role=policy-label]]:font-medium"
              >
                <SelectValue />
              </SelectTrigger>
              <SelectContent align="end" className="w-[340px]">
                {UPLOAD_POLICY_OPTIONS.map((opt) => {
                  const Icon = opt.icon;
                  const labelKey =
                    opt.value === "single"
                      ? "storage.policySingle"
                      : opt.value === "primary_fallback"
                        ? "storage.policyPrimaryFallback"
                        : opt.value === "round_robin"
                          ? "storage.policyRoundRobin"
                          : "storage.policyMirror";
                  const descKey =
                    opt.value === "single"
                      ? "storage.policySingleDesc"
                      : opt.value === "primary_fallback"
                        ? "storage.policyPrimaryFallbackDesc"
                        : opt.value === "round_robin"
                          ? "storage.policyRoundRobinDesc"
                          : "storage.policyMirrorDesc";
                  return (
                    <SelectItem key={opt.value} value={opt.value} className="py-2">
                      <div className="flex items-center gap-3">
                        <span
                          data-role="policy-icon"
                          className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted"
                        >
                          <Icon className="size-4" />
                        </span>
                        <div className="flex min-w-0 flex-col gap-0.5">
                          <span
                            data-role="policy-label"
                            className="text-sm font-medium"
                          >
                            {t(labelKey)}
                          </span>
                          <span
                            data-role="policy-desc"
                            className="text-xs text-muted-foreground"
                          >
                            {t(descKey)}
                          </span>
                        </div>
                      </div>
                    </SelectItem>
                  );
                })}
              </SelectContent>
            </Select>
          </div>
        }
      >
        {isStorageLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-64 rounded-xl" />
            ))}
          </div>
        ) : orderedItems.length === 0 ? (
          <EmptyKite
            title={t("storage.noStorage")}
            hint={t("storage.noStorageHint")}
            action={
              <Button size="sm" onClick={openCreate}>
                <Plus className="size-3.5" />
                {t("storage.addStorage")}
              </Button>
            }
          />
        ) : (
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            <SortableContext items={orderedIds} strategy={rectSortingStrategy}>
              <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                {orderedItems.map((cfg) => (
                  <SortableStorageCard
                    key={cfg.id}
                    cfg={cfg}
                    testResult={testResult[cfg.id]}
                    onTest={() => handleTest(cfg.id)}
                    onEdit={() => openEdit(cfg.id)}
                    onDelete={() => deleteMutation.mutate(cfg.id)}
                    onSetDefault={() => setDefaultMutation.mutate(cfg.id)}
                    onToggleActive={(next) => toggleActiveMutation.mutate({ id: cfg.id, active: next })}
                    togglePending={
                      toggleActiveMutation.isPending &&
                      toggleActiveMutation.variables?.id === cfg.id
                    }
                    setDefaultPending={
                      setDefaultMutation.isPending && setDefaultMutation.variables === cfg.id
                    }
                  />
                ))}
              </div>
            </SortableContext>
          </DndContext>
        )}
      </Section>

      {hasBreakdown && (
        <Card className="gap-4 py-5">
          <CardHeader className="flex flex-row items-start justify-between gap-3 [&]:grid-cols-none [&]:grid-rows-none">
            <div className="min-w-0">
              <CardTitle className="text-sm">{t("storage.globalBreakdown")}</CardTitle>
              <CardDescription className="mt-0.5 text-xs">
                {t("storage.breakdownByType")}
              </CardDescription>
            </div>
            <div className="flex shrink-0 items-center gap-2 rounded-md border bg-muted/30 px-2.5 py-1 text-xs">
              <span className="font-semibold tabular-nums">
                {formatSize(adminStats?.total_size ?? 0)}
              </span>
              <span className="size-0.5 rounded-full bg-muted-foreground/40" />
              <span className="tabular-nums text-muted-foreground">
                {(adminStats?.total_files ?? 0).toLocaleString()}
              </span>
            </div>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex h-2 w-full overflow-hidden rounded-full bg-muted/70">
              {breakdownSegs
                .filter((s) => s.bytes > 0)
                .map((seg) => {
                  const total = adminStats?.total_size ?? 1;
                  return (
                    <div
                      key={seg.kind}
                      style={{
                        width: `${(seg.bytes / total) * 100}%`,
                        background: seg.color,
                      }}
                      className="transition-all hover:opacity-80"
                      title={`${seg.label} · ${formatSize(seg.bytes)}`}
                    />
                  );
                })}
            </div>
            <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2 lg:grid-cols-4">
              {breakdownSegs.map((seg) => {
                const total = adminStats?.total_size ?? 0;
                const pct = total > 0 ? Math.round((seg.bytes / total) * 100) : 0;
                return (
                  <div
                    key={seg.kind}
                    className="flex min-w-0 items-center gap-2 rounded-md border bg-muted/20 px-2.5 py-1.5"
                  >
                    <span
                      className="size-1.5 shrink-0 rounded-full"
                      style={{ background: seg.color }}
                    />
                    <span className="truncate text-xs text-foreground">
                      {seg.label}
                    </span>
                    <span className="ml-auto flex shrink-0 items-baseline gap-1.5 tabular-nums">
                      <span className="text-sm font-semibold">
                        {seg.count.toLocaleString()}
                      </span>
                      <span className="text-[10px] text-muted-foreground">
                        {formatSize(seg.bytes)}
                      </span>
                      <span className="text-[10px] font-medium text-muted-foreground/80">
                        {pct}%
                      </span>
                    </span>
                  </div>
                );
              })}
            </div>
          </CardContent>
        </Card>
      )}

      <Dialog open={dialogOpen} onOpenChange={(open) => !open && closeDialog()}>
        <DialogContent className="grid-cols-1 sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editingId ? t("storage.editStorage") : t("storage.addStorage")}
            </DialogTitle>
          </DialogHeader>
          <div className="grid max-h-[60vh] gap-4 overflow-y-auto pr-1">
            <div className="grid gap-2">
              <Label>{t("common.name")}</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder={t("storage.namePlaceholder")}
              />
            </div>

            <div className="grid gap-2">
              <Label>{t("storage.driver")}</Label>
              <Select value={form.driver} onValueChange={(v) => changeDriver(v as Driver)}>
                <SelectTrigger className="h-auto py-2">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {DRIVER_OPTIONS.map((opt) => {
                    const optBrand = getBrandInfo(opt.provider, opt.value);
                    const optTint = optBrand.isBrand ? `${optBrand.color}1A` : undefined;
                    return (
                      <SelectItem key={opt.value} value={opt.value} className="py-2">
                        <div className="flex items-center gap-3">
                          <span
                            className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted"
                            style={optTint ? { backgroundColor: optTint } : undefined}
                          >
                            <BrandIcon
                              provider={opt.provider}
                              driver={opt.value}
                              className={optBrand.isBrand ? "size-4" : "size-4 text-muted-foreground"}
                            />
                          </span>
                          <div className="flex flex-col gap-0.5">
                            <span className="text-sm font-medium">{t(opt.labelKey)}</span>
                            <span className="text-xs text-muted-foreground">
                              {t(opt.descKey)}
                            </span>
                          </div>
                        </div>
                      </SelectItem>
                    );
                  })}
                </SelectContent>
              </Select>
            </div>

            <DriverFields form={form} updateConfig={updateConfig} />

            <div className="grid gap-2">
              <Label>{t("storage.capacityLimit")}</Label>
              <div className="flex gap-2">
                <Input
                  type="number"
                  min="0"
                  step="1"
                  value={form.capacityValue}
                  onChange={(e) =>
                    setForm((f) => ({ ...f, capacityValue: e.target.value }))
                  }
                  placeholder="0"
                  className="flex-1"
                />
                <Select
                  value={form.capacityUnit}
                  onValueChange={(v) =>
                    setForm((f) => ({ ...f, capacityUnit: v as Unit }))
                  }
                >
                  <SelectTrigger className="w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="MB">{t("storage.unitMB")}</SelectItem>
                    <SelectItem value="GB">{t("storage.unitGB")}</SelectItem>
                    <SelectItem value="TB">{t("storage.unitTB")}</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <p className="text-xs text-muted-foreground">{t("storage.capacityLimitHint")}</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog}>
              {t("common.cancel")}
            </Button>
            <Button
              onClick={() => saveMutation.mutate()}
              disabled={!form.name || saveMutation.isPending}
            >
              {saveMutation.isPending ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
 * Storage card (sortable, grid)
 * ═══════════════════════════════════════════════════════════ */
interface StorageCardProps {
  cfg: StorageListItem;
  testResult: "ok" | "fail" | "testing" | undefined;
  onTest: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onSetDefault: () => void;
  onToggleActive: (next: boolean) => void;
  togglePending: boolean;
  setDefaultPending: boolean;
}

function SortableStorageCard(props: StorageCardProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: props.cfg.id,
  });
  const { t } = useI18n();
  const brand = getBrandInfo(props.cfg.provider, props.cfg.driver);
  const vendor = resolveLogoVendor(props.cfg.provider, props.cfg.driver);
  const hasLimit = props.cfg.capacity_limit_bytes > 0;
  const percent = hasLimit
    ? Math.min(100, (props.cfg.used_bytes / props.cfg.capacity_limit_bytes) * 100)
    : 0;
  const nearFull = hasLimit && percent >= 90;
  const fileCount = props.cfg.files_count ?? 0;

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 10 : undefined,
  };

  return (
    <Card
      ref={setNodeRef}
      style={style}
      className="group gap-4 py-5 transition-colors hover:border-foreground/20"
    >
      <CardHeader className="flex flex-row items-start justify-between gap-3 [&]:grid-cols-none [&]:grid-rows-none">
        <div className="flex min-w-0 items-start gap-3">
          <StorageLogo vendor={vendor} size={40} rounded="rounded-lg" />
          <div className="min-w-0 flex-1">
            {/* Single-line title row: long storage names truncate instead of
             * wrapping, so the header keeps a fixed 2-line height across cards
             * regardless of badge combinations. */}
            <div className="flex min-w-0 items-center gap-1.5">
              <CardTitle className="min-w-0 truncate text-sm">
                {props.cfg.name}
              </CardTitle>
              {props.cfg.is_active ? (
                <Badge
                  variant="outline"
                  className="h-4 shrink-0 gap-1 border-emerald-500/20 bg-emerald-500/10 px-1.5 text-[10px] font-medium uppercase tracking-wide text-emerald-700 dark:text-emerald-400"
                >
                  {t("storage.activeBadge")}
                </Badge>
              ) : (
                <Badge
                  variant="outline"
                  className="h-4 shrink-0 px-1.5 text-[10px] font-medium uppercase tracking-wide text-muted-foreground"
                >
                  {t("storage.idleBadge")}
                </Badge>
              )}
              {props.cfg.is_default && (
                <Badge
                  variant="outline"
                  className="h-4 shrink-0 gap-1 border-amber-500/20 bg-amber-500/10 px-1.5 text-[10px] font-medium uppercase tracking-wide text-amber-700 dark:text-amber-400"
                  title={t("storage.defaultStorage")}
                >
                  <Star className="size-2.5 fill-current" />
                  {t("storage.defaultShort")}
                </Badge>
              )}
            </div>
            <CardDescription className="mt-1 flex min-w-0 items-center gap-1.5 font-mono text-xs">
              <span className="shrink-0 rounded bg-muted px-1 py-[1px] text-[9px] font-semibold uppercase tracking-wider not-italic text-muted-foreground">
                {brand.label}
              </span>
              <span className="truncate">P{props.cfg.priority}</span>
            </CardDescription>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button
            type="button"
            {...attributes}
            {...listeners}
            aria-label={t("storage.dragHandle")}
            className="hidden size-7 cursor-grab items-center justify-center rounded-md text-muted-foreground/60 transition-opacity hover:bg-muted hover:text-foreground active:cursor-grabbing sm:flex sm:opacity-0 sm:group-hover:opacity-100"
          >
            <GripVertical className="size-4" />
          </button>
          <Switch
            checked={props.cfg.is_active}
            onCheckedChange={props.onToggleActive}
            disabled={props.togglePending}
          />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="icon-sm" variant="ghost" className="size-7">
                <MoreHorizontal className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-40">
              {!props.cfg.is_default && (
                <>
                  <DropdownMenuItem
                    onClick={props.onSetDefault}
                    disabled={props.setDefaultPending || !props.cfg.is_active}
                  >
                    <Star className="size-4" />
                    {t("storage.setAsDefault")}
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                </>
              )}
              <DropdownMenuItem
                onClick={props.onDelete}
                disabled={props.cfg.is_default}
                variant="destructive"
              >
                <Trash2 className="size-4" />
                {t("common.delete")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>

      <CardContent className="space-y-3">
        <div className="grid grid-cols-3 gap-3 text-center">
          <MiniStat
            label={t("storage.capacityUsed")}
            value={formatSize(props.cfg.used_bytes)}
          />
          <MiniStat
            label={t("storage.capacityTotal")}
            value={
              hasLimit ? (
                formatSize(props.cfg.capacity_limit_bytes)
              ) : (
                <span className="inline-flex items-center justify-center">
                  <InfinityIcon className="size-4" />
                </span>
              )
            }
          />
          <MiniStat
            label={t("storage.filesCount")}
            value={fileCount.toLocaleString()}
          />
        </div>

        <div>
          <div className="flex items-baseline justify-between text-[11px]">
            <span className="text-muted-foreground">{t("storage.usageRate")}</span>
            <span
              className={cn(
                "font-medium tabular-nums",
                nearFull && "text-destructive",
              )}
            >
              {hasLimit ? `${Math.round(percent)}%` : "—"}
            </span>
          </div>
          <Progress
            value={hasLimit ? percent : 0}
            indicatorClassName={nearFull ? "bg-destructive" : undefined}
            className="mt-1.5 h-1.5"
          />
        </div>

        <div className="flex gap-2 pt-1">
          <Button
            variant="outline"
            size="sm"
            className="flex-1"
            onClick={props.onEdit}
          >
            <SettingsIcon className="size-3.5" />
            {t("storage.configure")}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="flex-1"
            onClick={props.onTest}
            disabled={props.testResult === "testing"}
          >
            {props.testResult === "testing" ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : props.testResult === "ok" ? (
              <Check className="size-3.5 text-emerald-600" />
            ) : props.testResult === "fail" ? (
              <AlertCircle className="size-3.5 text-destructive" />
            ) : (
              <RefreshCw className="size-3.5" />
            )}
            {t("storage.sync")}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function MiniStat({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-lg border bg-muted/30 p-2.5">
      <div className="text-[10px] uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-0.5 text-sm font-semibold tabular-nums">{value}</div>
    </div>
  );
}

/* ═══════════════════════════════════════════════════════════
 * Driver-specific form fields
 * ═══════════════════════════════════════════════════════════ */
interface DriverFieldsProps {
  form: StorageForm;
  updateConfig: (key: string, value: string | number) => void;
}

function DriverFields({ form, updateConfig }: DriverFieldsProps) {
  const { t } = useI18n();

  if (form.driver === "local") {
    return (
      <>
        <div className="grid gap-2">
          <Label>{t("storage.rootPath")}</Label>
          <Input
            value={form.config.base_path ?? ""}
            onChange={(e) => updateConfig("base_path", e.target.value)}
            placeholder="./uploads"
          />
        </div>
        <div className="grid gap-2">
          <Label>{t("storage.baseUrl")}</Label>
          <Input
            value={form.config.base_url ?? ""}
            onChange={(e) => updateConfig("base_url", e.target.value)}
            placeholder="https://cdn.example.com"
          />
        </div>
      </>
    );
  }

  if (form.driver === "ftp") {
    return (
      <>
        <div className="grid grid-cols-[1fr_100px] gap-3">
          <div className="grid gap-2">
            <Label>{t("storage.ftpHost")}</Label>
            <Input
              value={form.config.host ?? ""}
              onChange={(e) => updateConfig("host", e.target.value)}
              placeholder="ftp.example.com"
            />
          </div>
          <div className="grid gap-2">
            <Label>{t("storage.ftpPort")}</Label>
            <Input
              type="number"
              value={form.config.port ?? 21}
              onChange={(e) => updateConfig("port", Number(e.target.value) || 21)}
            />
          </div>
        </div>
        <div className="grid grid-cols-2 gap-3">
          <div className="grid gap-2">
            <Label>{t("storage.ftpUsername")}</Label>
            <Input
              value={form.config.username ?? ""}
              onChange={(e) => updateConfig("username", e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label>{t("storage.ftpPassword")}</Label>
            <Input
              type="password"
              value={form.config.password ?? ""}
              onChange={(e) => updateConfig("password", e.target.value)}
            />
          </div>
        </div>
        <div className="grid gap-2">
          <Label>{t("storage.ftpBasePath")}</Label>
          <Input
            value={form.config.base_path ?? ""}
            onChange={(e) => updateConfig("base_path", e.target.value)}
            placeholder="/"
          />
        </div>
        <div className="grid gap-2">
          <Label>{t("storage.baseUrl")}</Label>
          <Input
            value={form.config.base_url ?? ""}
            onChange={(e) => updateConfig("base_url", e.target.value)}
            placeholder="https://files.example.com"
          />
        </div>
      </>
    );
  }

  const endpointHints: Record<Driver, string> = {
    local: "",
    s3: "s3.amazonaws.com",
    oss: "oss-cn-hangzhou.aliyuncs.com",
    cos: "cos.ap-guangzhou.myqcloud.com",
    ftp: "",
  };

  return (
    <>
      <div className="grid grid-cols-2 gap-3">
        <div className="grid gap-2">
          <Label>Endpoint</Label>
          <Input
            value={form.config.endpoint ?? ""}
            onChange={(e) => updateConfig("endpoint", e.target.value)}
            placeholder={endpointHints[form.driver]}
          />
        </div>
        <div className="grid gap-2">
          <Label>Region</Label>
          <Input
            value={form.config.region ?? ""}
            onChange={(e) => updateConfig("region", e.target.value)}
            placeholder={form.driver === "s3" ? "us-east-1" : "cn-hangzhou"}
          />
        </div>
      </div>
      <div className="grid gap-2">
        <Label>Bucket</Label>
        <Input
          value={form.config.bucket ?? ""}
          onChange={(e) => updateConfig("bucket", e.target.value)}
          placeholder="my-bucket"
        />
      </div>
      <div className="grid gap-2">
        <Label>Access Key</Label>
        <Input
          value={form.config.access_key_id ?? ""}
          onChange={(e) => updateConfig("access_key_id", e.target.value)}
        />
      </div>
      <div className="grid gap-2">
        <Label>Secret Key</Label>
        <Input
          type="password"
          value={form.config.secret_access_key ?? ""}
          onChange={(e) => updateConfig("secret_access_key", e.target.value)}
        />
      </div>
      <div className="grid gap-2">
        <Label>{t("storage.cdnDomain")}</Label>
        <Input
          value={form.config.base_url ?? ""}
          onChange={(e) => updateConfig("base_url", e.target.value)}
          placeholder="https://cdn.example.com"
        />
      </div>
    </>
  );
}

function normalizeConfig(form: StorageForm): Record<string, unknown> {
  const c = form.config;
  switch (form.driver) {
    case "local":
      return { base_path: c.base_path ?? "", base_url: c.base_url ?? "" };
    case "ftp":
      return {
        host: c.host ?? "",
        port: c.port ?? 21,
        username: c.username ?? "",
        password: c.password ?? "",
        base_path: c.base_path ?? "/",
        base_url: c.base_url ?? "",
      };
    default:
      return {
        endpoint: c.endpoint ?? "",
        region: c.region ?? "",
        bucket: c.bucket ?? "",
        access_key_id: c.access_key_id ?? "",
        secret_access_key: c.secret_access_key ?? "",
        base_url: c.base_url ?? "",
        force_path_style: c.force_path_style ?? false,
      };
  }
}
