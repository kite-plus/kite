import { useState, useMemo, useCallback } from 'react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { ChevronDown, ChevronRight, Check, FolderOpen, X } from 'lucide-react'
import type { Category } from '@/types/category'

// ─── 类型定义 ────────────────────────────────────────────────────────

/** 级联选择器的节点数据结构 */
export interface CascaderNode {
  id: string
  name: string
  children?: CascaderNode[]
}

export interface CategoryCascaderProps {
  /** 树形数据源 */
  options: CascaderNode[]
  /** 当前选中的节点 ID（最终叶子节点 ID） */
  value?: string | null
  /** 选中回调，返回选中节点的 ID */
  onChange?: (id: string | null) => void
  /** 占位文字 */
  placeholder?: string
  /** 是否禁用 */
  disabled?: boolean
  /** 是否允许选择非叶子节点（父分类也可被选中） */
  allowSelectParent?: boolean
  /** 自定义触发器容器类名 */
  className?: string
}

// ─── 工具函数 ────────────────────────────────────────────────────────

/**
 * 根据节点 ID，通过深度优先搜索在树中查找完整路径
 * @param nodes 树形数据
 * @param targetId 目标节点 ID
 * @returns 从根到目标节点的路径数组，未找到返回 null
 */
function findPathById(nodes: CascaderNode[], targetId: string): CascaderNode[] | null {
  for (const node of nodes) {
    // 命中当前节点
    if (node.id === targetId) return [node]
    // 递归搜索子节点
    if (node.children?.length) {
      const childPath = findPathById(node.children, targetId)
      if (childPath) return [node, ...childPath]
    }
  }
  return null
}

/**
 * 将路径数组格式化为 「A / B / C」 的显示文本
 */
function formatPathLabel(path: CascaderNode[]): string {
  return path.map((n) => n.name).join(' / ')
}

/**
 * 将扁平的 Category 列表转换为 CascaderNode 树结构（支持无限层级）
 * @param flat 扁平分类数组（含 parentId 字段）
 * @returns 树形的 CascaderNode 数组
 */
export function buildCascaderTree(flat: Category[]): CascaderNode[] {
  // 先创建所有节点的映射
  const nodeMap = new Map<string, CascaderNode>()
  for (const cat of flat) {
    nodeMap.set(cat.id, { id: cat.id, name: cat.name })
  }

  const roots: CascaderNode[] = []

  // 遍历所有分类，将子节点挂载到父节点
  for (const cat of flat) {
    const node = nodeMap.get(cat.id)!
    if (cat.parentId) {
      const parent = nodeMap.get(cat.parentId)
      if (parent) {
        if (!parent.children) parent.children = []
        parent.children.push(node)
      } else {
        // parentId 指向不存在的节点，作为根节点处理
        roots.push(node)
      }
    } else {
      roots.push(node)
    }
  }

  return roots
}



// ─── 桌面端多列并排组件 ──────────────────────────────────────────────

interface ColumnPanelProps {
  options: CascaderNode[]
  value: string | null
  allowSelectParent: boolean
  onSelect: (id: string) => void
}

/**
 * 多列并排模式（macOS Finder 风格）
 * 选中父级后，右侧平滑展开子级列表
 */
