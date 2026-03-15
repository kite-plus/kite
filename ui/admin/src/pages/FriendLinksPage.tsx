import { useState } from 'react'
import { Card, Button, Input, Tag, Divider, Typography, Empty, Avatar } from '@douyinfe/semi-ui'
import { IconSearch, IconPlus, IconClose, IconDelete, IconLink, IconGlobe, IconTickCircle, IconAlertTriangle, IconExternalOpen } from '@douyinfe/semi-icons'
import { useFriendLinks, useCreateFriendLink, useDeleteFriendLink, useToggleLinkStatus } from '@/hooks/use-friend-links'
import type { LinkStatus } from '@/types/friend-link'

const { Title, Text } = Typography

/** 状态配置 */
const statusConfig: Record<LinkStatus, { text: string; color: string }> = {
  active: { text: '正常', color: 'blue' },
  pending: { text: '待审核', color: 'orange' },
  down: { text: '已下线', color: 'red' },
}

/**
 * 友链管理页面 — Semi Card / Input / Tag / Button / Avatar
 */
export function FriendLinksPage() {
  const [keyword, setKeyword] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [formData, setFormData] = useState({ name: '', url: '', description: '' })

  const { data: links, isLoading } = useFriendLinks(keyword)
  const createMutation = useCreateFriendLink()
  const deleteMutation = useDeleteFriendLink()
  const toggleMutation = useToggleLinkStatus()

  function handleCreate() {
    if (!formData.name.trim() || !formData.url.trim()) return
    createMutation.mutate(formData, { onSuccess: () => { setFormData({ name: '', url: '', description: '' }); setShowForm(false) } })
  }

  function handleDelete(id: string, name: string) {
    if (window.confirm(`确定删除友链「${name}」吗？`)) deleteMutation.mutate(id)
  }

  function handleToggle(id: string, currentStatus: LinkStatus) {
    toggleMutation.mutate({ id, status: currentStatus === 'active' ? 'down' : 'active' })
  }

  function extractDomain(url: string): string {
    try { return new URL(url).hostname } catch { return url }
  }

  const activeCount = links?.filter((l) => l.status === 'active').length ?? 0

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div>
          <Title heading={4}>友链管理</Title>
          <Text type="tertiary" size="small">管理博客友情链接，互换链接、共建生态</Text>
        </div>
        <Button icon={<IconPlus />} theme="solid" onClick={() => setShowForm(true)}>新增友链</Button>
      </div>

      {showForm && (
        <Card style={{ marginBottom: 24 }} title="新增友链" headerExtraContent={<Button icon={<IconClose />} theme="borderless" size="small" onClick={() => setShowForm(false)} />}>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
            <div>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>名称 *</Text>
              <Input value={formData.name} onChange={(v) => setFormData((p) => ({ ...p, name: v }))} placeholder="例如：阮一峰的网络日志" />
            </div>
            <div>
              <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>链接 *</Text>
              <Input value={formData.url} onChange={(v) => setFormData((p) => ({ ...p, url: v }))} placeholder="https://example.com" />
            </div>
          </div>
          <div style={{ marginTop: 16 }}>
            <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>描述</Text>
            <Input value={formData.description} onChange={(v) => setFormData((p) => ({ ...p, description: v }))} placeholder="一句话介绍这个博客…" />
          </div>
          <div style={{ marginTop: 16, display: 'flex', justifyContent: 'flex-end', gap: 8 }}>
            <Button theme="light" onClick={() => setShowForm(false)}>取消</Button>
            <Button theme="solid" onClick={handleCreate} disabled={!formData.name.trim() || !formData.url.trim() || createMutation.isPending}>
              {createMutation.isPending ? '创建中…' : '创建'}
            </Button>
          </div>
        </Card>
      )}

      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Input prefix={<IconSearch />} placeholder="搜索友链…" value={keyword} onChange={(v) => setKeyword(v)} style={{ maxWidth: 360 }} showClear />
        <Text type="tertiary" size="small">共 {links?.length ?? 0} 条友链 · {activeCount} 条在线</Text>
      </div>

      {isLoading && <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>}

      {!isLoading && links?.length === 0 && (
        <Card><Empty image={<IconLink style={{ fontSize: 48 }} />} description="暂无友链，点击上方按钮添加" /></Card>
      )}

      {links && links.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 16 }}>
          {links.map((link) => {
            const st = statusConfig[link.status]
            return (
              <Card key={link.id} style={{ opacity: link.status === 'down' ? 0.6 : 1 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                    <Avatar size="small" alt={link.name}>{link.name.charAt(0)}</Avatar>
                    <div>
                      <Text strong>{link.name}</Text>
                      <br />
                      <Text type="tertiary" size="small"><IconGlobe size="extra-small" /> {extractDomain(link.url)}</Text>
                    </div>
                  </div>
                  <Tag color={st.color} size="small">{st.text}</Tag>
                </div>
                {link.description && <Text type="tertiary" style={{ display: 'block', marginTop: 12, fontSize: 14 }}>{link.description}</Text>}
                <Divider margin={12} />
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Text type="tertiary" size="small">#{link.sortOrder}</Text>
                  <div style={{ display: 'flex', gap: 4 }}>
                    <Button icon={<IconExternalOpen />} theme="light" size="small" onClick={() => window.open(link.url, '_blank')}>访问</Button>
                    <Button icon={link.status === 'active' ? <IconAlertTriangle /> : <IconTickCircle />} theme="light" size="small" disabled={toggleMutation.isPending} onClick={() => handleToggle(link.id, link.status)}>
                      {link.status === 'active' ? '下线' : '上线'}
                    </Button>
                    <Button icon={<IconDelete />} theme="borderless" type="danger" size="small" disabled={deleteMutation.isPending} onClick={() => handleDelete(link.id, link.name)} />
                  </div>
                </div>
              </Card>
            )
          })}
        </div>
      )}
    </div>
  )
}
