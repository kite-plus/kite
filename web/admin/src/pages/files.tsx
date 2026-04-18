import { useState, useCallback, useEffect, useRef } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Upload,
  Trash2,
  Copy,
  FileText,
  LayoutGrid,
  List,
  Search,
  Check,
  ExternalLink,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { toast } from "sonner";

import { fileApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { Progress } from "@/components/ui/progress";
import { getFileIconInfo, getFileTypeLabel } from "@/lib/file-utils";
import { useAdaptiveGridPageSize } from "@/hooks/use-adaptive-grid-page-size";
import { PageHeader } from "@/components/page-header";

const LIST_PAGE_SIZE = 20;
const DEFAULT_GRID_PAGE_SIZE = 20;

function formatBytes(bytes: number) {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

interface FileItem {
  id: string;
  original_name: string;
  file_type: string;
  mime_type: string;
  size_bytes: number;
  url: string;
  source_url?: string;
  thumb_url?: string;
  created_at: string;
  width?: number;
  height?: number;
}

interface UploadTask {
  id: string;
  file: File;
  progress: number;
  status: "uploading" | "done" | "error";
}

export default function FilesPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [fileType, setFileType] = useState("");
  const [uploadOpen, setUploadOpen] = useState(false);
  const [detailFile, setDetailFile] = useState<FileItem | null>(null);
  const [uploads, setUploads] = useState<UploadTask[]>([]);
  const [copied, setCopied] = useState("");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  const fileInputRef = useRef<HTMLInputElement>(null);
  const {
    gridRef,
    pageSize: adaptiveGridPageSize,
    paginationRef,
  } = useAdaptiveGridPageSize({
    enabled: viewMode === "grid",
    defaultPageSize: DEFAULT_GRID_PAGE_SIZE,
    targetRows: 4,
    minRows: 2,
    maxRows: 4,
  });
  const pageSize = viewMode === "grid" ? adaptiveGridPageSize : LIST_PAGE_SIZE;

  useEffect(() => {
    setPage(1);
  }, [pageSize]);

  const { data, isLoading } = useQuery({
    queryKey: ["files", page, keyword, fileType, pageSize],
    queryFn: () =>
      fileApi
        .list({ page, size: pageSize, keyword, file_type: fileType, only_self: true })
        .then((r) => r.data.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => fileApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
      toast.success("文件删除成功");
    },
    onError: () => toast.error("文件删除失败"),
  });

  const uploadFiles = useCallback(
    (files: FileList | File[]) => {
      const newTasks: UploadTask[] = Array.from(files).map((file) => ({
        id: Math.random().toString(36).slice(2, 10),
        file,
        progress: 0,
        status: "uploading" as const,
      }));
      setUploads((prev) => [...prev, ...newTasks]);

      newTasks.forEach((task) => {
        const formData = new FormData();
        formData.append("file", task.file);

        const xhr = new XMLHttpRequest();
        xhr.open("POST", "/api/v1/upload");
        const token = localStorage.getItem("access_token");
        if (token) xhr.setRequestHeader("Authorization", `Bearer ${token}`);

        xhr.upload.onprogress = (e) => {
          if (e.lengthComputable) {
            const pct = Math.round((e.loaded / e.total) * 100);
            setUploads((prev) =>
              prev.map((u) => (u.id === task.id ? { ...u, progress: pct } : u))
            );
          }
        };
        xhr.onload = () => {
          if (xhr.status === 200) {
            setUploads((prev) =>
              prev.map((u) => (u.id === task.id ? { ...u, status: "done", progress: 100 } : u))
            );
            queryClient.invalidateQueries({ queryKey: ["files"] });
            toast.success(`${task.file.name} 上传成功`);
          } else {
            setUploads((prev) =>
              prev.map((u) => (u.id === task.id ? { ...u, status: "error" } : u))
            );
            toast.error(`${task.file.name} 上传失败`);
          }
        };
        xhr.onerror = () => {
          setUploads((prev) =>
            prev.map((u) => (u.id === task.id ? { ...u, status: "error" } : u))
          );
        };
        xhr.send(formData);
      });
    },
    [queryClient]
  );

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      if (e.dataTransfer.files.length > 0) uploadFiles(e.dataTransfer.files);
    },
    [uploadFiles]
  );

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      uploadFiles(e.target.files);
      e.target.value = "";
    }
  };

  const copyUrl = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    toast.success("链接已复制");
    setCopied(key);
    setTimeout(() => setCopied(""), 1500);
  };

  const clearDone = () => setUploads((prev) => prev.filter((u) => u.status === "uploading"));

  const types = [
    { value: "", labelKey: "common.all" },
    { value: "image", labelKey: "files.images" },
    { value: "video", labelKey: "files.videos" },
    { value: "audio", labelKey: "files.audio" },
    { value: "file", labelKey: "files.otherFiles" },
  ];

  return (
    <div className="space-y-8">
      <PageHeader
        title={t("files.title")}
        description={t("files.description")}
        actions={
          <Dialog open={uploadOpen} onOpenChange={setUploadOpen}>
            <DialogTrigger asChild>
              <Button>
                <Upload className="size-4" />
                {t("common.upload")}
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>{t("files.uploadFile")}</DialogTitle>
              </DialogHeader>
              <div
                onDragOver={(e) => e.preventDefault()}
                onDrop={handleDrop}
                onClick={() => fileInputRef.current?.click()}
                className="flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-10 text-center transition-colors hover:border-primary/50 hover:bg-accent/30"
              >
                <Upload className="mb-3 size-8 text-muted-foreground" />
                <p className="text-sm font-medium">{t("files.dragOrClick")}</p>
                <p className="mt-1 text-xs text-muted-foreground">{t("files.supportedTypes")}</p>
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={handleFileSelect}
                />
              </div>

              {uploads.length > 0 && (
                <div className="max-h-48 space-y-2 overflow-y-auto">
                  <div className="flex items-center justify-between">
                    <span className="text-xs text-muted-foreground">
                      {uploads.filter((u) => u.status === "done").length}/{uploads.length} {t("files.completed")}
                    </span>
                    {uploads.some((u) => u.status !== "uploading") && (
                      <button onClick={clearDone} className="text-xs text-muted-foreground hover:text-foreground">
                        {t("files.clearDone")}
                      </button>
                    )}
                  </div>
                  {uploads.map((task) => (
                    <div key={task.id} className="flex items-center gap-3 rounded-lg border px-3 py-2.5">
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-xs font-medium">{task.file.name}</p>
                        <Progress
                          className="mt-1.5 h-1"
                          value={task.progress}
                          indicatorClassName={
                            task.status === "error"
                              ? "bg-destructive"
                              : task.status === "done"
                                ? "bg-emerald-500"
                                : undefined
                          }
                        />
                      </div>
                      <span className="shrink-0 text-[10px] text-muted-foreground">
                        {task.status === "error" ? (
                          t("files.failed")
                        ) : task.status === "done" ? (
                          <Check className="size-3.5 text-emerald-500" />
                        ) : (
                          `${task.progress}%`
                        )}
                      </span>
                    </div>
                  ))}
                </div>
              )}
            </DialogContent>
          </Dialog>
        }
      />

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative order-1 flex-1 sm:max-w-xs">
          <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("files.searchFiles")}
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1); }}
            className="pl-9"
          />
        </div>
        <div className="order-2 inline-flex shrink-0 items-center gap-1 rounded-md border p-0.5 sm:order-3 sm:ml-auto">
          <Button
            size="icon-sm"
            variant={viewMode === "grid" ? "secondary" : "ghost"}
            onClick={() => setViewMode("grid")}
            title="网格视图"
          >
            <LayoutGrid className="size-4" />
          </Button>
          <Button
            size="icon-sm"
            variant={viewMode === "list" ? "secondary" : "ghost"}
            onClick={() => setViewMode("list")}
            title="列表视图"
          >
            <List className="size-4" />
          </Button>
        </div>
        <div className="order-3 flex w-full min-w-0 flex-wrap gap-1.5 sm:order-2 sm:w-auto">
          {types.map((tp) => (
            <Button
              key={tp.value}
              variant={fileType === tp.value ? "default" : "outline"}
              size="sm"
              onClick={() => { setFileType(tp.value); setPage(1); }}
            >
              {t(tp.labelKey)}
            </Button>
          ))}
        </div>
      </div>

      {/* File List/Grid */}
      {isLoading ? (
        viewMode === "grid" ? (
          <div ref={gridRef} className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {Array.from({ length: pageSize }).map((_, i) => (
              <Skeleton key={i} className="h-48 rounded-lg" />
            ))}
          </div>
        ) : (
          <div className="space-y-2">
            {Array.from({ length: 10 }).map((_, i) => (
              <Skeleton key={i} className="h-16 rounded-lg" />
            ))}
          </div>
        )
      ) : (
        <>
          {viewMode === "grid" ? (
            <div ref={gridRef} className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
              {data?.items?.map((file: FileItem) => {
                const fi = getFileIconInfo(file);
                const Icon = fi.icon;
                const previewUrl = file.file_type === "image" ? (file.thumb_url || file.url) : null;
                return (
                  <div
                    key={file.id}
                    className="group relative cursor-pointer overflow-hidden rounded-lg border bg-card transition-colors hover:border-foreground/20"
                    onClick={() => setDetailFile(file)}
                  >
                    <div className="flex h-32 items-center justify-center bg-muted/30">
                      {previewUrl ? (
                        <img
                          src={previewUrl}
                          alt={file.original_name}
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <div className={`flex size-14 items-center justify-center rounded-xl ${fi.bg}`}>
                          <Icon className={`size-7 ${fi.color}`} />
                        </div>
                      )}
                    </div>
                    <div className="p-3">
                      <p className="truncate text-sm font-medium">{file.original_name}</p>
                      <div className="mt-1 flex items-center gap-2">
                        <Badge variant="secondary" className="text-[10px]">{getFileTypeLabel(file)}</Badge>
                        <span className="text-xs text-muted-foreground">{formatBytes(file.size_bytes)}</span>
                      </div>
                    </div>
                    <div className="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <Button
                        size="icon-xs"
                        variant="secondary"
                        onClick={(e) => { e.stopPropagation(); copyUrl(file.url, file.id); }}
                      >
                        {copied === file.id ? <Check className="size-3" /> : <Copy className="size-3" />}
                      </Button>
                      <Button
                        size="icon-xs"
                        variant="destructive"
                        onClick={(e) => { e.stopPropagation(); deleteMutation.mutate(file.id); }}
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <div className="space-y-2">
              {data?.items?.map((file: FileItem) => {
                const fi = getFileIconInfo(file);
                const Icon = fi.icon;
                const previewUrl = file.file_type === "image" ? (file.thumb_url || file.url) : null;
                return (
                  <div
                    key={file.id}
                    className="flex cursor-pointer items-center gap-3 rounded-lg border bg-card px-3 py-2.5 transition-colors hover:border-foreground/20"
                    onClick={() => setDetailFile(file)}
                  >
                    {previewUrl ? (
                      <img
                        src={previewUrl}
                        alt={file.original_name}
                        className="size-10 shrink-0 rounded-md object-cover"
                      />
                    ) : (
                      <div className={`flex size-10 shrink-0 items-center justify-center rounded-md ${fi.bg}`}>
                        <Icon className={`size-5 ${fi.color}`} />
                      </div>
                    )}
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">{file.original_name}</p>
                      <div className="mt-1 flex items-center gap-2">
                        <Badge variant="secondary" className="text-[10px]">{getFileTypeLabel(file)}</Badge>
                        <span className="text-xs text-muted-foreground">{formatBytes(file.size_bytes)}</span>
                      </div>
                    </div>
                    <div className="flex shrink-0 items-center gap-1">
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        onClick={(e) => { e.stopPropagation(); copyUrl(file.url, file.id); }}
                      >
                        {copied === file.id ? <Check className="size-3" /> : <Copy className="size-3" />}
                      </Button>
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        onClick={(e) => { e.stopPropagation(); deleteMutation.mutate(file.id); }}
                      >
                        <Trash2 className="size-3 text-destructive" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {data?.items?.length === 0 && (
            <div className="flex flex-col items-center rounded-xl border border-dashed py-20 text-center">
              <div className="flex size-14 items-center justify-center rounded-full bg-muted">
                <FileText className="size-6 text-muted-foreground" />
              </div>
              <p className="mt-4 text-sm font-medium text-muted-foreground">{t("files.noFiles")}</p>
            </div>
          )}

          {data && data.total > pageSize && (
            <div ref={paginationRef} className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="icon-sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                <ChevronLeft className="size-4" />
              </Button>
              <span className="min-w-15 text-center text-sm text-muted-foreground">
                {page} / {Math.ceil(data.total / pageSize)}
              </span>
              <Button
                variant="outline"
                size="icon-sm"
                disabled={page >= Math.ceil(data.total / pageSize)}
                onClick={() => setPage((p) => p + 1)}
              >
                <ChevronRight className="size-4" />
              </Button>
            </div>
          )}
        </>
      )}

      {/* File Detail Dialog */}
      <Dialog open={!!detailFile} onOpenChange={(open) => !open && setDetailFile(null)}>
        <DialogContent className="grid-cols-1 sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="truncate pr-8">{detailFile?.original_name}</DialogTitle>
          </DialogHeader>
          {detailFile && (
            <div className="space-y-4">
              {detailFile.file_type === "image" && (
                <div className="overflow-hidden rounded-lg border bg-muted/30">
                  <img
                    src={detailFile.url}
                    alt={detailFile.original_name}
                    className="max-h-64 w-full object-contain"
                  />
                </div>
              )}

              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-xs text-muted-foreground">{t("common.type")}</span>
                  <p className="font-medium">{getFileTypeLabel(detailFile)}</p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">MIME</span>
                  <p className="text-xs font-medium break-all">{detailFile.mime_type}</p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">{t("common.size")}</span>
                  <p className="font-medium">{formatBytes(detailFile.size_bytes)}</p>
                </div>
                {detailFile.width && detailFile.height && (
                  <div>
                    <span className="text-xs text-muted-foreground">{t("files.dimensions")}</span>
                    <p className="font-medium">{detailFile.width} x {detailFile.height}</p>
                  </div>
                )}
                <div>
                  <span className="text-xs text-muted-foreground">{t("common.date")}</span>
                  <p className="font-medium">{new Date(detailFile.created_at).toLocaleString()}</p>
                </div>
              </div>

              <div className="space-y-2">
                <span className="text-xs font-medium text-muted-foreground">{t("files.linkFormats")}</span>
                {[
                  { label: "URL", value: detailFile.url },
                  ...(detailFile.source_url ? [{ label: t("files.sourceUrl"), value: detailFile.source_url }] : []),
                  { label: "Markdown", value: `![${detailFile.original_name}](${detailFile.url})` },
                  { label: "HTML", value: `<img src="${detailFile.url}" alt="${detailFile.original_name}">` },
                  { label: "BBCode", value: `[img]${detailFile.url}[/img]` },
                ].map((link) => (
                  <div key={link.label} className="flex items-center gap-2 rounded-md bg-muted/50 px-3 py-2">
                    <span className="w-16 shrink-0 text-xs text-muted-foreground">{link.label}</span>
                    <div className="min-w-0 flex-1 overflow-x-auto [scrollbar-width:none] [-ms-overflow-style:none] [&::-webkit-scrollbar]:hidden">
                      <code className="block w-max min-w-full whitespace-nowrap bg-transparent text-xs text-foreground/90">
                        {link.value}
                      </code>
                    </div>
                    <Button
                      size="icon-xs"
                      variant="ghost"
                      onClick={() => copyUrl(link.value, link.label)}
                    >
                      {copied === link.label ? <Check className="size-3" /> : <Copy className="size-3" />}
                    </Button>
                  </div>
                ))}
              </div>

              <div className="flex justify-end gap-2">
                <Button size="sm" variant="outline" asChild>
                  <a href={detailFile.url} target="_blank" rel="noopener noreferrer">
                    <ExternalLink className="size-3.5" />
                    {t("files.openOriginal")}
                  </a>
                </Button>
                <Button
                  size="sm"
                  variant="destructive"
                  onClick={() => {
                    deleteMutation.mutate(detailFile.id);
                    setDetailFile(null);
                  }}
                >
                  <Trash2 className="size-3.5" />
                  {t("common.delete")}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
