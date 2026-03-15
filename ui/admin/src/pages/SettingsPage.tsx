import { useState, useEffect } from 'react'
import { Card, Button, Input, Select, Switch, Tabs, TabPane, Typography, TextArea, Divider } from '@douyinfe/semi-ui'
import { IconSave, IconGlobe, IconArticle, IconServer, IconStar, IconTick } from '@douyinfe/semi-icons'
import { useSettings, useSaveSettings } from '@/hooks/use-settings'
import type { AllSettings } from '@/types/settings'

const { Title, Text } = Typography

/**
 * 系统设置页面 — Semi Tabs / Input / Select / Switch / Button
 */
export function SettingsPage() {
  const [form, setForm] = useState<AllSettings | null>(null)
  const [saved, setSaved] = useState(false)

  const { data: settings, isLoading } = useSettings()
  const saveMutation = useSaveSettings()

  useEffect(() => {
    if (settings && !form) setForm(structuredClone(settings))
  }, [settings, form])

  function handleSave() {
    if (!form) return
    saveMutation.mutate(form, { onSuccess: () => { setSaved(true); setTimeout(() => setSaved(false), 2000) } })
  }

  if (isLoading || !form) {
    return <div style={{ textAlign: 'center', padding: 64 }}><Text type="tertiary">加载中…</Text></div>
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 24 }}>
        <Title heading={4}>设置</Title>
        <Button icon={saved ? <IconTick /> : <IconSave />} theme="solid" onClick={handleSave} disabled={saveMutation.isPending}>
          {saved ? '已保存' : saveMutation.isPending ? '保存中…' : '保存设置'}
        </Button>
      </div>

      <Card>
        <Tabs type="line" tabPosition="left">
          {/* 站点信息 */}
          <TabPane tab={<><IconGlobe size="small" /> 站点信息</>} itemKey="site">
            <div style={{ padding: '0 24px', display: 'flex', flexDirection: 'column', gap: 20 }}>
              <FormRow label="站点名称">
                <Input value={form.site.siteName} onChange={(v) => updateField(setForm, 'site', 'siteName', v)} placeholder="Kite Blog" style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="站点 URL">
                <Input value={form.site.siteUrl} onChange={(v) => updateField(setForm, 'site', 'siteUrl', v)} placeholder="https://blog.example.com" style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="站点描述">
                <TextArea value={form.site.description} onChange={(v) => updateField(setForm, 'site', 'description', v)} placeholder="简要描述你的博客…" rows={2} style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="SEO 关键词">
                <Input value={form.site.keywords} onChange={(v) => updateField(setForm, 'site', 'keywords', v)} placeholder="博客,技术,Go,React" style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="ICP 备案号">
                <Input value={form.site.icp} onChange={(v) => updateField(setForm, 'site', 'icp', v)} placeholder="京ICP备XXXXXXXX号" style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="Favicon 路径">
                <Input value={form.site.favicon} onChange={(v) => updateField(setForm, 'site', 'favicon', v)} placeholder="/favicon.svg" style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="页脚文本">
                <Input value={form.site.footer} onChange={(v) => updateField(setForm, 'site', 'footer', v)} placeholder="© 2026 Blog" style={{ maxWidth: 360 }} />
              </FormRow>
            </div>
          </TabPane>

          {/* 文章设置 */}
          <TabPane tab={<><IconArticle size="small" /> 文章设置</>} itemKey="post">
            <div style={{ padding: '0 24px', display: 'flex', flexDirection: 'column', gap: 20 }}>
              <FormRow label="每页文章数">
                <Input type="number" value={String(form.post.postsPerPage)} onChange={(v) => updateField(setForm, 'post', 'postsPerPage', Number(v))} style={{ width: 100 }} />
              </FormRow>
              <FormRow label="摘要截取长度">
                <Input type="number" value={String(form.post.summaryLength)} onChange={(v) => updateField(setForm, 'post', 'summaryLength', Number(v))} style={{ width: 100 }} />
              </FormRow>
              <FormRow label="启用评论" description="允许读者在文章下方发表评论">
                <Switch checked={form.post.enableComment as boolean} onChange={(v) => updateField(setForm, 'post', 'enableComment', v)} />
              </FormRow>
              <FormRow label="启用目录" description="自动生成文章目录导航">
                <Switch checked={form.post.enableToc as boolean} onChange={(v) => updateField(setForm, 'post', 'enableToc', v)} />
              </FormRow>
            </div>
          </TabPane>

          {/* 渲染模式 */}
          <TabPane tab={<><IconServer size="small" /> 渲染模式</>} itemKey="render">
            <div style={{ padding: '0 24px', display: 'flex', flexDirection: 'column', gap: 20 }}>
              <FormRow label="渲染模式">
                <Select value={form.render.renderMode} onChange={(v) => updateField(setForm, 'render', 'renderMode', v as string)} style={{ maxWidth: 360 }}>
                  <Select.Option value="classic">Classic — Go Template 服务端渲染</Select.Option>
                  <Select.Option value="headless">Headless — 纯 JSON API 输出</Select.Option>
                </Select>
              </FormRow>
              <FormRow label="API 前缀">
                <Input value={form.render.apiPrefix} onChange={(v) => updateField(setForm, 'render', 'apiPrefix', v)} placeholder="/api/v1" style={{ maxWidth: 360 }} />
              </FormRow>
              <FormRow label="启用 CORS" description="允许跨域请求">
                <Switch checked={form.render.enableCors as boolean} onChange={(v) => updateField(setForm, 'render', 'enableCors', v)} />
              </FormRow>
            </div>
          </TabPane>

          {/* AI 集成 */}
          <TabPane tab={<><IconStar size="small" /> AI 集成</>} itemKey="ai">
            <div style={{ padding: '0 24px', display: 'flex', flexDirection: 'column', gap: 20 }}>
              <FormRow label="启用 AI 功能" description="开启后提供 AI 辅助">
                <Switch checked={form.ai.enabled as boolean} onChange={(v) => updateField(setForm, 'ai', 'enabled', v)} />
              </FormRow>
              {form.ai.enabled && (
                <>
                  <FormRow label="AI 服务商">
                    <Select value={form.ai.provider} onChange={(v) => updateField(setForm, 'ai', 'provider', v as string)} style={{ maxWidth: 360 }}>
                      <Select.Option value="deepseek">DeepSeek</Select.Option>
                      <Select.Option value="openai">OpenAI</Select.Option>
                    </Select>
                  </FormRow>
                  <FormRow label="模型名称">
                    <Input value={form.ai.model} onChange={(v) => updateField(setForm, 'ai', 'model', v)} placeholder="deepseek-chat" style={{ maxWidth: 360 }} />
                  </FormRow>
                  <FormRow label="API Key">
                    <Input mode="password" value={form.ai.apiKey} onChange={(v) => updateField(setForm, 'ai', 'apiKey', v)} placeholder="sk-xxxxxxxxxxxx" style={{ maxWidth: 360 }} />
                  </FormRow>
                  <FormRow label="自动生成摘要" description="发布时自动调用 AI 生成摘要">
                    <Switch checked={form.ai.autoSummary as boolean} onChange={(v) => updateField(setForm, 'ai', 'autoSummary', v)} />
                  </FormRow>
                  <FormRow label="自动推荐标签" description="基于内容推荐标签">
                    <Switch checked={form.ai.autoTag as boolean} onChange={(v) => updateField(setForm, 'ai', 'autoTag', v)} />
                  </FormRow>
                </>
              )}
            </div>
          </TabPane>
        </Tabs>
      </Card>
    </div>
  )
}

/* ========== 辅助组件 ========== */
function FormRow({ label, description, children }: { label: string; description?: string; children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', alignItems: 'flex-start', gap: 32 }}>
      <div style={{ width: 140, flexShrink: 0, paddingTop: 8 }}>
        <Text strong style={{ fontSize: 14 }}>{label}</Text>
        {description && <Text type="tertiary" size="small" style={{ display: 'block', marginTop: 2 }}>{description}</Text>}
      </div>
      <div style={{ flex: 1 }}>{children}</div>
    </div>
  )
}

/** 通用表单字段更新 */
function updateField(
  setForm: React.Dispatch<React.SetStateAction<AllSettings | null>>,
  section: keyof AllSettings,
  key: string,
  value: unknown
) {
  setForm((prev) => prev ? { ...prev, [section]: { ...prev[section], [key]: value } } : prev)
}
