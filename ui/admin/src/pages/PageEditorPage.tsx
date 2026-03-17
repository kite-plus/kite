import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate } from 'react-router'
import { Card, Button, Input, Switch, Typography, InputNumber, Select } from '@douyinfe/semi-ui'
import { IconArrowLeft, IconSave, IconSend, IconTick } from '@douyinfe/semi-icons'
import { TiptapEditor } from '@/components/TiptapEditor'
import { usePageDetail, useSavePage } from '@/hooks/use-pages'
import type { PageFormData } from '@/types/page'

const { Title, Text } = Typography

/** 模板配置字段定义 */
interface TemplateField {
  key: string
  label: string
  description?: string
  type: 'text' | 'number' | 'switch'
  defaultValue: string | number | boolean
  placeholder?: string
}

/** 各模板的字段定义 */
const TEMPLATE_FIELDS: Record<string, TemplateField[]> = {
  default: [],
  github: [
    { key: 'username', label: 'GitHub 用户名', type: 'text', defaultValue: '', placeholder: '如：octocat' },
    { key: 'count', label: '展示仓库数量', description: '按 Star 数排序取前 N 个', type: 'number', defaultValue: 6 },
    { key: 'show_fork', label: '显示 Fork 仓库', description: '是否包含 Fork 的仓库', type: 'switch', defaultValue: false },
  ],
}

/** 可用的页面模板列表 */
const PAGE_TEMPLATES = [
  { value: 'default', label: '默认模板' },
  { value: 'github', label: 'GitHub 风格' },
]

/**
 * 独立页面编辑器 — 复用 TiptapEditor，含模板选择和动态元数据表单
 */
