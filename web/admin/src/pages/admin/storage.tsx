import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { HardDrive, Trash2, Check, Plus, Pencil, AlertCircle } from "lucide-react";
import { storageApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

interface StorageConfig {
  id: string;
  name: string;
  driver: string;
  config?: Record<string, string>;
  is_default: boolean;
  is_active: boolean;
}

interface StorageForm {
  name: string;
  driver: string;
  config: {
    root_path?: string;
    base_url?: string;
    endpoint?: string;
    region?: string;
    bucket?: string;
    access_key?: string;
    secret_key?: string;
    cdn_domain?: string;
    path_style?: string;
  };
}

const emptyForm: StorageForm = {
  name: "",
  driver: "local",
  config: { root_path: "./uploads", base_url: "" },
};

export default function StoragePage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState<StorageForm>({ ...emptyForm });
  const [testResult, setTestResult] = useState<Record<string, "ok" | "fail" | "testing">>({});

  const { data, isLoading } = useQuery<StorageConfig[]>({
    queryKey: ["storage"],
    queryFn: () => storageApi.list().then((r) => r.data.data),
  });

  const saveMutation = useMutation({
    mutationFn: () => {
      const payload = { name: form.name, driver: form.driver, config: form.config };
      return editingId ? storageApi.update(editingId, payload) : storageApi.create(payload);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["storage"] });
      closeDialog();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => storageApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["storage"] }),
  });

  const handleTest = async (id: string) => {
    setTestResult((prev) => ({ ...prev, [id]: "testing" }));
    try {
      await storageApi.test(id);
      setTestResult((prev) => ({ ...prev, [id]: "ok" }));
      setTimeout(() => setTestResult((prev) => ({ ...prev, [id]: undefined! })), 3000);
    } catch {
      setTestResult((prev) => ({ ...prev, [id]: "fail" }));
      setTimeout(() => setTestResult((prev) => ({ ...prev, [id]: undefined! })), 3000);
    }
  };

  const openCreate = () => {
    setEditingId(null);
    setForm({ ...emptyForm });
    setDialogOpen(true);
  };

  const openEdit = (cfg: StorageConfig) => {
    setEditingId(cfg.id);
    setForm({
      name: cfg.name,
      driver: cfg.driver,
      config: cfg.config ?? {},
    });
    setDialogOpen(true);
  };

  const closeDialog = () => {
    setDialogOpen(false);
    setEditingId(null);
    setForm({ ...emptyForm });
  };

  const updateConfig = (key: string, value: string) =>
    setForm((prev) => ({ ...prev, config: { ...prev.config, [key]: value } }));

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("storage.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("storage.description")}</p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="size-4" />
          {t("storage.addStorage")}
        </Button>
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {Array.from({ length: 2 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-lg" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          {data?.map((cfg) => (
            <div
              key={cfg.id}
              className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 rounded-lg border bg-card p-4"
            >
              <div className="flex items-center gap-3">
                <HardDrive className="size-5 text-muted-foreground shrink-0" />
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="text-sm font-medium">{cfg.name}</p>
                    <Badge variant="outline" className="text-[10px]">{cfg.driver}</Badge>
                    {cfg.is_default && (
                      <Badge variant="secondary" className="text-[10px]">{t("common.default")}</Badge>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {cfg.is_active ? t("common.active") : t("common.inactive")}
                  </p>
                </div>
              </div>
              <div className="flex gap-2 shrink-0">
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => handleTest(cfg.id)}
                  disabled={testResult[cfg.id] === "testing"}
                >
                  {testResult[cfg.id] === "ok" ? (
                    <Check className="size-4 text-green-600" />
                  ) : testResult[cfg.id] === "fail" ? (
                    <AlertCircle className="size-4 text-destructive" />
                  ) : (
                    t("common.test")
                  )}
                </Button>
                <Button size="icon-sm" variant="ghost" onClick={() => openEdit(cfg)}>
                  <Pencil className="size-3.5" />
                </Button>
                <Button
                  size="icon-sm"
                  variant="ghost"
                  onClick={() => deleteMutation.mutate(cfg.id)}
                  disabled={cfg.is_default}
                >
                  <Trash2 className="size-3.5 text-destructive" />
                </Button>
              </div>
            </div>
          ))}

          {data?.length === 0 && (
            <div className="flex flex-col items-center py-16 text-center">
              <HardDrive className="mb-3 size-12 text-muted-foreground/30" />
              <p className="text-sm text-muted-foreground">{t("storage.noStorage")}</p>
            </div>
          )}
        </div>
      )}

      {/* Create/Edit Dialog */}
      <Dialog open={dialogOpen} onOpenChange={(open) => !open && closeDialog()}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle>
              {editingId ? t("storage.editStorage") : t("storage.addStorage")}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4 max-h-[60vh] overflow-y-auto pr-1">
            <div className="space-y-2">
              <Label>{t("common.name")}</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((f) => ({ ...f, name: e.target.value }))}
                placeholder={t("storage.namePlaceholder")}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("storage.driver")}</Label>
              <div className="flex gap-2">
                {["local", "s3"].map((d) => (
                  <Button
                    key={d}
                    type="button"
                    size="sm"
                    variant={form.driver === d ? "default" : "outline"}
                    onClick={() =>
                      setForm((f) => ({
                        ...f,
                        driver: d,
                        config: d === "local" ? { root_path: "./uploads", base_url: "" } : {},
                      }))
                    }
                  >
                    {d === "local" ? t("storage.local") : "S3"}
                  </Button>
                ))}
              </div>
            </div>

            {form.driver === "local" ? (
              <>
                <div className="space-y-2">
                  <Label>{t("storage.rootPath")}</Label>
                  <Input
                    value={form.config.root_path ?? ""}
                    onChange={(e) => updateConfig("root_path", e.target.value)}
                    placeholder="./uploads"
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("storage.baseUrl")}</Label>
                  <Input
                    value={form.config.base_url ?? ""}
                    onChange={(e) => updateConfig("base_url", e.target.value)}
                    placeholder="https://cdn.example.com"
                  />
                </div>
              </>
            ) : (
              <>
                <div className="grid grid-cols-2 gap-3">
                  <div className="space-y-2">
                    <Label>Endpoint</Label>
                    <Input
                      value={form.config.endpoint ?? ""}
                      onChange={(e) => updateConfig("endpoint", e.target.value)}
                      placeholder="s3.amazonaws.com"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>Region</Label>
                    <Input
                      value={form.config.region ?? ""}
                      onChange={(e) => updateConfig("region", e.target.value)}
                      placeholder="us-east-1"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>Bucket</Label>
                  <Input
                    value={form.config.bucket ?? ""}
                    onChange={(e) => updateConfig("bucket", e.target.value)}
                    placeholder="my-bucket"
                  />
                </div>
                <div className="space-y-2">
                  <Label>Access Key</Label>
                  <Input
                    value={form.config.access_key ?? ""}
                    onChange={(e) => updateConfig("access_key", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Secret Key</Label>
                  <Input
                    type="password"
                    value={form.config.secret_key ?? ""}
                    onChange={(e) => updateConfig("secret_key", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>{t("storage.cdnDomain")}</Label>
                  <Input
                    value={form.config.cdn_domain ?? ""}
                    onChange={(e) => updateConfig("cdn_domain", e.target.value)}
                    placeholder="https://cdn.example.com"
                  />
                </div>
              </>
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={closeDialog}>{t("common.cancel")}</Button>
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
