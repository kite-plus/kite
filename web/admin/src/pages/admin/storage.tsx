import { useEffect, useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  HardDrive,
  Trash2,
  Check,
  Plus,
  Pencil,
  AlertCircle,
  Layers,
  Repeat,
  Copy as CopyIcon,
  Infinity as InfinityIcon,
  GripVertical,
  Star,
  MoreHorizontal,
  Loader2,
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
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";

import { settingsApi, storageApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { formatSize } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
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
import { toast } from "sonner";

type Driver = "local" | "s3" | "oss" | "cos" | "ftp";
type Unit = "MB" | "GB" | "TB";

interface StorageListItem {
  id: string;
  name: string;
  driver: Driver;
  provider: string;
  capacity_limit_bytes: number;
  used_bytes: number;
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
    mutationFn: () => settingsApi.update({ "storage.upload_policy": uploadPolicy }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["settings"] });
      toast.success(t("storage.savePolicySuccess"));
    },
    onError: () => toast.error(t("storage.savePolicyFailed")),
  });

  const { data, isLoading: isStorageLoading } = useQuery<StorageListItem[]>({
    queryKey: ["storage"],
    queryFn: () => storageApi.list().then((r) => r.data.data),
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
        title={t("storage.uploadPolicy")}
        description={t("storage.uploadPolicyDesc")}
        actions={
          <Button
            variant="outline"
            onClick={() => saveUploadPolicyMutation.mutate()}
            disabled={saveUploadPolicyMutation.isPending}
          >
            {saveUploadPolicyMutation.isPending ? t("settings.saving") : t("settings.saveSettings")}
          </Button>
        }
      >
        <div className="max-w-xl">
          <Select value={uploadPolicy} onValueChange={setUploadPolicy}>
            <SelectTrigger className="h-auto w-full py-2.5">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
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
                      <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted">
                        <Icon className="size-4" />
                      </span>
                      <div className="flex flex-col gap-0.5">
                        <span className="text-sm font-medium">{t(labelKey)}</span>
                        <span className="text-xs text-muted-foreground">{t(descKey)}</span>
                      </div>
                    </div>
                  </SelectItem>
                );
              })}
            </SelectContent>
          </Select>
        </div>
      </Section>

      <Separator />

      <Section
        title={t("storage.title")}
        description={t("storage.priorityHint")}
      >
        {isStorageLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-xl" />
            ))}
          </div>
        ) : (
          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            <SortableContext items={orderedIds} strategy={verticalListSortingStrategy}>
              <div className="space-y-3">
                {orderedItems.map((cfg) => (
                  <SortableStorageRow
                    key={cfg.id}
                    cfg={cfg}
                    testResult={testResult[cfg.id]}
                    onTest={() => handleTest(cfg.id)}
                    onEdit={() => openEdit(cfg.id)}
                    onDelete={() => deleteMutation.mutate(cfg.id)}
                    onSetDefault={() => setDefaultMutation.mutate(cfg.id)}
                    setDefaultPending={
                      setDefaultMutation.isPending && setDefaultMutation.variables === cfg.id
                    }
                  />
                ))}

                {orderedItems.length === 0 && (
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
                )}
              </div>
            </SortableContext>
          </DndContext>
        )}
      </Section>

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

interface StorageListRowProps {
  cfg: StorageListItem;
  testResult: "ok" | "fail" | "testing" | undefined;
  onTest: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onSetDefault: () => void;
  setDefaultPending: boolean;
}

function SortableStorageRow(props: StorageListRowProps) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: props.cfg.id,
  });
  const { t } = useI18n();
  const brand = getBrandInfo(props.cfg.provider, props.cfg.driver);
  const hasLimit = props.cfg.capacity_limit_bytes > 0;
  const percent = hasLimit
    ? Math.min(100, (props.cfg.used_bytes / props.cfg.capacity_limit_bytes) * 100)
    : 0;
  const nearFull = hasLimit && percent >= 90;

  const style: React.CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
    zIndex: isDragging ? 10 : undefined,
  };

  const tintBg = brand.isBrand ? `${brand.color}1A` : undefined;

  return (
    <div
      ref={setNodeRef}
      style={style}
      className="group rounded-xl border bg-card p-4 transition-colors hover:border-foreground/20"
    >
      <div className="flex items-center gap-3">
        <button
          {...attributes}
          {...listeners}
          aria-label={t("storage.dragHandle")}
          className="hidden size-7 shrink-0 cursor-grab items-center justify-center rounded-md text-muted-foreground/60 transition-opacity hover:bg-muted hover:text-foreground active:cursor-grabbing sm:flex sm:opacity-0 sm:group-hover:opacity-100"
        >
          <GripVertical className="size-4" />
        </button>

        <div
          className="flex size-11 shrink-0 items-center justify-center rounded-xl"
          style={tintBg ? { backgroundColor: tintBg } : undefined}
        >
          <BrandIcon
            provider={props.cfg.provider}
            driver={props.cfg.driver}
            className={brand.isBrand ? "size-6" : "size-5 text-muted-foreground"}
          />
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <p className="truncate text-sm font-semibold">{props.cfg.name}</p>
            {props.cfg.is_default && (
              <Badge variant="secondary" className="h-5 gap-1 px-1.5 text-[10px]">
                <Star className="size-2.5 fill-current" />
                {t("storage.defaultStorage")}
              </Badge>
            )}
            {!props.cfg.is_active && (
              <Badge variant="outline" className="h-5 px-1.5 text-[10px]">
                {t("common.inactive")}
              </Badge>
            )}
          </div>
          <p className="mt-0.5 truncate text-xs text-muted-foreground">
            <span>{brand.label}</span>
            <span className="mx-1.5 text-muted-foreground/40">·</span>
            <span className="tabular-nums">P{props.cfg.priority}</span>
          </p>
        </div>

        <div className="flex shrink-0 items-center gap-1">
          <Button
            size="sm"
            variant="outline"
            onClick={props.onTest}
            disabled={props.testResult === "testing"}
            className="h-8 px-3"
          >
            {props.testResult === "testing" ? (
              <Loader2 className="size-3.5 animate-spin" />
            ) : props.testResult === "ok" ? (
              <Check className="size-3.5 text-green-600" />
            ) : props.testResult === "fail" ? (
              <AlertCircle className="size-3.5 text-destructive" />
            ) : (
              <span className="text-xs">{t("common.test")}</span>
            )}
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button size="icon-sm" variant="ghost" className="size-8">
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
              <DropdownMenuItem onClick={props.onEdit}>
                <Pencil className="size-4" />
                {t("common.edit")}
              </DropdownMenuItem>
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
      </div>

      <div className="mt-3 flex items-center gap-3 pl-0 sm:pl-[2.875rem]">
        <Progress
          value={hasLimit ? percent : 0}
          indicatorClassName={nearFull ? "bg-destructive" : undefined}
          className="h-1.5"
        />
        <div className="flex shrink-0 items-center gap-1 text-xs tabular-nums text-muted-foreground">
          <span className={nearFull ? "font-medium text-destructive" : "font-medium text-foreground"}>
            {formatSize(props.cfg.used_bytes)}
          </span>
          <span className="text-muted-foreground/50">/</span>
          {hasLimit ? (
            <span>{formatSize(props.cfg.capacity_limit_bytes)}</span>
          ) : (
            <span className="flex items-center gap-0.5">
              <InfinityIcon className="size-3" />
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

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
