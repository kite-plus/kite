import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Trash2,
  Image,
  Video,
  Music,
  FileText,
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

const typeIcons: Record<string, typeof FileText> = {
  image: Image,
  video: Video,
  audio: Music,
  file: FileText,
};

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
  const [detailFile, setDetailFile] = useState<FileItem | null>(null);
  const [copied, setCopied] = useState("");

  const { data, isLoading } = useQuery({
    queryKey: ["admin-files", page, keyword, fileType],
    queryFn: () =>
      adminFileApi
        .list({ page, size: 20, keyword, file_type: fileType })
        .then((r) => r.data.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminFileApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-files"] });
      toast.success("文件删除成功");
    },
    onError: () => toast.error("文件删除失败"),
  });

  const copyUrl = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    toast.success("链接已复制");
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

  const totalPages = Math.ceil((data?.total ?? 0) / 20);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">
          {t("files.adminTitle")}
        </h1>
        <p className="mt-1 text-sm text-muted-foreground">
          {t("files.adminDescription")}
        </p>
      </div>

      {/* Filters */}
      <div className="flex flex-col flex-wrap items-start gap-3 sm:flex-row sm:items-center">
        <div className="relative w-full sm:flex-1 sm:max-w-xs">
          <Search className="absolute left-2.5 top-2.5 size-4 text-muted-foreground" />
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
        <div className="flex flex-wrap gap-1">
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

      {/* File Table */}
      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 8 }).map((_, i) => (
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
                  <TableHead>{t("files.fileName")}</TableHead>
                  <TableHead>{t("files.uploader")}</TableHead>
                  <TableHead>{t("common.type")}</TableHead>
                  <TableHead>{t("common.size")}</TableHead>
                  <TableHead className="w-20" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {data?.items?.map((file: FileItem) => {
                  const Icon = typeIcons[file.file_type] ?? FileText;
                  return (
                    <TableRow
                      key={file.id}
                      className="cursor-pointer"
                      onClick={() => setDetailFile(file)}
                    >
                      <TableCell>
                        <div className="flex items-center gap-3 min-w-0">
                          {file.file_type === "image" && file.thumb_url ? (
                            <img
                              src={file.thumb_url}
                              className="size-8 shrink-0 rounded-md object-cover"
                              alt=""
                            />
                          ) : (
                            <div className="flex size-8 shrink-0 items-center justify-center rounded-md bg-muted">
                              <Icon className="size-4 text-muted-foreground" />
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
                          {file.file_type}
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

          {/* Mobile cards */}
          <div className="space-y-3 sm:hidden">
            {data?.items?.map((file: FileItem) => {
              const Icon = typeIcons[file.file_type] ?? FileText;
              return (
                <div
                  key={file.id}
                  className="flex cursor-pointer items-center gap-3 rounded-xl border bg-card p-3"
                  onClick={() => setDetailFile(file)}
                >
                  {file.file_type === "image" && file.thumb_url ? (
                    <img
                      src={file.thumb_url}
                      className="size-10 shrink-0 rounded-lg object-cover"
                      alt=""
                    />
                  ) : (
                    <div className="flex size-10 shrink-0 items-center justify-center rounded-lg bg-muted">
                      <Icon className="size-5 text-muted-foreground" />
                    </div>
                  )}
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">
                      {file.original_name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {formatBytes(file.size_bytes)} · {file.file_type}
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

      {/* File Detail Dialog */}
      <Dialog
        open={!!detailFile}
        onOpenChange={(open) => !open && setDetailFile(null)}
      >
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="truncate pr-8">
              {detailFile?.original_name}
            </DialogTitle>
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
                  <span className="text-xs text-muted-foreground">
                    {t("common.type")}
                  </span>
                  <p className="font-medium">{detailFile.mime_type}</p>
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
                    <input
                      type="text"
                      readOnly
                      value={link.value}
                      className="flex-1 truncate bg-transparent text-xs outline-none"
                      onClick={(e) => (e.target as HTMLInputElement).select()}
                    />
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
