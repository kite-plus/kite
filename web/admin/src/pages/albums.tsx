import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  Plus, ChevronLeft, ChevronRight, Folder, ArrowLeft,
  Image as ImageIcon, Video, Music, FileText,
  Copy, ExternalLink, FolderPlus, Pencil, Trash2, FolderOpen, Home,
  ChevronRight as BreadcrumbSep,
} from "lucide-react";
import { useSearchParams } from "react-router-dom";
import {
  DndContext,
  DragOverlay,
  PointerSensor,
  KeyboardSensor,
  useSensor,
  useSensors,
  useDroppable,
  useDraggable,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { CSS } from "@dnd-kit/utilities";

import { albumApi, fileApi } from "@/lib/api";
import { useI18n } from "@/i18n";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Skeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/page-header";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { useAuth } from "@/hooks/use-auth";

// ─── types ────────────────────────────────────────────────────────────────────

interface FolderItem {
  id: string;
  name: string;
  description: string;
  is_public: boolean;
  file_count: number;
  folder_count: number;
  parent_id?: string;
  created_at: string;
}

interface FileItem {
  id: string;
  original_name: string;
  file_type: string;
  size_bytes: number;
  url: string;
  thumb_url?: string;
  album_id?: string;
}

interface FolderListData {
  items: FolderItem[];
  total: number;
  page: number;
  size: number;
  current_folder?: FolderItem;
  ancestors?: FolderItem[];
}

interface FileListData {
  items: FileItem[];
  total: number;
  page: number;
  size: number;
}

type ActiveDragItem =
  | { type: "folder"; item: FolderItem }
  | { type: "file"; item: FileItem };

interface CtxMenu {
  x: number;
  y: number;
  item: { type: "folder"; data: FolderItem } | { type: "file"; data: FileItem };
}

// ─── helpers ─────────────────────────────────────────────────────────────────

function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
}

function FileTypeIcon({ type, className }: { type: string; className?: string }) {
  const cls = cn("text-muted-foreground", className);
  if (type === "image") return <ImageIcon className={cls} />;
  if (type === "video") return <Video className={cls} />;
  if (type === "audio") return <Music className={cls} />;
  return <FileText className={cls} />;
}

// ─── FolderCard ──────────────────────────────────────────────────────────────

