import { useState } from 'react'
import { useNavigate } from 'react-router'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Pagination,
  PaginationContent,
  PaginationItem,
  PaginationNext,
  PaginationPrevious,
} from '@/components/ui/pagination'
import { Plus, Pencil, Trash2, FileText, Loader2, ExternalLink, X, Eye } from 'lucide-react'
import { usePosts, useDeletePost } from '@/hooks/use-posts'
import { useCategoryList } from '@/hooks/use-categories'
import type { PostStatus } from '@/types/post'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search as SearchBtn } from '@/components/search'

function formatDate(dateStr: string | null) {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

/**
 * 文章管理页面
 */
export function PostsPage() {
  const navigate = useNavigate()
  const [keyword, setKeyword] = useState('')
  const [status, setStatus] = useState<PostStatus | 'all'>('all')
  const [categoryId, setCategoryId] = useState('')
  const [page, setPage] = useState(1)
  const [deleteTarget, setDeleteTarget] = useState<{ id: string; title: string } | null>(null)
  const pageSize = 10

  const { data, isLoading } = usePosts({ page, pageSize, keyword, status, categoryId: categoryId || undefined })
  const { data: categories } = useCategoryList()
  const deleteMutation = useDeletePost()

  function handleDelete() {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget.id)
    setDeleteTarget(null)
  }

  const totalPages = data ? Math.ceil(data.pagination.total / pageSize) : 0

  return (
    <>
      <Header fixed>
        <SearchBtn />
        <div className='ml-auto' />
      </Header>
      <Main className="flex flex-1 flex-col gap-4 sm:gap-6">
      {/* 标题区 */}
      <div className="flex flex-wrap items-end justify-between gap-2">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">文章列表</h2>
          <p className="text-muted-foreground mt-1">
            管理您的博客文章及其状态。
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => navigate('/posts/new')} className="h-9">
            <Plus className="w-4 h-4 mr-2" /> 新建文章
          </Button>
        </div>
      </div>

      {/* 搜索与筛选工具栏 */}
      <div className="flex items-center justify-between">
        <div className="flex flex-1 flex-col-reverse items-start gap-y-2 sm:flex-row sm:items-center sm:space-x-2">
          <Input
            placeholder="搜索文章标题…"
            value={keyword}
            onChange={(e) => { setKeyword(e.target.value); setPage(1) }}
            className="h-8 w-[150px] lg:w-[250px]"
          />
          <div className="flex gap-x-2">
            <Select value={status} onValueChange={(v: any) => { setStatus(v); setPage(1) }}>
              <SelectTrigger className="h-8 w-[130px] border-dashed">
                <SelectValue placeholder="状态" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">所有状态</SelectItem>
                <SelectItem value="published">已发布</SelectItem>
                <SelectItem value="draft">草稿</SelectItem>
                <SelectItem value="archived">已归档</SelectItem>
              </SelectContent>
            </Select>

            <Select value={categoryId || 'all'} onValueChange={(v) => { setCategoryId(v === 'all' ? '' : v); setPage(1) }}>
              <SelectTrigger className="h-8 w-[150px] border-dashed">
                <SelectValue placeholder="全部分类" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部分类</SelectItem>
                {categories?.map((c) => (
                  <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {(keyword || status !== 'all' || categoryId) && (
            <Button
              variant="ghost"
              onClick={() => {
                setKeyword('')
                setStatus('all')
                setCategoryId('')
                setPage(1)
              }}
              className="h-8 px-2 lg:px-3 text-muted-foreground"
            >
              重设
              <X className="ml-2 h-4 w-4" />
            </Button>
          )}
        </div>
      </div>

      {/* 博客列表 */}
      <div className="overflow-hidden rounded-md border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-[100px]">状态</TableHead>
              <TableHead className="w-[100px]">分类</TableHead>
              <TableHead>文章信息</TableHead>
              <TableHead className="w-[80px] text-center">浏览</TableHead>
              <TableHead className="w-[120px]">日期</TableHead>
              <TableHead className="w-[100px] text-right">操作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-16">
                  <Loader2 className="w-5 h-5 animate-spin text-zinc-400 mx-auto" />
                </TableCell>
              </TableRow>
            ) : data?.items && data.items.length > 0 ? (
              data.items.map((post) => {
                return (
                <TableRow
                  key={post.id}
                  className="cursor-pointer"
                  onDoubleClick={() => navigate(`/posts/${post.id}/edit`)}
                >
                  <TableCell>
                    {post.status === 'published' ? (
                      <Badge variant="outline" className="text-emerald-600 border-emerald-500/20 bg-emerald-500/10 dark:text-emerald-400 dark:bg-emerald-500/10 font-medium">已发布</Badge>
                    ) : (
                      <Badge variant="outline" className="text-muted-foreground font-medium">草稿</Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {post.category ? (
                      <Badge variant="outline" className="text-xs">{post.category.name}</Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell onClick={() => navigate(`/posts/${post.id}/edit`)}>
                    <p className="text-sm font-medium text-foreground">{post.title}</p>
                    <p className="text-xs text-muted-foreground mt-0.5 truncate max-w-[400px] leading-relaxed">{post.summary}</p>
                  </TableCell>
                  <TableCell className="text-center">
                    <span className="inline-flex items-center gap-1 text-xs text-muted-foreground">
                      <Eye className="w-3.5 h-3.5" />
                      {post.viewCount ?? 0}
                    </span>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">{formatDate(post.updatedAt)}</TableCell>
                  <TableCell className="text-right">
                    <div className="flex gap-1 justify-end" onClick={(e) => e.stopPropagation()}>
                      {post.status === 'published' && (
                        <Tooltip delayDuration={0}>
                          <TooltipTrigger asChild>
                            <Button variant="ghost" size="icon" className="w-8 h-8" onClick={() => window.open(`/posts/${post.slug}`, '_blank')}>
                              <ExternalLink className="w-4 h-4" />
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent side="top">预览</TooltipContent>
                        </Tooltip>
                      )}
                      <Tooltip delayDuration={0}>
                        <TooltipTrigger asChild>
                          <Button variant="ghost" size="icon" className="w-8 h-8" onClick={() => navigate(`/posts/${post.id}/edit`)}>
                            <Pencil className="w-4 h-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent side="top">编辑</TooltipContent>
                      </Tooltip>
                      <Tooltip delayDuration={0}>
                        <TooltipTrigger asChild>
                          <Button variant="ghost" size="icon" className="w-8 h-8 text-destructive hover:text-destructive" onClick={() => setDeleteTarget({ id: post.id, title: post.title })}>
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent side="top">删除</TooltipContent>
                      </Tooltip>
                    </div>
                  </TableCell>
                </TableRow>
                )
              })
            ) : (
              <TableRow>
                <TableCell colSpan={6} className="text-center py-16">
                  <div className="flex flex-col items-center">
                    <FileText className="w-10 h-10 text-muted-foreground mb-3" />
                    <p className="text-sm font-medium text-foreground">没有找到匹配的文章</p>
                    <p className="text-sm text-muted-foreground mt-1">换个关键词试试，或者点击右上角新建文章</p>
                  </div>
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {/* 分页 */}
      {data && data.pagination.total > 0 && (
        <div className="flex items-center justify-between pb-4">
          <div className="flex-1 text-sm text-muted-foreground">
            共 {data.pagination.total} 篇文章
          </div>
          {totalPages > 1 && (
            <div className="flex items-center space-x-6 lg:space-x-8">
              <div className="flex items-center space-x-2">
                <p className="text-sm font-medium">第 {page} 页，共 {totalPages} 页</p>
              </div>
              <Pagination>
                <PaginationContent>
                  <PaginationItem>
                    <PaginationPrevious onClick={() => setPage(Math.max(1, page - 1))} className={page <= 1 ? 'pointer-events-none opacity-50' : 'cursor-pointer'} />
                  </PaginationItem>
                  <PaginationItem>
                    <PaginationNext onClick={() => setPage(Math.min(totalPages, page + 1))} className={page >= totalPages ? 'pointer-events-none opacity-50' : 'cursor-pointer'} />
                  </PaginationItem>
                </PaginationContent>
              </Pagination>
            </div>
          )}
        </div>
      )}

      {/* 删除确认 */}
      <AlertDialog open={deleteTarget !== null} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <AlertDialogContent className="rounded-lg">
          <AlertDialogHeader>
            <AlertDialogTitle className="text-zinc-950 dark:text-zinc-50">删除文章</AlertDialogTitle>
            <AlertDialogDescription className="text-zinc-500">
              {deleteTarget && `确定要删除「${deleteTarget.title}」吗？此操作不可撤销。`}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel className="shadow-none border-zinc-200 dark:border-zinc-800">取消</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-red-500 text-white hover:bg-red-600 shadow-none">确认删除</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      </Main>
    </>
  )
}
