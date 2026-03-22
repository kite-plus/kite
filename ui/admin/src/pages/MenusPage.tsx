import { useState, useCallback, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Select, SelectContent, SelectItem, SelectTrigger, SelectValue,
} from '@/components/ui/select'
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter,
} from '@/components/ui/dialog'
import {
  Popover, PopoverContent, PopoverTrigger,
} from '@/components/ui/popover'
import { Plus, GripVertical, ExternalLink, Loader2, X, Menu, ChevronRight, Search } from 'lucide-react'
import { Icon } from '@iconify/react'
import { useMenuList, useSaveMenus, type NavMenuItem } from '@/hooks/use-menus'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search as SearchBtn } from '@/components/search'
import { useConfirm } from '@/hooks/use-confirm'

/** 常用 Lucide 图标列表 */
const ICON_LIST: { name: string; id: string }[] = [
  { name: '首页', id: 'lucide:home' },
  { name: '文件', id: 'lucide:file-text' },
  { name: '文章', id: 'lucide:notebook-pen' },
  { name: '页面', id: 'lucide:file' },
  { name: '文件夹', id: 'lucide:folder' },
  { name: '分类', id: 'lucide:folder-tree' },
  { name: '标签', id: 'lucide:tags' },
  { name: '链接', id: 'lucide:link' },
  { name: '外部链接', id: 'lucide:external-link' },
  { name: '用户', id: 'lucide:user' },
  { name: '评论', id: 'lucide:message-square' },
  { name: '邮件', id: 'lucide:mail' },
  { name: '通知', id: 'lucide:bell' },
  { name: '设置', id: 'lucide:settings' },
  { name: '搜索', id: 'lucide:search' },
  { name: '图片', id: 'lucide:image' },
  { name: '相机', id: 'lucide:camera' },
  { name: '音乐', id: 'lucide:music' },
  { name: '视频', id: 'lucide:video' },
  { name: '书籍', id: 'lucide:book-open' },
  { name: '代码', id: 'lucide:code' },
  { name: '终端', id: 'lucide:terminal' },
  { name: '数据库', id: 'lucide:database' },
  { name: '全球', id: 'lucide:globe' },
  { name: '地图', id: 'lucide:map' },
  { name: '导航', id: 'lucide:compass' },
  { name: '火箭', id: 'lucide:rocket' },
  { name: '星星', id: 'lucide:star' },
  { name: '心形', id: 'lucide:heart' },
  { name: '火焰', id: 'lucide:flame' },
  { name: '闪电', id: 'lucide:zap' },
  { name: '钻石', id: 'lucide:gem' },
  { name: '礼物', id: 'lucide:gift' },
  { name: '图钉', id: 'lucide:pin' },
  { name: '购物车', id: 'lucide:shopping-cart' },
  { name: '奖杯', id: 'lucide:trophy' },
  { name: '毕业帽', id: 'lucide:graduation-cap' },
  { name: '信息', id: 'lucide:info' },
  { name: '帮助', id: 'lucide:circle-help' },
  { name: '日历', id: 'lucide:calendar' },
  { name: '时钟', id: 'lucide:clock' },
  { name: '下载', id: 'lucide:download' },
  { name: 'RSS', id: 'lucide:rss' },
  { name: 'GitHub', id: 'mdi:github' },
  { name: 'Twitter', id: 'mdi:twitter' },
  { name: '微信', id: 'mdi:wechat' },
  { name: '哔哩哔哩', id: 'ri:bilibili-fill' },
  { name: 'Telegram', id: 'mdi:telegram' },
]

/**
 * 图标选择器组件
 */
