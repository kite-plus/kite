import { useMemo, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Trash2,
  FileText,
  Image as ImageIcon,
  Video,
  Music,
  File as FileIcon,
  Filter,
  Download,
  Copy,
  Check,
  ExternalLink,
  ChevronLeft,
  ChevronRight,
  MoreHorizontal,
  Search,
  X,
} from "lucide-react";
import { toast } from "sonner";

import { adminFileApi, adminStatsApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent } from "@/components/ui/card";
import { Avatar, AvatarFallback } from "@/components/ui/avatar";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getFileTypeLabel } from "@/lib/file-utils";
import { PageHeader } from "@/components/page-header";
import { cn, formatRelativeTime } from "@/lib/utils";

const LIST_PAGE_SIZE = 20;

/* ────────────────────────────────────────────────────────────
 * Types
 * ──────────────────────────────────────────────────────────── */
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

interface AdminStats {
  users: number;
  total_files: number;
  total_size: number;
  images: number;
  videos: number;
  audios: number;
  others: number;
  images_size: number;
  videos_size: number;
  audios_size: number;
  others_size: number;
}

/* ────────────────────────────────────────────────────────────
 * Helpers
 * ──────────────────────────────────────────────────────────── */
function formatBytes(bytes: number) {
  if (!bytes || bytes <= 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
}

function fileExt(name: string): string {
  const m = name.match(/\.([A-Za-z0-9]{1,5})$/);
  return m ? m[1].toLowerCase() : "";
}

/** Short uppercase badge label: extension if present, otherwise a type tag. */
function extLabel(file: FileItem): string {
  const ext = fileExt(file.original_name);
  if (ext) return ext.toUpperCase();
  switch (file.file_type) {
    case "image":
      return "IMG";
    case "video":
      return "VID";
    case "audio":
      return "AUD";
    default:
      return "FILE";
  }
}

/** Stable hue hash for per-tile color tinting. */
function hashHue(seed: string): number {
  let h = 0;
  for (let i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.charCodeAt(i)) | 0;
  }
  return Math.abs(h) % 360;
}

/** Initial letter for avatar fallback — first non-space character uppercased,
 *  falling back to "U" for unknown identifiers. */
function avatarInitial(name: string): string {
  const ch = name.trim().charAt(0);
  return ch ? ch.toUpperCase() : "U";
}

/** Short display name for the owner column: guests keep the translated label,
 *  real user IDs are truncated to the first 8 hex chars. */
function ownerLabel(userId: string, guestLabel: string): string {
  if (!userId || userId === "guest") return guestLabel;
  return userId.slice(0, 8);
}

/* ────────────────────────────────────────────────────────────
 * FileTypeIcon — hue-tinted rounded square with kind icon
 *  Mirrors target pages-user.jsx/FileTypeIcon: bg/border/fg derived
 *  from a single HSL hue.
 * ──────────────────────────────────────────────────────────── */
function FileTypeIcon({ file }: { file: FileItem }) {
  const hue = hashHue(file.id);
  const Icon =
    file.file_type === "image"
      ? ImageIcon
      : file.file_type === "video"
        ? Video
        : file.file_type === "audio"
          ? Music
          : FileText;
  return (
    <div
      className="flex size-7 shrink-0 items-center justify-center rounded-md border"
      style={{
        background: `hsl(${hue} 50% 95%)`,
        borderColor: `hsl(${hue} 40% 85%)`,
      }}
    >
      <Icon className="size-3.5" style={{ color: `hsl(${hue} 50% 40%)` }} />
    </div>
  );
}

/* ────────────────────────────────────────────────────────────
 * StatTile — label row + value + hint (target 1:1)
 * ──────────────────────────────────────────────────────────── */
function StatTile({
  label,
  value,
  hint,
  icon: Icon,
}: {
  label: string;
  value: React.ReactNode;
  hint: string;
  icon: React.ElementType;
}) {
  return (
    <Card className="gap-0 py-0">
      <CardContent className="px-4 py-4">
        <div className="flex items-center justify-between">
          <span className="text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
            {label}
          </span>
          <Icon className="size-3.5 text-muted-foreground" strokeWidth={1.8} />
        </div>
        <div className="mt-2 text-2xl font-semibold tabular-nums tracking-tight">
          {value}
        </div>
        <div className="mt-1 text-[11px] text-muted-foreground">{hint}</div>
      </CardContent>
    </Card>
  );
}

/* ────────────────────────────────────────────────────────────
 * AdminFilesPage
 * ──────────────────────────────────────────────────────────── */