function FolderCard({
  folder,
  onOpen,
  onEdit,
  onDelete,
  onContextMenu,
}: {
  folder: FolderItem;
  onOpen: (id: string) => void;
  onEdit: (f: FolderItem) => void;
  onDelete: (id: string) => void;
  onContextMenu: (e: React.MouseEvent, f: FolderItem) => void;
}) {
  const {
    attributes,
    listeners,
    setNodeRef: setDragRef,
    isDragging,
    transform,
  } = useDraggable({ id: `folder-${folder.id}`, data: { type: "folder", item: folder } });

  const { setNodeRef: setDropRef, isOver } = useDroppable({
    id: `folder-${folder.id}`,
    data: { folderId: folder.id },
  });

  const nodeRef = useCallback(
    (el: HTMLElement | null) => {
      setDragRef(el);
      setDropRef(el);
    },
    [setDragRef, setDropRef],
  );

  const style = {
    transform: CSS.Translate.toString(transform),
    opacity: isDragging ? 0.35 : 1,
    zIndex: isDragging ? 50 : undefined,
  };

  return (
    <div
      ref={nodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onContextMenu={(e) => { e.preventDefault(); onContextMenu(e, folder); }}
      onClick={() => onOpen(folder.id)}
      className={cn(
        "group relative flex flex-col items-center gap-2 rounded-xl border bg-card p-4 cursor-pointer select-none",
        "transition-all duration-150 hover:border-primary/40 hover:bg-accent/20 hover:shadow-sm",
        isOver && !isDragging && "border-primary bg-primary/10 ring-2 ring-primary/40 scale-[1.02]",
      )}
    >
      {/* quick-action buttons */}
      {!isDragging && (
        <div
          className="absolute right-2 top-2 flex gap-1 opacity-0 transition-opacity group-hover:opacity-100"
          onClick={(e) => e.stopPropagation()}
        >
          <Button size="icon-xs" variant="ghost" onClick={() => onEdit(folder)}>
            <Pencil className="size-3" />
          </Button>
          <Button size="icon-xs" variant="ghost" onClick={() => onDelete(folder.id)}>
            <Trash2 className="size-3 text-destructive" />
          </Button>
        </div>
      )}

      {/* folder icon SVG */}
      <div className="relative flex h-14 w-18 items-end justify-center">
        <svg viewBox="0 0 80 60" className="h-full w-full drop-shadow-sm" fill="none" xmlns="http://www.w3.org/2000/svg">
          <rect x="2" y="16" width="76" height="42" rx="6" fill={isOver ? "#3b82f6" : "#60a5fa"} />
          <path d="M2 22 L2 16 Q2 14 4 14 L28 14 Q32 14 34 18 L36 22 Z" fill={isOver ? "#2563eb" : "#3b82f6"} />
          <rect x="8" y="26" width="20" height="2.5" rx="1.25" fill="white" opacity="0.3" />
        </svg>
        {isOver && (
          <div className="absolute inset-0 flex items-center justify-center pb-1">
            <span className="text-[9px] font-bold text-white drop-shadow">移入</span>
          </div>
        )}
      </div>

      {/* name */}
      <span className="w-full text-center text-xs font-medium leading-tight line-clamp-2 break-all">
        {folder.name}
      </span>

      {/* meta */}
      <div className="flex gap-2 text-[10px] text-muted-foreground">
        {folder.folder_count > 0 && <span>{folder.folder_count} 夹</span>}
        <span>{folder.file_count} 文件</span>
      </div>

      {folder.is_public && (
        <Badge variant="outline" className="absolute bottom-1 right-1 text-[9px] px-1 py-0">公开</Badge>
      )}
    </div>
  );
}

// ─── FileCard ────────────────────────────────────────────────────────────────

function FileCard({
  file,
  onContextMenu,
}: {
  file: FileItem;
  onContextMenu: (e: React.MouseEvent, f: FileItem) => void;
}) {
  const { attributes, listeners, setNodeRef, isDragging, transform } = useDraggable({
    id: `file-${file.id}`,
    data: { type: "file", item: file },
  });

  const style = {
    transform: CSS.Translate.toString(transform),
    opacity: isDragging ? 0.35 : 1,
    zIndex: isDragging ? 50 : undefined,
  };

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...attributes}
      {...listeners}
      onContextMenu={(e) => { e.preventDefault(); onContextMenu(e, file); }}
      className="group relative flex flex-col items-center gap-1.5 rounded-xl border bg-card p-3 cursor-grab select-none transition-all hover:border-primary/30 hover:shadow-sm active:cursor-grabbing"
    >
      {/* thumbnail or icon */}
      <div className="flex h-14 w-full items-center justify-center overflow-hidden rounded-lg bg-muted">
        {file.thumb_url ? (
          <img
            src={file.thumb_url}
            alt={file.original_name}
            className="h-full w-full object-cover"
            draggable={false}
          />
        ) : (
          <FileTypeIcon type={file.file_type} className="size-7" />
        )}
      </div>

      {/* name */}
      <span className="w-full text-center text-[11px] leading-tight line-clamp-2 break-all">
        {file.original_name}
      </span>

      {/* size */}
      <span className="text-[10px] text-muted-foreground">{formatBytes(file.size_bytes)}</span>
    </div>
  );
}

// ─── Context Menu ─────────────────────────────────────────────────────────────

function ContextMenuOverlay({
  menu,
  onClose,
  actions,
}: {
  menu: CtxMenu;
  onClose: () => void;
  actions: { label: string; icon: React.ReactNode; onClick: () => void; danger?: boolean }[];
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ x: menu.x, y: menu.y });

  useEffect(() => {
    if (ref.current) {
      const rect = ref.current.getBoundingClientRect();
      const vw = window.innerWidth;
      const vh = window.innerHeight;
      setPos({
        x: menu.x + rect.width > vw ? vw - rect.width - 8 : menu.x,
        y: menu.y + rect.height > vh ? vh - rect.height - 8 : menu.y,
      });
    }
  }, [menu.x, menu.y]);

  useEffect(() => {
    const close = () => onClose();
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") onClose(); };
    document.addEventListener("click", close);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("click", close);
      document.removeEventListener("keydown", onKey);
    };
  }, [onClose]);

  return (
    <div
      ref={ref}
      style={{ position: "fixed", left: pos.x, top: pos.y, zIndex: 9999 }}
      className="min-w-44 rounded-xl border bg-popover py-1 shadow-xl"
      onClick={(e) => e.stopPropagation()}
    >
      <div className="border-b px-3 py-1.5 text-xs font-medium text-muted-foreground truncate max-w-52">
        {menu.item.type === "folder" ? "📁" : "📄"} {menu.item.type === "folder" ? menu.item.data.name : menu.item.data.original_name}
      </div>
      {actions.map((action, i) => (
        <button
          key={i}
          className={cn(
            "flex w-full items-center gap-2 px-3 py-1.5 text-sm transition-colors hover:bg-accent",
            action.danger && "text-destructive hover:text-destructive",
          )}
          onClick={() => { action.onClick(); onClose(); }}
        >
          {action.icon}
          {action.label}
        </button>
      ))}
    </div>
  );
}