function IconPicker({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [search, setSearch] = useState('')
  const [customInput, setCustomInput] = useState('')

  const filtered = useMemo(() => {
    if (!search) return ICON_LIST
    const q = search.toLowerCase()
    return ICON_LIST.filter(i => i.name.toLowerCase().includes(q) || i.id.toLowerCase().includes(q))
  }, [search])

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="h-9 w-9 flex items-center justify-center rounded-md border border-zinc-200 dark:border-zinc-800 bg-transparent hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
          title="选择图标"
        >
          {value ? (
            <Icon icon={value} className="w-4 h-4 text-zinc-700 dark:text-zinc-300" />
          ) : (
            <Plus className="w-3.5 h-3.5 text-zinc-300 dark:text-zinc-600" />
          )}
        </button>
      </PopoverTrigger>
      <PopoverContent className="w-[320px] p-0" align="start">
        {/* 搜索栏 */}
        <div className="flex items-center gap-2 px-3 py-2 border-b border-zinc-100 dark:border-zinc-800">
          <Search className="w-3.5 h-3.5 text-zinc-400 shrink-0" />
          <input
            className="flex-1 text-sm bg-transparent outline-none placeholder:text-zinc-400"
            placeholder="搜索图标…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {/* 图标网格 */}
        <div className="p-2 max-h-[240px] overflow-y-auto">
          <div className="grid grid-cols-8 gap-0.5">
            {filtered.map((icon) => (
              <button
                key={icon.id}
                type="button"
                className={`w-8 h-8 flex items-center justify-center rounded-md hover:bg-zinc-100 dark:hover:bg-zinc-800 transition-colors ${
                  value === icon.id ? 'bg-zinc-100 dark:bg-zinc-800 ring-1 ring-zinc-300 dark:ring-zinc-600' : ''
                }`}
                title={`${icon.name} (${icon.id})`}
                onClick={() => onChange(icon.id)}
              >
                <Icon icon={icon.id} className="w-4 h-4" />
              </button>
            ))}
          </div>
          {filtered.length === 0 && (
            <p className="text-xs text-zinc-400 text-center py-4">未找到匹配图标</p>
          )}
        </div>

        {/* 自定义输入 */}
        <div className="flex items-center gap-2 px-3 py-2 border-t border-zinc-100 dark:border-zinc-800">
          <input
            className="flex-1 text-xs bg-transparent outline-none placeholder:text-zinc-400 font-mono"
            placeholder="自定义：lucide:home"
            value={customInput}
            onChange={(e) => setCustomInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && customInput.trim()) {
                onChange(customInput.trim())
                setCustomInput('')
              }
            }}
          />
          {value && (
            <button
              type="button"
              className="text-[11px] text-zinc-400 hover:text-red-500 transition-colors whitespace-nowrap"
              onClick={() => onChange('')}
            >
              清除
            </button>
          )}
        </div>
      </PopoverContent>
    </Popover>
  )
}

/**
 * 菜单管理页面 — 支持二级菜单 + Iconify 图标
 */