export function PageEditorPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const isEdit = !!id
  const [saved, setSaved] = useState(false)

  const [form, setForm] = useState<PageFormData>({
    title: '', slug: '', contentMarkdown: '',
    status: 'draft', sortOrder: 10, showInNav: false,
    template: 'default', config: '',
  })

  const { data: page, isLoading } = usePageDetail(id)
  const saveMutation = useSavePage()

  /** 解析 config JSON 为对象 */
  const configObj = useMemo<Record<string, unknown>>(() => {
    if (!form.config) return {}
    try { return JSON.parse(form.config) } catch { return {} }
  }, [form.config])

  /** 更新 config 中的某个字段 */
  function updateConfigField(key: string, value: unknown) {
    const newConfig = { ...configObj, [key]: value }
    setForm((prev) => ({ ...prev, config: JSON.stringify(newConfig) }))
  }

  /** 获取当前模板的字段定义 */
  const currentFields = TEMPLATE_FIELDS[form.template] || []

  /** 切换模板时，初始化默认 config */
  function handleTemplateChange(template: string) {
    const fields = TEMPLATE_FIELDS[template] || []
    const defaults: Record<string, unknown> = {}
    fields.forEach((f) => {
      // 保留已有值，只为新字段设默认值
      defaults[f.key] = configObj[f.key] !== undefined ? configObj[f.key] : f.defaultValue
    })
    setForm((prev) => ({
      ...prev,
      template,
      config: fields.length > 0 ? JSON.stringify(defaults) : '',
    }))
  }

  useEffect(() => {
    if (page) {
      setForm({
        title: page.title, slug: page.slug, contentMarkdown: page.contentMarkdown || '',
        status: page.status, sortOrder: page.sortOrder, showInNav: page.showInNav,
        template: page.template || 'default', config: page.config || '',
      })
    }
  }, [page])

  function handleTitleChange(title: string) {
    setForm((prev) => ({
      ...prev, title,
      slug: prev.slug || title.toLowerCase().replace(/[\s]+/g, '-').replace(/[^a-z0-9\u4e00-\u9fa5-]/g, ''),
    }))
  }

  function handleSave(publish = false) {
    const data = { ...form, id }
    if (publish) data.status = 'published'
    saveMutation.mutate(data, {
      onSuccess: () => { setSaved(true); setTimeout(() => setSaved(false), 2000); if (!isEdit) navigate('/pages') },
    })
  }

  if (isEdit && isLoading) {
    return <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>
  }

  return (
    <div>
      {/* 顶部操作栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Button icon={<IconArrowLeft />} theme="borderless" onClick={() => navigate('/pages')} />
          <div>
            <Title heading={4}>{isEdit ? '编辑页面' : '新建页面'}</Title>
            <Text type="tertiary" size="small">{isEdit ? `正在编辑：${page?.title}` : '创建独立页面'}</Text>
          </div>
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <Button icon={saved ? <IconTick /> : <IconSave />} theme="light" onClick={() => handleSave(false)} disabled={saveMutation.isPending}>
            {saved ? '已保存' : '保存草稿'}
          </Button>
          <Button icon={<IconSend />} theme="solid" onClick={() => handleSave(true)} disabled={saveMutation.isPending || !form.title.trim()}>
            发布页面
          </Button>
        </div>
      </div>

      <div style={{ display: 'flex', gap: 24 }}>
        {/* 左侧编辑区 */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Input
            value={form.title}
            onChange={(v) => handleTitleChange(v)}
            placeholder="页面标题"
            size="large"
          />
          <TiptapEditor
            content={form.contentMarkdown}
            onChange={(html) => setForm((prev) => ({ ...prev, contentMarkdown: html }))}
            placeholder="开始编写页面内容…"
          />
        </div>

        {/* 右侧元数据面板 */}
        <div style={{ width: 280, display: 'flex', flexDirection: 'column', gap: 16 }}>
          <Card title="URL Slug">
            <Input
              value={form.slug}
              onChange={(v) => setForm((prev) => ({ ...prev, slug: v }))}
              placeholder="page-slug"
              prefix="/"
            />
            <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
              访问地址：/{form.slug || 'page-slug'}
            </Text>
          </Card>

          {/* 模板选择 */}
          <Card title="页面模板">
            <Select
              value={form.template}
              onChange={(v) => handleTemplateChange(v as string)}
              optionList={PAGE_TEMPLATES}
              style={{ width: '100%' }}
              placeholder="选择模板"
            />
            <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 8 }}>
              模板文件：pages/{form.template || 'default'}.html
            </Text>
          </Card>

          {/* 模板参数 — 根据所选模板动态渲染表单字段 */}
          {currentFields.length > 0 && (
            <Card title="模板参数">
              <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                {currentFields.map((field) => (
                  <div key={field.key}>
                    {field.type === 'switch' ? (
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <Text strong style={{ fontSize: 13 }}>{field.label}</Text>
                          {field.description && (
                            <Text type="tertiary" size="small" style={{ display: 'block' }}>{field.description}</Text>
                          )}
                        </div>
                        <Switch
                          checked={Boolean(configObj[field.key] ?? field.defaultValue)}
                          onChange={(v) => updateConfigField(field.key, v)}
                        />
                      </div>
                    ) : (
                      <>
                        <Text strong style={{ fontSize: 13, display: 'block', marginBottom: 6 }}>{field.label}</Text>
                        {field.type === 'text' ? (
                          <Input
                            value={String(configObj[field.key] ?? field.defaultValue)}
                            onChange={(v) => updateConfigField(field.key, v)}
                            placeholder={field.placeholder}
                          />
                        ) : (
                          <InputNumber
                            value={Number(configObj[field.key] ?? field.defaultValue)}
                            onChange={(v) => updateConfigField(field.key, v)}
                            min={1}
                            max={100}
                            style={{ width: '100%' }}
                          />
                        )}
                        {field.description && (
                          <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 4 }}>{field.description}</Text>
                        )}
                      </>
                    )}
                  </div>
                ))}
              </div>
            </Card>
          )}

          <Card title="页面设置">
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <div>
                  <Text strong style={{ fontSize: 13 }}>显示在导航栏</Text>
                  <Text type="tertiary" size="small" style={{ display: 'block' }}>前台顶部导航显示此页面</Text>
                </div>
                <Switch
                  checked={form.showInNav}
                  onChange={(v) => setForm((prev) => ({ ...prev, showInNav: v }))}
                />
              </div>
              <div>
                <Text strong style={{ fontSize: 13, display: 'block', marginBottom: 8 }}>排序优先级</Text>
                <InputNumber
                  value={form.sortOrder}
                  onChange={(v) => setForm((prev) => ({ ...prev, sortOrder: v as number }))}
                  min={0}
                  max={999}
                  style={{ width: '100%' }}
                />
                <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 4 }}>数值越小越靠前</Text>
              </div>
            </div>
          </Card>
        </div>
      </div>
    </div>
  )
}