function TreeNode({
  node,
  value,
  depth,
  siblings,
  onSelect,
  expandedIds,
  onHoverNode,
}: {
  node: CascaderNode
  value: string | null
  depth: number
  siblings: CascaderNode[]
  onSelect: (id: string) => void
  expandedIds: Set<string>
  onHoverNode: (id: string, siblings: CascaderNode[]) => void
}) {
  const hasChildren = !!(node.children?.length)
  const isSelected = node.id === value
  const isExpanded = expandedIds.has(node.id)

  return (
    <>
      <div
        className={cn(
          'flex items-center gap-1.5 py-1.5 px-2 rounded-md text-sm transition-colors',
          hasChildren ? 'cursor-default' : 'cursor-pointer',
          isSelected
            ? 'bg-blue-500 text-white'
            : 'text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800/60'
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
        onMouseEnter={() => {
          if (hasChildren) {
            onHoverNode(node.id, siblings)
          }
        }}
        onClick={() => {
          if (!hasChildren) {
            onSelect(node.id)
          }
        }}
      >
        {hasChildren ? (
          <ChevronRight
            className={cn(
              'w-3.5 h-3.5 shrink-0 transition-transform duration-150',
              isExpanded && 'rotate-90',
              isSelected ? 'opacity-80' : 'opacity-50'
            )}
          />
        ) : (
          <span className="w-3.5 shrink-0" />
        )}
        <FolderOpen className={cn('w-3.5 h-3.5 shrink-0', isSelected ? 'opacity-80' : 'text-zinc-400')} />
        <span className="flex-1 truncate">{node.name}</span>
        {isSelected && <Check className="w-3.5 h-3.5 shrink-0" />}
      </div>
      {hasChildren && isExpanded && (
        <div className="animate-in slide-in-from-top-1 fade-in-70 duration-150">
          {node.children!.map((child) => (
            <TreeNode
              key={child.id}
              node={child}
              value={value}
              depth={depth + 1}
              siblings={node.children!}
              onSelect={onSelect}
              expandedIds={expandedIds}
              onHoverNode={onHoverNode}
            />
          ))}
        </div>
      )}
    </>
  )
}

/**
 * 树形面板（手风琴模式）
 * - hover 展开当前项，自动收起同级兄弟
 * - 鼠标移出面板后全部收起，只显示根分类
 */
function TreePanel({ options, value, onSelect }: ColumnPanelProps) {
  const [expandedIds, setExpandedIds] = useState<Set<string>>(() => {
    const ids = new Set<string>()
    if (value) {
      const path = findPathById(options, value)
      if (path) {
        for (let i = 0; i < path.length - 1; i++) ids.add(path[i].id)
      }
    }
    return ids
  })

  // hover 某个父节点时：展开它，关闭同级其他兄弟
  const onHoverNode = useCallback((id: string, siblings: CascaderNode[]) => {
    setExpandedIds((prev) => {
      const next = new Set(prev)
      // 收起同级兄弟
      for (const sib of siblings) {
        if (sib.id !== id) next.delete(sib.id)
      }
      next.add(id)
      return next
    })
  }, [])

  // 鼠标移出整个面板，全部收起
  function handleMouseLeave() {
    // 保留选中项的展开路径
    const keep = new Set<string>()
    if (value) {
      const path = findPathById(options, value)
      if (path) {
        for (let i = 0; i < path.length - 1; i++) keep.add(path[i].id)
      }
    }
    setExpandedIds(keep)
  }

  return (
    <div className="w-[240px]" onMouseLeave={handleMouseLeave}>
      <ScrollArea className="max-h-64">
        <div className="p-1.5">
          {options.length === 0 && (
            <div className="px-3 py-5 text-center text-xs text-zinc-400">暂无分类数据</div>
          )}
          {options.map((node) => (
            <TreeNode
              key={node.id}
              node={node}
              value={value}
              depth={0}
              siblings={options}
              onSelect={onSelect}
              expandedIds={expandedIds}
              onHoverNode={onHoverNode}
            />
          ))}
        </div>
      </ScrollArea>
    </div>
  )
}

// ─── 主组件 ──────────────────────────────────────────────────────────

/**
 * 分类级联选择器
 *
 * 基于 shadcn Popover：
 * - 桌面端：多列并排 (macOS Finder 风格)
 * - 触发器回显完整分类路径（例：后端 / Golang / 并发编程）
 *
 * @example
 * ```tsx
 * <CategoryCascader
 *   options={treeData}
 *   value={selectedId}
 *   onChange={setSelectedId}
 *   allowSelectParent
 * />
 * ```
 */
export function CategoryCascader({
  options,
  value = null,
  onChange,
  placeholder = '选择分类',
  disabled = false,
  allowSelectParent = true,
  className,
}: CategoryCascaderProps) {
  const [open, setOpen] = useState(false)

  /** 回显路径 label */
  const displayLabel = useMemo(() => {
    if (!value) return null
    const path = findPathById(options, value)
    if (!path) return null
    return formatPathLabel(path)
  }, [value, options])

  /** 选中处理 */
  const handleSelect = useCallback(
    (id: string) => {
      onChange?.(id)
      setOpen(false)
    },
    [onChange],
  )

  /** 清除选中 */
  const handleClear = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation()
      onChange?.(null)
    },
    [onChange],
  )

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          role="combobox"
          aria-expanded={open}
          disabled={disabled}
          className={cn(
            'w-full justify-between font-normal shadow-none rounded-md border-zinc-200 dark:border-zinc-700 hover:bg-transparent',
            !value && 'text-muted-foreground',
            className,
          )}
        >
          <span className="flex items-center gap-1.5 truncate">
            {displayLabel ? (
              <>
                <FolderOpen className="w-3.5 h-3.5 text-zinc-400 shrink-0" />
                <span className="truncate">{displayLabel}</span>
              </>
            ) : (
              <span>{placeholder}</span>
            )}
          </span>
          <span className="flex items-center gap-0.5 shrink-0">
            {value && (
              <span
                className="p-0.5 rounded-sm hover:bg-zinc-100 dark:hover:bg-zinc-800 cursor-pointer"
                onClick={handleClear}
              >
                <X className="w-3 h-3 text-zinc-400" />
              </span>
            )}
            <ChevronDown className="w-3.5 h-3.5 text-zinc-400" />
          </span>
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="start"
        className="w-auto p-0 rounded-md"
      >
        {options.length === 0 ? (
          <div className="px-4 py-6 text-center text-xs text-zinc-400">暂无分类数据</div>
        ) : (
          <TreePanel
            options={options}
            value={value}
            allowSelectParent={allowSelectParent}
            onSelect={handleSelect}
          />
        )}
      </PopoverContent>
    </Popover>
  )
}