export function MenusPage() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingPath, setEditingPath] = useState<number[] | null>(null)
  const [formData, setFormData] = useState<NavMenuItem>({ title: '', url: '', icon: '', openInNewTab: false })
  const [parentSelect, setParentSelect] = useState<string>('none')
  const [dragIndex, setDragIndex] = useState<number | null>(null)
  const [dragChildInfo, setDragChildInfo] = useState<{ parent: number; child: number } | null>(null)

  const { data: menus, isLoading } = useMenuList()
  const saveMutation = useSaveMenus()
  const { confirm, ConfirmDialog } = useConfirm()

  function cloneMenus(): NavMenuItem[] {
    return menus ? JSON.parse(JSON.stringify(menus)) : []
  }

  const saveMenus = useCallback((newMenus: NavMenuItem[]) => {
    saveMutation.mutate(newMenus)
  }, [saveMutation])

  function openCreateDialog() {
    setEditingPath(null)
    setFormData({ title: '', url: '', icon: '', openInNewTab: false })
    setParentSelect('none')
    setDialogOpen(true)
  }

  function openCreateChildDialog(parentIndex: number) {
    setEditingPath(null)
    setFormData({ title: '', url: '', icon: '', openInNewTab: false })
    setParentSelect(String(parentIndex))
    setDialogOpen(true)
  }

  function openEditDialog(index: number) {
    if (!menus) return
    const m = menus[index]
    setEditingPath([index])
    setFormData({ title: m.title, url: m.url, icon: m.icon || '', openInNewTab: m.openInNewTab })
    setParentSelect('none')
    setDialogOpen(true)
  }

  function openEditChildDialog(parentIndex: number, childIndex: number) {
    if (!menus) return
    const child = menus[parentIndex].children?.[childIndex]
    if (!child) return
    setEditingPath([parentIndex, childIndex])
    setFormData({ title: child.title, url: child.url, icon: child.icon || '', openInNewTab: child.openInNewTab })
    setParentSelect(String(parentIndex))
    setDialogOpen(true)
  }

  const isEditing = editingPath !== null

  function buildItem(): NavMenuItem {
    return {
      title: formData.title,
      url: formData.url,
      icon: formData.icon || undefined,
      openInNewTab: formData.openInNewTab,
    }
  }

  function handleSubmit() {
    if (!formData.title.trim() || !formData.url.trim()) return
    const current = cloneMenus()
    const isChild = parentSelect !== 'none'
    const parentIdx = isChild ? parseInt(parentSelect) : -1
    const item = buildItem()

    if (!isEditing) {
      if (isChild) {
        if (!current[parentIdx].children) current[parentIdx].children = []
        current[parentIdx].children!.push(item)
      } else {
        current.push({ ...item, children: [] })
      }
    } else if (editingPath!.length === 1) {
      const idx = editingPath![0]
      if (isChild && parentIdx !== idx) {
        current.splice(idx, 1)
        const actualParent = parentIdx > idx ? parentIdx - 1 : parentIdx
        if (!current[actualParent].children) current[actualParent].children = []
        current[actualParent].children!.push(item)
      } else {
        current[idx] = { ...current[idx], ...item }
      }
    } else {
      const [pi, ci] = editingPath!
      if (parentSelect === 'none') {
        current[pi].children?.splice(ci, 1)
        current.push({ ...item, children: [] })
      } else if (parentIdx === pi) {
        current[pi].children![ci] = item
      } else {
        current[pi].children?.splice(ci, 1)
        if (!current[parentIdx].children) current[parentIdx].children = []
        current[parentIdx].children!.push(item)
      }
    }

    saveMenus(current)
    setDialogOpen(false)
  }

  async function handleDelete(e: React.MouseEvent, index: number) {
    e.stopPropagation()
    const item = menus?.[index]
    const hasChildren = item?.children && item.children.length > 0
    if (await confirm({
      title: '删除菜单项',
      description: hasChildren ? `"${item?.title}" 下有 ${item?.children?.length} 个子菜单，将一并删除。` : '确定删除此菜单项吗？',
      confirmText: '删除',
      variant: 'destructive',
    })) {
      const current = cloneMenus()
      current.splice(index, 1)
      saveMenus(current)
    }
  }

  async function handleDeleteChild(e: React.MouseEvent, parentIndex: number, childIndex: number) {
    e.stopPropagation()
    if (await confirm({ title: '删除子菜单', description: '确定删除此子菜单项吗？', confirmText: '删除', variant: 'destructive' })) {
      const current = cloneMenus()
      current[parentIndex].children?.splice(childIndex, 1)
      saveMenus(current)
    }
  }

  function handleDragStart(index: number) { setDragIndex(index) }
  function handleDragOver(e: React.DragEvent, index: number) {
    e.preventDefault()
    if (dragIndex === null || dragIndex === index || !menus) return
    const current = cloneMenus()
    const [item] = current.splice(dragIndex, 1)
    current.splice(index, 0, item)
    saveMenus(current)
    setDragIndex(index)
  }
  function handleDragEnd() { setDragIndex(null) }

  function handleChildDragStart(parentIndex: number, childIndex: number) {
    setDragChildInfo({ parent: parentIndex, child: childIndex })
  }
  function handleChildDragOver(e: React.DragEvent, parentIndex: number, childIndex: number) {
    e.preventDefault()
    e.stopPropagation()
    if (!dragChildInfo || dragChildInfo.parent !== parentIndex || dragChildInfo.child === childIndex || !menus) return
    const current = cloneMenus()
    const children = current[parentIndex].children || []
    const [item] = children.splice(dragChildInfo.child, 1)
    children.splice(childIndex, 0, item)
    current[parentIndex].children = children
    saveMenus(current)
    setDragChildInfo({ parent: parentIndex, child: childIndex })
  }
  function handleChildDragEnd() { setDragChildInfo(null) }

  function getParentOptions() {
    if (!menus) return []
    return menus
      .map((m, i) => ({ title: m.title, icon: m.icon, index: i }))
      .filter(({ index }) => !(editingPath?.length === 1 && editingPath[0] === index))
  }

  return (
    <>
      <ConfirmDialog />
      <Header fixed>
        <SearchBtn />
        <div className='ml-auto' />
      </Header>
      <Main>
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-lg font-semibold text-zinc-950 dark:text-zinc-50">菜单管理</h1>
            <p className="text-sm text-zinc-500 mt-1">配置前台导航栏菜单，支持二级下拉菜单</p>
          </div>
          <Button className="bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 shadow-none rounded-md hover:bg-zinc-800 dark:hover:bg-zinc-200" onClick={openCreateDialog}>
            <Plus className="w-4 h-4 mr-1.5" /> 添加菜单项
          </Button>
        </div>

        {isLoading && <div className="text-center py-16"><Loader2 className="w-5 h-5 animate-spin text-zinc-400 mx-auto" /></div>}

        {!isLoading && (!menus || menus.length === 0) && (
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900 py-16">
            <div className="flex flex-col items-center text-center">
              <Menu className="w-10 h-10 text-zinc-300 dark:text-zinc-700 mb-3" />
              <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100">暂无自定义菜单</p>
              <p className="text-sm text-zinc-500 mt-1">前台将显示默认导航（首页 + 页面 + 友链）</p>
              <Button variant="outline" size="sm" className="mt-3 shadow-none border-zinc-200 dark:border-zinc-800" onClick={openCreateDialog}>添加菜单项</Button>
            </div>
          </div>
        )}

        {menus && menus.length > 0 && (
          <div className="border border-zinc-200 dark:border-zinc-800 rounded-md bg-white dark:bg-zinc-900">
            {menus.map((item, index) => (
              <div key={`menu-${index}`}>
                <div
                  draggable
                  onDragStart={() => handleDragStart(index)}
                  onDragOver={(e) => handleDragOver(e, index)}
                  onDragEnd={handleDragEnd}
                  className={`group flex items-center gap-3 px-4 py-3 cursor-pointer transition-colors hover:bg-zinc-50 dark:hover:bg-zinc-800/50 ${
                    index > 0 ? 'border-t border-zinc-100 dark:border-zinc-800' : ''
                  } ${dragIndex === index ? 'opacity-50' : ''}`}
                  onClick={() => openEditDialog(index)}
                >
                  <GripVertical className="w-4 h-4 text-zinc-300 dark:text-zinc-600 shrink-0 cursor-grab" />
                  {item.icon && <Icon icon={item.icon} className="w-4 h-4 text-zinc-500 dark:text-zinc-400 shrink-0" />}
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <p className="text-sm font-medium text-zinc-900 dark:text-zinc-100 truncate">{item.title}</p>
                      {item.children && item.children.length > 0 && (
                        <span className="text-[10px] text-zinc-400 bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded">{item.children.length} 个子菜单</span>
                      )}
                    </div>
                    <p className="text-xs text-zinc-400 truncate">{item.url}</p>
                  </div>
                  {item.openInNewTab && <ExternalLink className="w-3.5 h-3.5 text-zinc-400 shrink-0" />}
                  <span
                    className="w-6 h-6 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-950"
                    title="添加子菜单"
                    onClick={(e) => { e.stopPropagation(); openCreateChildDialog(index) }}
                  >
                    <Plus className="w-3.5 h-3.5" />
                  </span>
                  <span
                    className="w-6 h-6 rounded-full flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
                    onClick={(e) => handleDelete(e, index)}
                  >
                    <X className="w-3.5 h-3.5" />
                  </span>
                </div>

                {item.children && item.children.length > 0 && (
                  <div className="bg-zinc-50/50 dark:bg-zinc-800/20">
                    {item.children.map((child, ci) => (
                      <div
                        key={`child-${ci}`}
                        draggable
                        onDragStart={(e) => { e.stopPropagation(); handleChildDragStart(index, ci) }}
                        onDragOver={(e) => handleChildDragOver(e, index, ci)}
                        onDragEnd={handleChildDragEnd}
                        className={`group/child flex items-center gap-3 pl-10 pr-4 py-2.5 cursor-pointer transition-colors hover:bg-zinc-100/50 dark:hover:bg-zinc-800/50 border-t border-zinc-100 dark:border-zinc-800 ${
                          dragChildInfo?.parent === index && dragChildInfo?.child === ci ? 'opacity-50' : ''
                        }`}
                        onClick={(e) => { e.stopPropagation(); openEditChildDialog(index, ci) }}
                      >
                        <ChevronRight className="w-3 h-3 text-zinc-300 dark:text-zinc-600 shrink-0" />
                        <GripVertical className="w-3.5 h-3.5 text-zinc-300 dark:text-zinc-600 shrink-0 cursor-grab" />
                        {child.icon && <Icon icon={child.icon} className="w-3.5 h-3.5 text-zinc-500 dark:text-zinc-400 shrink-0" />}
                        <div className="flex-1 min-w-0">
                          <p className="text-sm text-zinc-700 dark:text-zinc-300 truncate">{child.title}</p>
                          <p className="text-xs text-zinc-400 truncate">{child.url}</p>
                        </div>
                        {child.openInNewTab && <ExternalLink className="w-3 h-3 text-zinc-400 shrink-0" />}
                        <span
                          className="w-5 h-5 rounded-full flex items-center justify-center opacity-0 group-hover/child:opacity-100 transition-opacity text-zinc-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-950"
                          onClick={(e) => handleDeleteChild(e, index, ci)}
                        >
                          <X className="w-3 h-3" />
                        </span>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}

        {menus && menus.length > 0 && (
          <p className="text-xs text-zinc-400 mt-3">💡 拖拽可调整顺序</p>
        )}

        {/* 新建/编辑弹窗 */}
        <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
          <DialogContent className="sm:max-w-[440px]">
            <DialogHeader>
              <DialogTitle className="text-base">{isEditing ? '编辑菜单项' : '添加菜单项'}</DialogTitle>
            </DialogHeader>
            <div className="flex flex-col gap-4 pt-1 pb-2">
              {/* 图标 + 标题 横排 */}
              <div className="flex gap-3">
                <div className="flex flex-col gap-1.5">
                  <label className="text-xs font-medium text-zinc-500">图标</label>
                  <IconPicker
                    value={formData.icon || ''}
                    onChange={(v) => setFormData((p) => ({ ...p, icon: v }))}
                  />
                </div>
                <div className="flex-1 flex flex-col gap-1.5">
                  <label className="text-xs font-medium text-zinc-500">标题 *</label>
                  <Input
                    value={formData.title}
                    onChange={(e) => setFormData((p) => ({ ...p, title: e.target.value }))}
                    placeholder="例如：首页"
                    className="border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none"
                    autoFocus
                    onKeyDown={(e) => { if (e.key === 'Enter') { e.preventDefault(); handleSubmit() } }}
                  />
                </div>
              </div>

              {/* 链接 */}
              <div className="flex flex-col gap-1.5">
                <label className="text-xs font-medium text-zinc-500">链接 *</label>
                <Input
                  value={formData.url}
                  onChange={(e) => setFormData((p) => ({ ...p, url: e.target.value }))}
                  placeholder="例如：/ 或 https://example.com"
                  className="border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none"
                />
              </div>

              {/* 所属菜单 + 新标签 横排 */}
              <div className="flex items-end gap-3">
                <div className="flex-1 flex flex-col gap-1.5">
                  <label className="text-xs font-medium text-zinc-500">所属菜单</label>
                  <Select value={parentSelect} onValueChange={setParentSelect}>
                    <SelectTrigger className="border-zinc-200 dark:border-zinc-800 bg-transparent shadow-none h-9">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="none">无（顶级菜单）</SelectItem>
                      {getParentOptions().map((opt) => (
                        <SelectItem key={opt.index} value={String(opt.index)}>
                          <span className="flex items-center gap-1.5">
                            {opt.icon && <Icon icon={opt.icon} className="w-3.5 h-3.5" />}
                            {opt.title}
                          </span>
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <label className="flex items-center gap-2 cursor-pointer pb-1.5">
                  <Checkbox
                    checked={formData.openInNewTab}
                    onCheckedChange={(checked) => setFormData((p) => ({ ...p, openInNewTab: !!checked }))}
                    className="rounded"
                  />
                  <span className="text-sm text-zinc-600 dark:text-zinc-400 whitespace-nowrap">新窗口</span>
                </label>
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" className="shadow-none border-zinc-200 dark:border-zinc-800" onClick={() => setDialogOpen(false)}>取消</Button>
              <Button
                className="bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 shadow-none hover:bg-zinc-800 dark:hover:bg-zinc-200"
                onClick={handleSubmit}
                disabled={!formData.title.trim() || !formData.url.trim() || saveMutation.isPending}
              >
                {saveMutation.isPending ? '保存中…' : isEditing ? '保存修改' : '添加'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </Main>
    </>
  )
}
