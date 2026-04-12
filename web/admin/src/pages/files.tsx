import { useState, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Upload,
  Trash2,
  Copy,
  Image,
  Video,
  Music,
  FileText,
  Search,
} from "lucide-react";
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

export default function FilesPage() {
  const { t } = useI18n();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [fileType, setFileType] = useState("");
  const [uploadOpen, setUploadOpen] = useState(false);

  const { data, isLoading } = useQuery({
    queryKey: ["files", page, keyword, fileType],
    queryFn: () =>
      fileApi
        .list({ page, size: 20, keyword, file_type: fileType })
        .then((r) => r.data.data),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => fileApi.delete(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["files"] }),
  });

  const uploadMutation = useMutation({
    mutationFn: (file: File) => fileApi.upload(file),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["files"] });
      setUploadOpen(false);
    },
  });

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const files = Array.from(e.dataTransfer.files);
      if (files.length > 0) uploadMutation.mutate(files[0]);
    },
    [uploadMutation]
  );

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) uploadMutation.mutate(files[0]);
  };

  const copyUrl = (url: string) => {
    navigator.clipboard.writeText(url);
  };

  const types = [
    { value: "", labelKey: "common.all" },
    { value: "image", labelKey: "files.images" },
    { value: "video", labelKey: "files.videos" },
    { value: "audio", labelKey: "files.audio" },
    { value: "file", labelKey: "files.otherFiles" },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{t("files.title")}</h1>
          <p className="text-sm text-muted-foreground">
            {t("files.description")}
          </p>
        </div>
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
              className="flex flex-col items-center justify-center rounded-lg border-2 border-dashed p-12 text-center transition-colors hover:border-primary/50"
            >
              <Upload className="mb-3 size-8 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                {t("files.dragOrClick")}
              </p>
              <input
                type="file"
                className="absolute inset-0 cursor-pointer opacity-0"
                onChange={handleFileSelect}
              />
            </div>
            {uploadMutation.isPending && (
              <p className="text-center text-sm text-muted-foreground">
                {t("files.uploading")}
              </p>
            )}
          </DialogContent>
        </Dialog>
      </div>

      {/* Filters */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 max-w-xs">
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
        <div className="flex gap-1">
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

      {/* File List */}
      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <Skeleton key={i} className="h-48 rounded-lg" />
          ))}
        </div>
      ) : (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
            {data?.items?.map(
              (file: {
                id: string;
                original_name: string;
                file_type: string;
                mime_type: string;
                size_bytes: number;
                url: string;
                thumb_url?: string;
                created_at: string;
              }) => {
                const Icon = typeIcons[file.file_type] ?? FileText;
                return (
                  <div
                    key={file.id}
                    className="group relative overflow-hidden rounded-lg border bg-card transition-shadow hover:shadow-md"
                  >
                    {/* Preview area */}
                    <div className="flex h-32 items-center justify-center bg-muted/30">
                      {file.file_type === "image" && file.thumb_url ? (
                        <img
                          src={file.thumb_url}
                          alt={file.original_name}
                          className="h-full w-full object-cover"
                        />
                      ) : (
                        <Icon className="size-10 text-muted-foreground/50" />
                      )}
                    </div>

                    {/* Info */}
                    <div className="p-3">
                      <p className="truncate text-sm font-medium">
                        {file.original_name}
                      </p>
                      <div className="mt-1 flex items-center gap-2">
                        <Badge variant="secondary" className="text-[10px]">
                          {file.file_type}
                        </Badge>
                        <span className="text-xs text-muted-foreground">
                          {formatBytes(file.size_bytes)}
                        </span>
                      </div>
                    </div>

                    {/* Actions overlay */}
                    <div className="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100">
                      <Button
                        size="icon-xs"
                        variant="secondary"
                        onClick={() => copyUrl(file.url)}
                      >
                        <Copy />
                      </Button>
                      <Button
                        size="icon-xs"
                        variant="destructive"
                        onClick={() => deleteMutation.mutate(file.id)}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  </div>
                );
              }
            )}
          </div>

          {data?.items?.length === 0 && (
            <div className="flex flex-col items-center py-16 text-center">
              <FileText className="mb-3 size-12 text-muted-foreground/30" />
              <p className="text-sm text-muted-foreground">{t("files.noFiles")}</p>
            </div>
          )}

          {/* Pagination */}
          {data && data.total > 20 && (
            <div className="flex items-center justify-center gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={page <= 1}
                onClick={() => setPage((p) => p - 1)}
              >
                {t("common.previous")}
              </Button>
              <span className="text-sm text-muted-foreground">
                {t("common.page")} {page} {t("common.of")} {Math.ceil(data.total / 20)}
              </span>
              <Button
                variant="outline"
                size="sm"
                disabled={page >= Math.ceil(data.total / 20)}
                onClick={() => setPage((p) => p + 1)}
              >
                {t("common.next")}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
