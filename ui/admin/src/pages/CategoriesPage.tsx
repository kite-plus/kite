import { useState, useMemo, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter, DialogDescription,
} from '@/components/ui/dialog'
import {
  Tooltip, TooltipContent, TooltipProvider, TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Search, Plus, Pencil, Trash2, FolderOpen, FileText, Loader2,
  CornerDownRight, ChevronRight, ChevronDown,
  // ── 预置图标 ──
  Code, Camera, Music, BookOpen, Palette, Globe, Cpu, Gamepad2,
  Heart, Star, Briefcase, GraduationCap, Plane, Coffee, Utensils,
  Dumbbell, Film, Headphones, Mountain, Flower2, Car, Home,
  ShoppingBag, PenTool, Microscope, Rocket, Zap, Sun, Moon, Cloud,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import { useCategoryList, useCreateCategory, useUpdateCategory, useDeleteCategory } from '@/hooks/use-categories'
import type { Category } from '@/types/category'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search as SearchBtn } from '@/components/search'
import { useConfirm } from '@/hooks/use-confirm'
import { CategoryCascader, buildCascaderTree } from '@/components/category-cascader'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

// ─── 预置图标映射 ────────────────────────────────────────────────────

/** 预置的 lucide 图标，key 是存入数据库的名称 */
const PRESET_ICONS: Record<string, LucideIcon> = {
  code: Code, camera: Camera, music: Music, book: BookOpen, palette: Palette,
  globe: Globe, cpu: Cpu, gamepad: Gamepad2, heart: Heart, star: Star,
  briefcase: Briefcase, graduation: GraduationCap, plane: Plane, coffee: Coffee,
  utensils: Utensils, dumbbell: Dumbbell, film: Film, headphones: Headphones,
  mountain: Mountain, flower: Flower2, car: Car, home: Home,
  shopping: ShoppingBag, pen: PenTool, microscope: Microscope, rocket: Rocket,
  zap: Zap, sun: Sun, moon: Moon, cloud: Cloud,
}

/** 获取预置 lucide 图标组件，未命中返回 null */
export function getPresetIcon(name: string): LucideIcon | null {
  return PRESET_ICONS[name] ?? null
}

// ─── Iconify 渲染组件 ────────────────────────────────────────────────

/**
 * 渲染 Iconify 图标（通过 CDN API）
 * 支持 iconify 格式如 mdi:home, ri:code-line
 */
function IconifyIcon({ name, className }: { name: string; className?: string }) {
  // 将 iconify 格式 "mdi:home" 转成 CDN URL
  const [prefix, icon] = name.split(':')
  if (!prefix || !icon) return <FolderOpen className={className} />
  const src = `https://api.iconify.design/${prefix}/${icon}.svg`
  return <img src={src} alt={name} className={cn('inline-block', className)} style={{ width: '1em', height: '1em' }} />
}

/**
 * 通用图标渲染：先查预置 lucide，否则尝试 iconify，都无则默认 FolderOpen
 */
function CategoryIcon({ icon, className }: { icon?: string; className?: string }) {
  if (!icon) return <FolderOpen className={className} />

  // 预置 lucide 图标
  const Preset = getPresetIcon(icon)
  if (Preset) return <Preset className={className} />

  // iconify 格式（含冒号）
  if (icon.includes(':')) return <IconifyIcon name={icon} className={className} />

  // 未匹配则默认
  return <FolderOpen className={className} />
}

// ─── 图标选择器组件 ──────────────────────────────────────────────────

interface IconPickerProps {
  value: string
  onChange: (v: string) => void
}

function IconPicker({ value, onChange }: IconPickerProps) {
  const [customInput, setCustomInput] = useState('')

  return (
    <div className="space-y-3">
      {/* 预置图标网格 */}
      <div>
        <p className="text-xs font-medium text-zinc-500 mb-2">预置图标</p>
        <div className="grid grid-cols-10 gap-1">
          {Object.entries(PRESET_ICONS).map(([name, Icon]) => (
            <TooltipProvider key={name} delayDuration={200}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <button
                    type="button"
                    className={cn(
                      'w-8 h-8 flex items-center justify-center rounded-md border transition-colors cursor-pointer',
                      value === name
                        ? 'border-blue-500 bg-blue-50 dark:bg-blue-500/10 text-blue-600'
                        : 'border-zinc-200 dark:border-zinc-700 hover:bg-zinc-50 dark:hover:bg-zinc-800 text-zinc-600 dark:text-zinc-400'
                    )}
                    onClick={() => onChange(name)}
                  >
                    <Icon className="w-4 h-4" />
                  </button>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="text-xs">{name}</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ))}
        </div>
      </div>

      {/* 自定义 Iconify */}
      <div>
        <p className="text-xs font-medium text-zinc-500 mb-1.5">自定义 Iconify 图标</p>
        <div className="flex gap-2">
          <Input
            placeholder="例如：mdi:home 或 ri:code-line"
            value={customInput}
            onChange={(e) => setCustomInput(e.target.value)}
            className="text-sm border-zinc-200 dark:border-zinc-700 bg-transparent shadow-none rounded-md flex-1"
          />
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="shadow-none shrink-0"
            disabled={!customInput.includes(':')}
            onClick={() => { onChange(customInput.trim()); }}
          >
            应用
          </Button>
        </div>
        {value && value.includes(':') && (
          <div className="mt-2 flex items-center gap-2 text-sm text-zinc-600">
            <span>预览:</span>
            <IconifyIcon name={value} className="w-5 h-5" />
            <span className="font-mono text-xs text-zinc-400">{value}</span>
          </div>
        )}
      </div>
    </div>
  )
}

