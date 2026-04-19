import { useEffect, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Trash2,
  FileText,
  LayoutGrid,
  List,
  Search,
  Copy,
  Check,
  ExternalLink,
  ChevronLeft,
  ChevronRight,
} from "lucide-react";
import { toast } from "sonner";

import { adminFileApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getFileIconInfo, getFileTypeLabel } from "@/lib/file-utils";
import { useAdaptiveGridPageSize } from "@/hooks/use-adaptive-grid-page-size";
import { PageHeader } from "@/components/page-header";
import { cn } from "@/lib/utils";

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
  user_id: string;
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

export default function AdminFilesPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [fileType, setFileType] = useState("");
  const [viewMode, setViewMode] = useState<"table" | "grid">("table");
  const [detailFile, setDetailFile] = useState<FileItem | null>(null);
  const [copied, setCopied] = useState("");
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
    queryKey: ["admin-files", page, keyword, fileType, pageSize],
    queryFn: () =>
      adminFileApi
        .list({ page, size: pageSize, keyword, file_type: fileType })
        .then((r) => r.data.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminFileApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-files"] });
      toast.success(t("files.deleteSuccess"));
    },
    onError: () => toast.error(t("files.deleteFailed")),
  });

  const copyUrl = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    toast.success(t("files.linkCopied"));
    setCopied(key);
    setTimeout(() => setCopied(""), 1500);
  };

  const types = [
    { value: "", labelKey: "common.all" },
    { value: "image", labelKey: "files.images" },
    { value: "video", labelKey: "files.videos" },
    { value: "audio", labelKey: "files.audio" },
    { value: "file", labelKey: "files.otherFiles" },
  ];

  const totalPages = Math.ceil((data?.total ?? 0) / pageSize);

  return (
    <div className="space-y-8">
      <PageHeader
        title={t("files.adminTitle")}
        description={t("files.adminDescription")}
      />

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative order-1 flex-1 sm:max-w-xs">
          <Search className="absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder={t("files.searchFiles")}
            value={keyword}
            onChange={(e) => {
              setKeyword(e.target.value);
              setPage(1);
            }}
            className="pl-9"
          />
        </div>
        <div className="order-2 inline-flex shrink-0 items-center gap-1 rounded-md border p-0.5 sm:order-3 sm:ml-auto">
          <Button
            size="icon-sm"
            variant={viewMode === "table" ? "secondary" : "ghost"}
            onClick={() => setViewMode("table")}
            title={t("files.listView")}
          >
            <List className="size-4" />
          </Button>
          <Button
            size="icon-sm"
            variant={viewMode === "grid" ? "secondary" : "ghost"}
            onClick={() => setViewMode("grid")}
            title={t("files.gridView")}
          >
            <LayoutGrid className="size-4" />
          </Button>
        </div>
        <div className="order-3 flex w-full min-w-0 flex-wrap gap-1.5 sm:order-2 sm:w-auto">
          {types.map((tp) => (
            <Button
              key={tp.value}
              variant={fileType === tp.value ? "default" : "outline"}
              size="sm"
              onClick={() => {
                setFileType(tp.value);
                setPage(1);
              }}
            >
              {t(tp.labelKey)}
            </Button>
          ))}
        </div>
      </div>

      {/* File List/Grid */}
      {isLoading ? (
        viewMode === "table" ? (
          <div className="space-y-2">
            {Array.from({ length: 8 }).map((_, i) => (
              <Skeleton key={i} className="h-12 rounded-lg" />
            ))}
          </div>
        ) : (
          <div ref={gridRef} className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {Array.from({ length: pageSize }).map((_, i) => (
              <Skeleton key={i} className="h-44 rounded-lg" />
            ))}
          </div>
        )
      ) : (
        <>
          {viewMode === "table" ? (
            <>
              <div className="hidden overflow-hidden rounded-xl border sm:block">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/30 hover:bg-muted/30">
                      <TableHead>{t("files.fileName")}</TableHead>
                      <TableHead>{t("files.uploader")}</TableHead>
                      <TableHead>{t("common.type")}</TableHead>
                      <TableHead>{t("common.size")}</TableHead>
                      <TableHead className="w-20" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data?.items?.map((file: FileItem) => {
                      const fi = getFileIconInfo(file);
                      const Icon = fi.icon;
                      const previewUrl = file.file_type === "image" ? (file.thumb_url || file.url) : null;
                      return (
                        <TableRow
                          key={file.id}
                          className="cursor-pointer"
                          onClick={() => setDetailFile(file)}
                        >
                          <TableCell>
                            <div className="flex items-center gap-3 min-w-0">
                              {previewUrl ? (
                                <img
                                  src={previewUrl}
                                  className="checker-bg size-8 shrink-0 rounded-md object-cover"
                                  alt=""
                                />
                              ) : (
                                <div className={`flex size-8 shrink-0 items-center justify-center rounded-md ${fi.bg}`}>
                                  <Icon className={`size-4 ${fi.color}`} />
                                </div>
                              )}
                              <span className="truncate font-medium">
                                {file.original_name}
                              </span>
                            </div>
                          </TableCell>
                          <TableCell className="text-xs text-muted-foreground">
                            {file.user_id === "guest"
                              ? t("files.guest")
                              : file.user_id.slice(0, 8)}
                          </TableCell>
                          <TableCell>
                            <Badge variant="secondary" className="text-[10px]">
                              {getFileTypeLabel(file)}
                            </Badge>
                          </TableCell>
                          <TableCell className="text-sm text-muted-foreground">
                            {formatBytes(file.size_bytes)}
                          </TableCell>
                          <TableCell>
                            <div className="flex justify-end gap-1">
                              <Button
                                size="icon-xs"
                                variant="ghost"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  copyUrl(file.url, file.id);
                                }}
                              >
                                {copied === file.id ? (
                                  <Check className="size-3" />
                                ) : (
                                  <Copy className="size-3" />
                                )}
                              </Button>
                              <Button
                                size="icon-xs"
                                variant="ghost"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  deleteMutation.mutate(file.id);
                                }}
                              >
                                <Trash2 className="size-3 text-destructive" />
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>

              <div className="space-y-3 sm:hidden">
                {data?.items?.map((file: FileItem) => {
                  const fi = getFileIconInfo(file);
                  const Icon = fi.icon;
                  const previewUrl = file.file_type === "image" ? (file.thumb_url || file.url) : null;
                  return (
                    <div
                      key={file.id}
                      className="flex cursor-pointer items-center gap-3 rounded-xl border bg-card p-3"
                      onClick={() => setDetailFile(file)}
                    >
                      {previewUrl ? (
                        <img
                          src={previewUrl}
                          className="checker-bg size-10 shrink-0 rounded-lg object-cover"
                          alt=""
                        />
                      ) : (
                        <div className={`flex size-10 shrink-0 items-center justify-center rounded-lg ${fi.bg}`}>
                          <Icon className={`size-5 ${fi.color}`} />
                        </div>
                      )}
                      <div className="min-w-0 flex-1">
                        <p className="truncate text-sm font-medium">
                          {file.original_name}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {formatBytes(file.size_bytes)} · {getFileTypeLabel(file)}
                        </p>
                      </div>
                      <Button
                        size="icon-xs"
                        variant="ghost"
                        onClick={(e) => {
                          e.stopPropagation();
                          deleteMutation.mutate(file.id);
                        }}
                      >
                        <Trash2 className="size-3.5 text-destructive" />
                      </Button>
                    </div>
                  );
                })}
              </div>
            </>
          ) : (
            <div ref={gridRef} className="grid gap-4 grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
              {data?.items?.map((file: FileItem) => {
                const fi = getFileIconInfo(file);
                const Icon = fi.icon;
                const previewUrl = file.file_type === "image" ? (file.thumb_url || file.url) : null;
                return (
                  <div
                    key={file.id}
                    className="group relative overflow-hidden rounded-lg border bg-card cursor-pointer"
                    onClick={() => setDetailFile(file)}
                  >
                    <div
                      className={cn(
                        "flex h-28 items-center justify-center",
                        previewUrl ? "checker-bg" : "bg-muted/30",
                      )}
                    >
                      {previewUrl ? (
                        <img
                          src={previewUrl}
                          alt={file.original_name}
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <div className={`flex size-12 items-center justify-center rounded-xl ${fi.bg}`}>
                          <Icon className={`size-6 ${fi.color}`} />
                        </div>
                      )}
                    </div>
                    <div className="p-3">
                      <p className="truncate text-sm font-medium">{file.original_name}</p>
                      <div className="mt-1 flex items-center gap-2">
                        <Badge variant="secondary" className="text-[10px]">{getFileTypeLabel(file)}</Badge>
                        <span className="text-xs text-muted-foreground">{formatBytes(file.size_bytes)}</span>
                      </div>
                      <p className="mt-1 truncate text-[11px] text-muted-foreground">
                        {file.user_id === "guest" ? t("files.guest") : file.user_id.slice(0, 8)}
                      </p>
                    </div>
                    <div className="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <Button
                        size="icon-xs"
                        variant="secondary"
                        onClick={(e) => {
                          e.stopPropagation();
                          copyUrl(file.url, file.id);
                        }}
                      >
                        {copied === file.id ? <Check className="size-3" /> : <Copy className="size-3" />}
                      </Button>
                      <Button
                        size="icon-xs"
                        variant="destructive"
                        onClick={(e) => {
                          e.stopPropagation();
                          deleteMutation.mutate(file.id);
                        }}
                      >
                        <Trash2 className="size-3" />
                      </Button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          {data?.items?.length === 0 && (
            <div className="flex flex-col items-center py-16 text-center">
              <div className="flex size-14 items-center justify-center rounded-full bg-muted">
                <FileText className="size-6 text-muted-foreground" />
              </div>
              <p className="mt-4 text-sm font-medium text-muted-foreground">
                {t("files.noFiles")}
              </p>
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

      {/* File Detail Dialog */}
      <Dialog
        open={!!detailFile}
        onOpenChange={(open) => !open && setDetailFile(null)}
      >
        <DialogContent className="grid-cols-1 sm:max-w-lg">
          <DialogHeader>
            <DialogTitle className="truncate pr-8">
              {detailFile?.original_name}
            </DialogTitle>
          </DialogHeader>
          {detailFile && (
            <div className="space-y-4">
              {detailFile.file_type === "image" && (
                <div className="checker-bg overflow-hidden rounded-lg border">
                  <img
                    src={detailFile.url}
                    alt={detailFile.original_name}
                    className="max-h-64 w-full object-contain"
                  />
                </div>
              )}
              <div className="grid grid-cols-2 gap-3 text-sm">
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t("common.type")}
                  </span>
                  <p className="font-medium">{getFileTypeLabel(detailFile)}</p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    MIME
                  </span>
                  <p className="font-medium text-xs break-all">{detailFile.mime_type}</p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t("common.size")}
                  </span>
                  <p className="font-medium">
                    {formatBytes(detailFile.size_bytes)}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t("files.uploader")}
                  </span>
                  <p className="font-medium">
                    {detailFile.user_id === "guest"
                      ? t("files.guest")
                      : detailFile.user_id.slice(0, 8)}
                  </p>
                </div>
                <div>
                  <span className="text-xs text-muted-foreground">
                    {t("common.date")}
                  </span>
                  <p className="font-medium">
                    {new Date(detailFile.created_at).toLocaleString()}
                  </p>
                </div>
              </div>
              <div className="space-y-2">
                <span className="text-xs font-medium text-muted-foreground">
                  {t("files.linkFormats")}
                </span>
                {[
                  { label: "URL", value: detailFile.url },
                  ...(detailFile.source_url ? [{ label: t("files.sourceUrl"), value: detailFile.source_url }] : []),
                  {
                    label: "Markdown",
                    value: `![${detailFile.original_name}](${detailFile.url})`,
                  },
                  {
                    label: "HTML",
                    value: `<img src="${detailFile.url}" alt="${detailFile.original_name}">`,
                  },
                  {
                    label: "BBCode",
                    value: `[img]${detailFile.url}[/img]`,
                  },
                ].map((link) => (
                  <div
                    key={link.label}
                    className="flex items-center gap-2 rounded-md bg-muted/50 px-3 py-2"
                  >
                    <span className="w-16 shrink-0 text-xs text-muted-foreground">
                      {link.label}
                    </span>
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
                      {copied === link.label ? (
                        <Check className="size-3" />
                      ) : (
                        <Copy className="size-3" />
                      )}
                    </Button>
                  </div>
                ))}
              </div>
              <div className="flex justify-end gap-2">
                <Button size="sm" variant="outline" asChild>
                  <a
                    href={detailFile.url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
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