// ─── Main Page ────────────────────────────────────────────────────────────────

export default function AlbumsPage() {
  const { t } = useI18n();
  const { user } = useAuth();
  const queryClient = useQueryClient();
  const [searchParams, setSearchParams] = useSearchParams();
  const [folderPage, setFolderPage] = useState(1);
  const [filePage, setFilePage] = useState(1);
  const [createOpen, setCreateOpen] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [editingFolder, setEditingFolder] = useState<FolderItem | null>(null);
  const [form, setForm] = useState({ name: "", description: "", is_public: false });
  const [ctxMenu, setCtxMenu] = useState<CtxMenu | null>(null);
  const [activeDrag, setActiveDrag] = useState<ActiveDragItem | null>(null);

  const currentFolderId = searchParams.get("parent_id") || "";

  const getErrorMessage = (error: unknown, fallback: string) => {
    const message = (error as { response?: { data?: { message?: string } } })?.response?.data?.message;
    return message || fallback;
  };

  useEffect(() => {
    setFolderPage(1);
    setFilePage(1);
  }, [currentFolderId]);

  // ── Queries ────────────────────────────────────────────────────────────────

  const { data: foldersData, isLoading: foldersLoading } = useQuery({
    queryKey: ["folders", folderPage, currentFolderId],
    queryFn: () =>
      albumApi
        .list({ page: folderPage, size: 30, ...(currentFolderId ? { parent_id: currentFolderId } : {}) })
        .then((r) => r.data.data as FolderListData),
  });

  const { data: filesData, isLoading: filesLoading } = useQuery({
    queryKey: ["folder-files", filePage, currentFolderId],
    queryFn: () =>
      fileApi
        .list({
          page: filePage,
          size: 30,
          only_self: true,
          ...(currentFolderId ? { album_id: currentFolderId } : { no_album: "true" }),
        })
        .then((r) => r.data.data as FileListData),
    enabled: !!user?.user_id,
  });

  // ── Mutations ──────────────────────────────────────────────────────────────

  const createMutation = useMutation({
    mutationFn: () => albumApi.create({ ...form, parent_id: currentFolderId || undefined }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders"] });
      setCreateOpen(false);
      setForm({ name: "", description: "", is_public: false });
      toast.success(t("albums.createSuccess"));
    },
    onError: () => toast.error(t("albums.createFailed")),
  });

  const updateMutation = useMutation({
    mutationFn: () => albumApi.update(editingFolder!.id, form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders"] });
      setEditOpen(false);
      setEditingFolder(null);
      toast.success(t("albums.updateSuccess"));
    },
    onError: () => toast.error(t("albums.updateFailed")),
  });

  const deleteFolderMutation = useMutation({
    mutationFn: (id: string) => albumApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders"] });
      toast.success(t("albums.deleteSuccess"));
    },
    onError: () => toast.error(t("albums.deleteFailed")),
  });

  const moveFolderMutation = useMutation({
    mutationFn: ({ id, parentId }: { id: string; parentId: string | null }) =>
      albumApi.update(id, { parent_id: parentId ?? "" }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folders"] });
      toast.success(t("albums.moveSuccess"));
    },
    onError: (error) => toast.error(getErrorMessage(error, t("albums.moveFailed"))),
  });

  const moveFileMutation = useMutation({
    mutationFn: ({ id, folderId }: { id: string; folderId: string | null }) =>
      fileApi.move(id, folderId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folder-files"] });
      queryClient.invalidateQueries({ queryKey: ["folders"] });
      toast.success(t("albums.moveSuccess"));
    },
    onError: (error) => toast.error(getErrorMessage(error, t("albums.moveFailed"))),
  });

  const deleteFileMutation = useMutation({
    mutationFn: (id: string) => fileApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["folder-files"] });
      toast.success(t("albums.fileDeleteSuccess"));
    },
    onError: () => toast.error(t("albums.fileDeleteFailed")),
  });

  // ── Navigation ─────────────────────────────────────────────────────────────

  const breadcrumbItems = useMemo(() => {
    const ancestors = foldersData?.ancestors ?? [];
    return [{ id: "", name: t("albums.root") }, ...ancestors];
  }, [foldersData?.ancestors, t]);

  const openFolder = useCallback((folderId: string) => {
    setSearchParams(folderId ? { parent_id: folderId } : {});
  }, [setSearchParams]);

  const goToAncestor = (folderId: string) => {
    setSearchParams(folderId ? { parent_id: folderId } : {});
  };

  const goUp = () => {
    const ancestors = foldersData?.ancestors ?? [];
    if (ancestors.length <= 1) { setSearchParams({}); return; }
    const parent = ancestors[ancestors.length - 2];
    setSearchParams(parent ? { parent_id: parent.id } : {});
  };

  // ── DnD ───────────────────────────────────────────────────────────────────

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor),
  );

  const handleDragStart = ({ active }: DragStartEvent) => {
    setActiveDrag(active.data.current as ActiveDragItem);
  };

  const handleDragEnd = ({ active, over }: DragEndEvent) => {
    setActiveDrag(null);
    if (!over) return;
    const targetFolderId = over.data.current?.folderId as string | undefined;
    if (!targetFolderId) return;

    const activeId = String(active.id);
    if (activeId.startsWith("folder-")) {
      const folderId = activeId.replace("folder-", "");
      if (folderId === targetFolderId) return;
      moveFolderMutation.mutate({ id: folderId, parentId: targetFolderId });
    } else if (activeId.startsWith("file-")) {
      const fileId = activeId.replace("file-", "");
      moveFileMutation.mutate({ id: fileId, folderId: targetFolderId });
    }
  };

  // ── Context menu actions ───────────────────────────────────────────────────

  const openCtxMenu = useCallback((e: React.MouseEvent, item: CtxMenu["item"]) => {
    e.preventDefault();
    setCtxMenu({ x: e.clientX, y: e.clientY, item });
  }, []);

  const folderContextActions = (folder: FolderItem) => [
    {
      label: t("albums.open"),
      icon: <FolderOpen className="size-4" />,
      onClick: () => openFolder(folder.id),
    },
    {
      label: t("albums.newSubfolder"),
      icon: <FolderPlus className="size-4" />,
      onClick: () => {
        openFolder(folder.id);
        setTimeout(() => setCreateOpen(true), 150);
      },
    },
    {
      label: t("albums.rename"),
      icon: <Pencil className="size-4" />,
      onClick: () => {
        setEditingFolder(folder);
        setForm({ name: folder.name, description: folder.description, is_public: folder.is_public });
        setEditOpen(true);
      },
    },
    ...(currentFolderId
      ? [{
          label: t("albums.moveToRoot"),
          icon: <Home className="size-4" />,
          onClick: () => moveFolderMutation.mutate({ id: folder.id, parentId: null }),
        }]
      : []),
    {
      label: t("common.delete"),
      icon: <Trash2 className="size-4" />,
      onClick: () => deleteFolderMutation.mutate(folder.id),
      danger: true,
    },
  ];

  const fileContextActions = (file: FileItem) => [
    {
      label: t("albums.preview"),
      icon: <ExternalLink className="size-4" />,
      onClick: () => window.open(file.url, "_blank"),
    },
    {
      label: t("albums.copyLink"),
      icon: <Copy className="size-4" />,
      onClick: () => {
        navigator.clipboard.writeText(file.url).then(() => toast.success(t("common.copied")));
      },
    },
    ...(file.album_id
      ? [{
          label: t("albums.moveToRoot"),
          icon: <Home className="size-4" />,
          onClick: () => moveFileMutation.mutate({ id: file.id, folderId: null }),
        }]
      : []),
    {
      label: t("albums.deleteFile"),
      icon: <Trash2 className="size-4" />,
      onClick: () => deleteFileMutation.mutate(file.id),
      danger: true,
    },
  ];

  // ── Render ─────────────────────────────────────────────────────────────────

  const isEmpty =
    !foldersLoading &&
    !filesLoading &&
    (foldersData?.items?.length ?? 0) === 0 &&
    (filesData?.items?.length ?? 0) === 0;

  const isLoading = foldersLoading || filesLoading;

  return (
    <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="space-y-6">
        <PageHeader
          title={t("albums.title")}
          description={t("albums.description")}
          actions={
            <>
              {currentFolderId && (
                <Button variant="outline" size="sm" onClick={goUp}>
                  <ArrowLeft className="size-4" />
                  {t("albums.up")}
                </Button>
              )}
              <Button size="sm" onClick={() => setCreateOpen(true)}>
                <Plus className="size-4" />
                {t("albums.newAlbum")}
              </Button>
            </>
          }
        >
          {/* breadcrumb */}
          <nav className="flex flex-wrap items-center gap-1 pt-1 text-sm">
            {breadcrumbItems.map((item, index) => {
              const isLast = index === breadcrumbItems.length - 1;
              return (
                <div key={item.id || "root"} className="flex items-center gap-1">
                  {index > 0 && <BreadcrumbSep className="size-3 text-muted-foreground/50" />}
                  <button
                    type="button"
                    className={cn(
                      "rounded px-1.5 py-0.5 text-xs transition-colors",
                      isLast
                        ? "bg-muted font-medium text-foreground"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground",
                    )}
                    onClick={() => goToAncestor(item.id)}
                  >
                    {index === 0
                      ? <span className="flex items-center gap-1"><Home className="size-3" />{item.name}</span>
                      : item.name}
                  </button>
                </div>
              );
            })}
          </nav>
        </PageHeader>

        {/* Drag hint */}
        {activeDrag && (
          <div className="flex items-center gap-2 rounded-lg border border-primary/30 bg-primary/5 px-3 py-2 text-sm text-primary">
            <Folder className="size-4" />
            <span>拖动到文件夹上方松手即可移动</span>
          </div>
        )}

        {/* Grid */}
        {isLoading ? (
          <div className="grid gap-3 grid-cols-3 sm:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
            {Array.from({ length: 12 }).map((_, i) => (
              <Skeleton key={i} className="h-32 rounded-xl" />
            ))}
          </div>
        ) : isEmpty ? (
          <div className="flex flex-col items-center py-24 text-center">
            <div className="flex size-16 items-center justify-center rounded-full bg-muted">
              <FolderOpen className="size-7 text-muted-foreground" />
            </div>
            <p className="mt-4 text-sm font-medium text-muted-foreground">{t("albums.emptyFolder")}</p>
            <Button className="mt-4" size="sm" onClick={() => setCreateOpen(true)}>
              <Plus className="size-4" />
              {t("albums.newAlbum")}
            </Button>
          </div>
        ) : (
          <>
            {/* folders */}
            {(foldersData?.items?.length ?? 0) > 0 && (
              <div>
                <div className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">文件夹</div>
                <div className="grid gap-3 grid-cols-3 sm:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {foldersData!.items.map((folder) => (
                    <FolderCard
                      key={folder.id}
                      folder={folder}
                      onOpen={openFolder}
                      onEdit={(f) => {
                        setEditingFolder(f);
                        setForm({ name: f.name, description: f.description, is_public: f.is_public });
                        setEditOpen(true);
                      }}
                      onDelete={(id) => deleteFolderMutation.mutate(id)}
                      onContextMenu={(e, f) => openCtxMenu(e, { type: "folder", data: f })}
                    />
                  ))}
                </div>
                {foldersData && foldersData.total > 30 && (
                  <div className="mt-3 flex items-center justify-center gap-2">
                    <Button variant="outline" size="icon-sm" disabled={folderPage <= 1} onClick={() => setFolderPage((p) => p - 1)}>
                      <ChevronLeft className="size-4" />
                    </Button>
                    <span className="text-xs text-muted-foreground">{folderPage} / {Math.ceil(foldersData.total / 30)}</span>
                    <Button variant="outline" size="icon-sm" disabled={folderPage >= Math.ceil(foldersData.total / 30)} onClick={() => setFolderPage((p) => p + 1)}>
                      <ChevronRight className="size-4" />
                    </Button>
                  </div>
                )}
              </div>
            )}

            {/* files */}
            {(filesData?.items?.length ?? 0) > 0 && (
              <div>
                <div className="mb-2 text-xs font-medium uppercase tracking-wider text-muted-foreground">文件</div>
                <div className="grid gap-3 grid-cols-3 sm:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
                  {filesData!.items.map((file) => (
                    <FileCard
                      key={file.id}
                      file={file}
                      onContextMenu={(e, f) => openCtxMenu(e, { type: "file", data: f })}
                    />
                  ))}
                </div>
                {filesData && filesData.total > 30 && (
                  <div className="mt-3 flex items-center justify-center gap-2">
                    <Button variant="outline" size="icon-sm" disabled={filePage <= 1} onClick={() => setFilePage((p) => p - 1)}>
                      <ChevronLeft className="size-4" />
                    </Button>
                    <span className="text-xs text-muted-foreground">{filePage} / {Math.ceil(filesData.total / 30)}</span>
                    <Button variant="outline" size="icon-sm" disabled={filePage >= Math.ceil(filesData.total / 30)} onClick={() => setFilePage((p) => p + 1)}>
                      <ChevronRight className="size-4" />
                    </Button>
                  </div>
                )}
              </div>
            )}
          </>
        )}
      </div>

      {/* DragOverlay */}
      <DragOverlay dropAnimation={null}>
        {activeDrag && (
          <div className="flex items-center gap-2 rounded-lg border bg-popover px-3 py-2 shadow-xl text-sm font-medium opacity-90 pointer-events-none">
            {activeDrag.type === "folder" ? (
              <Folder className="size-4 text-blue-400" />
            ) : (
              <FileTypeIcon type={(activeDrag.item as FileItem).file_type} className="size-4" />
            )}
            <span className="max-w-40 truncate">
              {activeDrag.type === "folder"
                ? (activeDrag.item as FolderItem).name
                : (activeDrag.item as FileItem).original_name}
            </span>
          </div>
        )}
      </DragOverlay>

      {/* Context Menu */}
      {ctxMenu && (
        <ContextMenuOverlay
          menu={ctxMenu}
          onClose={() => setCtxMenu(null)}
          actions={
            ctxMenu.item.type === "folder"
              ? folderContextActions(ctxMenu.item.data as FolderItem)
              : fileContextActions(ctxMenu.item.data as FileItem)
          }
        />
      )}

      {/* Create Dialog */}
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("albums.createAlbum")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("common.name")}</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((a) => ({ ...a, name: e.target.value }))}
                placeholder={t("albums.albumName")}
                onKeyDown={(e) => { if (e.key === "Enter" && form.name) createMutation.mutate(); }}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("albums.albumDescLabel")}</Label>
              <Input
                value={form.description}
                onChange={(e) => setForm((a) => ({ ...a, description: e.target.value }))}
                placeholder={t("albums.albumDesc")}
              />
            </div>
            <div className="rounded-lg border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
              <span className="font-medium text-foreground">{t("albums.current")}</span>
              <span className="ml-2">{foldersData?.current_folder?.name ?? t("albums.root")}</span>
            </div>
            <div className="flex items-center justify-between">
              <Label>{t("albums.publicAlbum")}</Label>
              <Button
                type="button"
                size="sm"
                variant={form.is_public ? "default" : "outline"}
                onClick={() => setForm((a) => ({ ...a, is_public: !a.is_public }))}
              >
                {form.is_public ? t("common.public") : t("albums.private")}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateOpen(false)}>{t("common.cancel")}</Button>
            <Button onClick={() => createMutation.mutate()} disabled={!form.name || createMutation.isPending}>
              {createMutation.isPending ? t("albums.creating") : t("common.create")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Edit Dialog */}
      <Dialog
        open={editOpen}
        onOpenChange={(open) => { if (!open) { setEditOpen(false); setEditingFolder(null); } }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t("albums.editAlbum")}</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>{t("common.name")}</Label>
              <Input
                value={form.name}
                onChange={(e) => setForm((a) => ({ ...a, name: e.target.value }))}
                onKeyDown={(e) => { if (e.key === "Enter" && form.name) updateMutation.mutate(); }}
              />
            </div>
            <div className="space-y-2">
              <Label>{t("albums.albumDescLabel")}</Label>
              <Input
                value={form.description}
                onChange={(e) => setForm((a) => ({ ...a, description: e.target.value }))}
              />
            </div>
            <div className="flex items-center justify-between">
              <Label>{t("albums.publicAlbum")}</Label>
              <Button
                type="button"
                size="sm"
                variant={form.is_public ? "default" : "outline"}
                onClick={() => setForm((a) => ({ ...a, is_public: !a.is_public }))}
              >
                {form.is_public ? t("common.public") : t("albums.private")}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => { setEditOpen(false); setEditingFolder(null); }}>
              {t("common.cancel")}
            </Button>
            <Button onClick={() => updateMutation.mutate()} disabled={!form.name || updateMutation.isPending}>
              {updateMutation.isPending ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </DndContext>
  );
}