// ─── 分类表单弹窗 ────────────────────────────────────────────────────

interface CategoryFormData {
  name: string
  slug: string
  description: string
  icon: string
  parentId: string
}

const EMPTY_FORM: CategoryFormData = { name: '', slug: '', description: '', icon: '', parentId: '' }

interface CategoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** 编辑模式时传入已有数据 */
  editingCategory: Category | null
  categories: Category[]
}

function CategoryDialog({ open, onOpenChange, editingCategory, categories }: CategoryDialogProps) {
  const isEdit = !!editingCategory
  const [form, setForm] = useState<CategoryFormData>(EMPTY_FORM)

  const createMutation = useCreateCategory()
  const updateMutation = useUpdateCategory()

  // 弹窗打开时回填数据
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => {
    if (open) {
      if (editingCategory) {
        setForm({
          name: editingCategory.name,
          slug: editingCategory.slug,
          description: editingCategory.description || '',
          icon: editingCategory.icon || '',
          parentId: editingCategory.parentId || '',
        })
      } else {
        setForm(EMPTY_FORM)
      }
    }
  }, [open, editingCategory])

  // 名称自动生成 slug
  function handleNameChange(name: string) {
    setForm((prev) => ({
      ...prev,
      name,
      slug: isEdit ? prev.slug : name.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, ''),
    }))
  }

  function handleSubmit() {
    if (!form.name.trim()) return
    const payload = {
      name: form.name,
      slug: form.slug,
      description: form.description,
      icon: form.icon,
      parent_id: form.parentId || undefined,
    }
    if (isEdit) {
      updateMutation.mutate(
        { id: editingCategory!.id, ...payload },
        {
          onSuccess: () => { onOpenChange(false); toast.success('分类已更新') },
          onError: (err) => toast.error(`更新失败: ${err.message}`),
        },
      )
    } else {
      createMutation.mutate(payload, {
        onSuccess: () => { onOpenChange(false); setForm(EMPTY_FORM); toast.success('分类已创建') },
        onError: (err) => toast.error(`创建失败: ${err.message}`),
      })
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{isEdit ? '编辑分类' : '新建分类'}</DialogTitle>
          <DialogDescription>{isEdit ? '修改分类信息' : '创建一个新的文章分类'}</DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-2">
          {/* 名称 & Slug */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-xs font-medium text-zinc-500 mb-1.5 block">分类名称 *</label>
              <Input value={form.name} onChange={(e) => handleNameChange(e.target.value)} placeholder="例如：前端开发" className="border-zinc-200 dark:border-zinc-700 bg-transparent shadow-none rounded-md" />
            </div>
            <div>
              <label className="text-xs font-medium text-zinc-500 mb-1.5 block">Slug *</label>
              <Input value={form.slug} onChange={(e) => setForm((prev) => ({ ...prev, slug: e.target.value }))} placeholder="例如：frontend" className="border-zinc-200 dark:border-zinc-700 bg-transparent shadow-none rounded-md" />
            </div>
          </div>

          {/* 描述 */}
          <div>
            <label className="text-xs font-medium text-zinc-500 mb-1.5 block">描述</label>
            <Input value={form.description} onChange={(e) => setForm((prev) => ({ ...prev, description: e.target.value }))} placeholder="简要描述此分类的内容范围…" className="border-zinc-200 dark:border-zinc-700 bg-transparent shadow-none rounded-md" />
          </div>

          {/* 父分类 */}
          <div>
            <label className="text-xs font-medium text-zinc-500 mb-1.5 block">父分类</label>
            <CategoryCascader
              options={buildCascaderTree(categories.filter((c) => c.id !== editingCategory?.id))}
              value={form.parentId || null}
              onChange={(id) => setForm((prev) => ({ ...prev, parentId: id || '' }))}
              placeholder="无（顶级分类）"
              allowSelectParent
            />
          </div>

          {/* 图标选择器 */}
          <div>
            <label className="text-xs font-medium text-zinc-500 mb-1.5 block">图标</label>
            <IconPicker value={form.icon} onChange={(v) => setForm((prev) => ({ ...prev, icon: v }))} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" className="shadow-none" onClick={() => onOpenChange(false)}>取消</Button>
          <Button className="bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 shadow-none rounded-md hover:bg-zinc-800 dark:hover:bg-zinc-200" onClick={handleSubmit} disabled={!form.name.trim() || !form.slug.trim() || isPending}>
            {isPending ? '保存中…' : isEdit ? '保存' : '创建'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// ─── 工具函数 ────────────────────────────────────────────────────────

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

function buildTree(flat: Category[]): Category[] {
  const nodeMap = new Map<string, Category>()
  for (const cat of flat) {
    nodeMap.set(cat.id, { ...cat, children: [] })
  }
  const roots: Category[] = []
  for (const cat of flat) {
    const node = nodeMap.get(cat.id)!
    if (cat.parentId) {
      const parent = nodeMap.get(cat.parentId)
      if (parent) {
        parent.children = parent.children || []
        parent.children.push(node)
      } else {
        roots.push(node)
      }
    } else {
      roots.push(node)
    }
  }
  return roots
}

function getDepthLabel(depth: number): string {
  if (depth === 0) return '顶级'
  return `L${depth + 1}`
}

// ─── 递归渲染分类节点 ──────────────────────────────────────────────

interface CategoryNodeProps {
  cat: Category
  depth: number
  onEdit: (cat: Category) => void
  onDelete: (id: string, name: string) => void
}

function CategoryNode({ cat, depth, onEdit, onDelete }: CategoryNodeProps) {
  const [expanded, setExpanded] = useState(true)
  const hasChildren = !!(cat.children?.length)

  return (
    <div>
      <div className="flex items-center gap-3 px-5 py-3 hover:bg-zinc-50 dark:hover:bg-zinc-800/50 transition-colors"
        style={{ paddingLeft: `${20 + depth * 24}px` }}
      >
        {hasChildren ? (
          <button
            className="w-5 h-5 flex items-center justify-center text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 cursor-pointer rounded-sm hover:bg-zinc-100 dark:hover:bg-zinc-700 transition-colors"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
          </button>
        ) : (
          <span className="w-5 h-5 flex items-center justify-center">
            <CornerDownRight className="w-3 h-3 text-zinc-300 dark:text-zinc-600" />
          </span>
        )}

        {/* 图标 */}
        {depth === 0 ? (
          <div className="w-8 h-8 rounded-md bg-zinc-900 dark:bg-zinc-100 flex items-center justify-center shrink-0">
            <CategoryIcon icon={cat.icon} className="w-4 h-4 text-white dark:text-zinc-900" />
          </div>
        ) : (
          <div className="w-7 h-7 rounded-md bg-zinc-100 dark:bg-zinc-800 flex items-center justify-center shrink-0">
            <CategoryIcon icon={cat.icon} className="w-3.5 h-3.5 text-zinc-400" />
          </div>
        )}

        {/* 名称 & slug */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <p className={`text-sm ${depth === 0 ? 'font-semibold text-zinc-950 dark:text-zinc-50' : 'text-zinc-700 dark:text-zinc-300'}`}>
              {cat.name}
            </p>
            <span className="text-[10px] px-1.5 py-0.5 rounded bg-zinc-100 dark:bg-zinc-800 text-zinc-500 font-medium">
              {getDepthLabel(depth)}
            </span>
            {hasChildren && (
              <span className="text-[10px] text-zinc-400">{cat.children!.length} 个子分类</span>
            )}
          </div>
          <p className="text-xs text-zinc-400 font-mono mt-0.5">/{cat.slug}</p>
        </div>

        <span className="text-xs text-zinc-500 flex items-center gap-1"><FileText className="w-3 h-3" /> {cat.postCount} 篇</span>
        <span className="text-xs text-zinc-400 w-24 text-right">{formatDate(cat.updatedAt)}</span>
        <div className="flex gap-0.5">
          <Button variant="ghost" size="icon" className="w-7 h-7" onClick={() => onEdit(cat)}><Pencil className="w-3.5 h-3.5" /></Button>
          {hasChildren ? (
            <TooltipProvider delayDuration={200}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span>
                    <Button variant="ghost" size="icon" className="w-7 h-7 text-zinc-300 dark:text-zinc-600 cursor-not-allowed" disabled>
                      <Trash2 className="w-3.5 h-3.5" />
                    </Button>
                  </span>
                </TooltipTrigger>
                <TooltipContent side="bottom" className="text-xs">存在子分类，无法删除</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          ) : (
            <Button variant="ghost" size="icon" className="w-7 h-7 text-red-500 hover:text-red-600" onClick={() => onDelete(cat.id, cat.name)}><Trash2 className="w-3.5 h-3.5" /></Button>
          )}
        </div>
      </div>

      {hasChildren && expanded && (
        <div className="border-l-2 border-zinc-100 dark:border-zinc-800" style={{ marginLeft: `${32 + depth * 24}px` }}>
          {cat.children!.map((child, childIdx) => (
            <div key={child.id}>
              {childIdx > 0 && <Separator className="bg-zinc-50 dark:bg-zinc-800/30" />}
              <CategoryNode cat={child} depth={depth + 1} onEdit={onEdit} onDelete={onDelete} />
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

// ─── 主页面 ────────────────────────────────────────────────────────

export function CategoriesPage() {
  const [keyword, setKeyword] = useState('')
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingCategory, setEditingCategory] = useState<Category | null>(null)

  const { data: categories, isLoading } = useCategoryList(keyword)
  const deleteMutation = useDeleteCategory()
  const { confirm, ConfirmDialog } = useConfirm()

  const tree = useMemo(() => categories ? buildTree(categories) : [], [categories])

  function handleCreate() {
    setEditingCategory(null)
    setDialogOpen(true)
  }

  function handleEdit(cat: Category) {
    setEditingCategory(cat)
    setDialogOpen(true)
  }

  async function handleDelete(id: string, name: string) {
    if (await confirm({ title: '删除分类', description: `确定删除分类「${name}」吗？此操作不可撤销。`, confirmText: '删除', variant: 'destructive' })) {
      deleteMutation.mutate(id, {
        onError: (err) => toast.error(`删除失败: ${err.message}`),
      })
    }
  }

  return (
    <>
      <ConfirmDialog />
      <CategoryDialog open={dialogOpen} onOpenChange={setDialogOpen} editingCategory={editingCategory} categories={categories || []} />
      <Header fixed>
        <SearchBtn />
        <div className='ml-auto' />
      </Header>
      <Main>
      <div className="flex justify-between items-center mb-6">
        <div>
          <h1 className="text-lg font-semibold text-zinc-950 dark:text-zinc-50">分类管理</h1>
          <p className="text-sm text-zinc-500 mt-1">管理博客文章的分类体系，支持多级分类结构</p>
        </div>
        <Button className="bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 shadow-none rounded-md hover:bg-zinc-800 dark:hover:bg-zinc-200" onClick={handleCreate}>
          <Plus className="w-4 h-4 mr-1.5" /> 新建分类
        </Button>
      </div>

      {/* 搜索 */}
      <div className="mb-4 max-w-sm">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-zinc-400" />
          <Input placeholder="搜索分类名称或 Slug…" value={keyword} onChange={(e) => setKeyword(e.target.value)} className="pl-9 border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none rounded-md" />
        </div>
      </div>

      {isLoading && (
        <div className="text-center py-16">
          <Loader2 className="w-5 h-5 animate-spin text-zinc-400 mx-auto" />
        </div>
      )}

      {!isLoading && categories?.length === 0 && (
        <div className="border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900 py-16">
          <div className="flex flex-col items-center text-center">
            <FolderOpen className="w-10 h-10 text-zinc-300 dark:text-zinc-700 mb-3" />
            <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100">暂无分类</p>
            <p className="text-sm text-zinc-500 mt-1">点击上方按钮创建第一个分类</p>
            <Button variant="outline" size="sm" className="mt-3 shadow-none border-zinc-200 dark:border-zinc-800" onClick={handleCreate}>新建分类</Button>
          </div>
        </div>
      )}

      {tree.length > 0 && (
        <>
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-lg bg-white dark:bg-zinc-900 overflow-hidden">
            {tree.map((root, idx) => (
              <div key={root.id}>
                {idx > 0 && <Separator className="bg-zinc-100 dark:bg-zinc-800" />}
                <CategoryNode cat={root} depth={0} onEdit={handleEdit} onDelete={handleDelete} />
              </div>
            ))}
          </div>
          <p className="text-xs text-zinc-500 mt-4">共 {categories?.length || 0} 个分类</p>
        </>
      )}
    </Main>
    </>
  )
}