export default function AdminFilesPage() {
  const { t, locale } = useI18n();
  const queryClient = useQueryClient();

  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState("");
  const [fileType, setFileType] = useState("");
  const [detailFile, setDetailFile] = useState<FileItem | null>(null);
  const [copied, setCopied] = useState("");
  const [filterOpen, setFilterOpen] = useState(false);

  // Staged filter values inside the 高级筛选 dialog — only applied on Save
  // so cancelling discards the edits.
  const [draftKeyword, setDraftKeyword] = useState("");
  const [draftType, setDraftType] = useState("");

  /* ── data ─────────────────────────────────────────────── */
  const { data, isLoading } = useQuery({
    queryKey: ["admin-files", page, keyword, fileType, LIST_PAGE_SIZE],
    queryFn: () =>
      adminFileApi
        .list({
          page,
          size: LIST_PAGE_SIZE,
          keyword,
          file_type: fileType,
        })
        .then((r) => r.data.data),
  });

  const { data: stats } = useQuery<AdminStats>({
    queryKey: ["admin-stats"],
    queryFn: () => adminStatsApi.get().then((r) => r.data.data),
    staleTime: 30_000,
  });

  /* ── mutations ────────────────────────────────────────── */
  const deleteMutation = useMutation({
    mutationFn: (id: string) => adminFileApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-files"] });
      queryClient.invalidateQueries({ queryKey: ["admin-stats"] });
      toast.success(t("files.deleteSuccess"));
    },
    onError: () => toast.error(t("files.deleteFailed")),
  });

  /* ── helpers ─────────────────────────────────────────── */
  const copyUrl = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    toast.success(t("files.linkCopied"));
    setCopied(key);
    setTimeout(() => setCopied(""), 1500);
  };

  const openFilterDialog = () => {
    setDraftKeyword(keyword);
    setDraftType(fileType);
    setFilterOpen(true);
  };

  const applyFilter = () => {
    setKeyword(draftKeyword);
    setFileType(draftType);
    setPage(1);
    setFilterOpen(false);
  };

  const clearFilter = () => {
    setDraftKeyword("");
    setDraftType("");
    setKeyword("");
    setFileType("");
    setPage(1);
    setFilterOpen(false);
  };

  /** Minimal CSV export of the visible page — RFC 4180-ish escaping. */
  const exportCsv = () => {
    const items: FileItem[] = data?.items ?? [];
    if (items.length === 0) {
      toast.info(t("files.noFiles"));
      return;
    }
    const header = [
      t("files.fileName"),
      t("files.uploader"),
      t("common.type"),
      t("common.size"),
      t("files.uploadedAt"),
      "URL",
    ];
    const escape = (v: string) => {
      const s = String(v ?? "");
      return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
    };
    const rows = items.map((f) => [
      f.original_name,
      ownerLabel(f.user_id, t("files.guest")),
      extLabel(f),
      formatBytes(f.size_bytes),
      new Date(f.created_at).toISOString(),
      f.url,
    ]);
    const csv = [header, ...rows]
      .map((r) => r.map(escape).join(","))
      .join("\n");
    const blob = new Blob(["\ufeff", csv], { type: "text/csv;charset=utf-8" });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.download = `admin-files-p${page}.csv`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(link.href);
  };

  const types = useMemo(
    () => [
      { value: "", labelKey: "common.all" },
      { value: "image", labelKey: "files.images" },
      { value: "video", labelKey: "files.videos" },
      { value: "audio", labelKey: "files.audio" },
      { value: "file", labelKey: "files.otherFiles" },
    ],
    []
  );

  const totalFiles = data?.total ?? stats?.total_files ?? 0;
  const totalPages = Math.ceil((data?.total ?? 0) / LIST_PAGE_SIZE);

  return (
    <div className="space-y-5">
      {/* ── Header ─────────────────────────────────────── */}
      <PageHeader
        title={t("files.adminTitle")}
        description={t("files.adminSub").replace(
          "{total}",
          totalFiles.toLocaleString()
        )}
        actions={
          <>
            <Button variant="outline" size="sm" onClick={openFilterDialog}>
              <Filter className="size-3.5" strokeWidth={1.8} />
              {t("files.advancedFilter")}
            </Button>
            <Button variant="outline" size="sm" onClick={exportCsv}>
              <Download className="size-3.5" strokeWidth={1.8} />
              {t("files.exportList")}
            </Button>
          </>
        }
      />

      {/* ── 4 Stat tiles: 图片 / 视频 / 音频 / 其它 ───────── */}
      <div className="grid gap-3 sm:grid-cols-4">
        <StatTile
          label={t("files.images")}
          value={(stats?.images ?? 0).toLocaleString()}
          hint={formatBytes(stats?.images_size ?? 0)}
          icon={ImageIcon}
        />
        <StatTile
          label={t("files.videos")}
          value={(stats?.videos ?? 0).toLocaleString()}
          hint={formatBytes(stats?.videos_size ?? 0)}
          icon={Video}
        />
        <StatTile
          label={t("files.audio")}
          value={(stats?.audios ?? 0).toLocaleString()}
          hint={formatBytes(stats?.audios_size ?? 0)}
          icon={Music}
        />
        <StatTile
          label={t("files.otherFiles")}
          value={(stats?.others ?? 0).toLocaleString()}
          hint={t("files.otherDocArchive")}
          icon={FileText}
        />
      </div>

      {/* ── File table (desktop) ──────────────────────── */}
      <Card className="gap-0 py-0">
        <CardContent className="px-0 py-4">
          {isLoading ? (
            <div className="space-y-2 px-5">
              {Array.from({ length: 8 }).map((_, i) => (
                <Skeleton key={i} className="h-10 rounded-md" />
              ))}
            </div>
          ) : (data?.items?.length ?? 0) === 0 ? (
            <div className="flex flex-col items-center py-12 text-center">
              <div className="flex size-12 items-center justify-center rounded-full bg-muted">
                <FileText className="size-5 text-muted-foreground" />
              </div>
              <p className="mt-3 text-sm text-muted-foreground">
                {t("files.noFiles")}
              </p>
            </div>
          ) : (
            <>
              <div className="hidden sm:block">
                <Table>
                  <TableHeader>
                    <TableRow className="hover:bg-transparent">
                      <TableHead className="pl-5">
                        {t("files.fileName")}
                      </TableHead>
                      <TableHead>{t("files.uploader")}</TableHead>
                      <TableHead>{t("common.type")}</TableHead>
                      <TableHead>{t("common.size")}</TableHead>
                      <TableHead>{t("files.accessesCol")}</TableHead>
                      <TableHead>{t("files.uploadedAt")}</TableHead>
                      <TableHead className="w-10 pr-4" />
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {data?.items?.map((file: FileItem) => (
                      <TableRow
                        key={file.id}
                        className="cursor-pointer"
                        onClick={() => setDetailFile(file)}
                      >
                        <TableCell className="pl-5">
                          <div className="flex min-w-0 items-center gap-2.5">
                            <FileTypeIcon file={file} />
                            <span className="truncate font-medium">
                              {file.original_name}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-1.5">
                            <Avatar className="size-[18px]">
                              <AvatarFallback className="text-[9px]">
                                {avatarInitial(
                                  ownerLabel(file.user_id, t("files.guest"))
                                )}
                              </AvatarFallback>
                            </Avatar>
                            <span className="text-muted-foreground">
                              @{ownerLabel(file.user_id, t("files.guest"))}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="font-mono text-[11px] uppercase text-muted-foreground">
                            {extLabel(file)}
                          </span>
                        </TableCell>
                        <TableCell className="tabular-nums">
                          {formatBytes(file.size_bytes)}
                        </TableCell>
                        <TableCell className="tabular-nums text-muted-foreground">
                          —
                        </TableCell>
                        <TableCell className="text-muted-foreground">
                          {formatRelativeTime(file.created_at, locale)}
                        </TableCell>
                        <TableCell className="pr-4 text-right">
                          <div
                            onClick={(e) => e.stopPropagation()}
                            className="inline-flex"
                          >
                            <DropdownMenu>
                              <DropdownMenuTrigger asChild>
                                <Button
                                  size="icon-xs"
                                  variant="ghost"
                                  aria-label={t("files.more")}
                                >
                                  <MoreHorizontal className="size-3.5" />
                                </Button>
                              </DropdownMenuTrigger>
                              <DropdownMenuContent align="end">
                                <DropdownMenuItem
                                  onSelect={(e) => {
                                    e.preventDefault();
                                    copyUrl(file.url, file.id);
                                  }}
                                >
                                  {copied === file.id ? (
                                    <Check className="size-3.5" />
                                  ) : (
                                    <Copy className="size-3.5" />
                                  )}
                                  {t("files.copyLink")}
                                </DropdownMenuItem>
                                <DropdownMenuItem
                                  onSelect={(e) => {
                                    e.preventDefault();
                                    window.open(
                                      file.url,
                                      "_blank",
                                      "noopener,noreferrer"
                                    );
                                  }}
                                >
                                  <ExternalLink className="size-3.5" />
                                  {t("files.openOriginal")}
                                </DropdownMenuItem>
                                <DropdownMenuSeparator />
                                <DropdownMenuItem
                                  variant="destructive"
                                  onSelect={(e) => {
                                    e.preventDefault();
                                    deleteMutation.mutate(file.id);
                                  }}
                                >
                                  <Trash2 className="size-3.5" />
                                  {t("common.delete")}
                                </DropdownMenuItem>
                              </DropdownMenuContent>
                            </DropdownMenu>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              {/* ── Mobile: card list ─────────────────── */}
              <div className="space-y-2 px-3 sm:hidden">
                {data?.items?.map((file: FileItem) => (
                  <div
                    key={file.id}
                    className="flex cursor-pointer items-center gap-3 rounded-lg border bg-card p-3"
                    onClick={() => setDetailFile(file)}
                  >
                    <FileTypeIcon file={file} />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium">
                        {file.original_name}
                      </p>
                      <p className="truncate text-[11px] text-muted-foreground">
                        @{ownerLabel(file.user_id, t("files.guest"))} ·{" "}
                        {formatBytes(file.size_bytes)} ·{" "}
                        {formatRelativeTime(file.created_at, locale)}
                      </p>
                    </div>
                    <div onClick={(e) => e.stopPropagation()}>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button
                            size="icon-xs"
                            variant="ghost"
                            aria-label={t("files.more")}
                          >
                            <MoreHorizontal className="size-3.5" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onSelect={(e) => {
                              e.preventDefault();
                              copyUrl(file.url, file.id);
                            }}
                          >
                            <Copy className="size-3.5" />
                            {t("files.copyLink")}
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            variant="destructive"
                            onSelect={(e) => {
                              e.preventDefault();
                              deleteMutation.mutate(file.id);
                            }}
                          >
                            <Trash2 className="size-3.5" />
                            {t("common.delete")}
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </div>
                  </div>
                ))}
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* ── Pagination ──────────────────────────────── */}
      {data && data.total > LIST_PAGE_SIZE && (
        <div className="flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="icon-sm"
            disabled={page <= 1}
            onClick={() => setPage((p) => p - 1)}
          >
            <ChevronLeft className="size-4" />
          </Button>
          <span className="min-w-15 text-center text-sm text-muted-foreground tabular-nums">
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

      {/* ── Advanced filter dialog ──────────────────── */}
      <Dialog open={filterOpen} onOpenChange={setFilterOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{t("files.advancedFilter")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t("files.searchFiles")}
              </label>
              <div className="relative">
                <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  placeholder={t("files.searchFiles")}
                  value={draftKeyword}
                  onChange={(e) => setDraftKeyword(e.target.value)}
                  className="pl-8"
                />
              </div>
            </div>
            <div>
              <label className="mb-1.5 block text-xs font-medium text-muted-foreground">
                {t("common.type")}
              </label>
              <div className="flex flex-wrap gap-1.5">
                {types.map((tp) => {
                  const active = draftType === tp.value;
                  const Icon =
                    tp.value === "image"
                      ? ImageIcon
                      : tp.value === "video"
                        ? Video
                        : tp.value === "audio"
                          ? Music
                          : tp.value === "file"
                            ? FileIcon
                            : null;
                  return (
                    <button
                      type="button"
                      key={tp.value}
                      onClick={() => setDraftType(tp.value)}
                      className={cn(
                        "inline-flex h-8 items-center gap-1.5 rounded-full border px-3 text-xs font-medium transition-colors",
                        active
                          ? "border-foreground/30 bg-foreground text-background"
                          : "border-border text-muted-foreground hover:border-foreground/30 hover:text-foreground"
                      )}
                    >
                      {Icon && <Icon className="size-3.5" strokeWidth={1.8} />}
                      {t(tp.labelKey)}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>
          <DialogFooter className="gap-2 sm:gap-2">
            <Button variant="outline" size="sm" onClick={clearFilter}>
              <X className="size-3.5" />
              {t("common.clear")}
            </Button>
            <Button size="sm" onClick={applyFilter}>
              <Check className="size-3.5" />
              {t("common.apply")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ── File detail dialog (preserved) ──────────── */}
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
                  <span className="text-xs text-muted-foreground">MIME</span>
                  <p className="break-all text-xs font-medium">
                    {detailFile.mime_type}
                  </p>
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
                    @{ownerLabel(detailFile.user_id, t("files.guest"))}
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
                  ...(detailFile.source_url
                    ? [
                        {
                          label: t("files.sourceUrl"),
                          value: detailFile.source_url,
                        },
                      ]
                    : []),
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
