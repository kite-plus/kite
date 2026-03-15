import { useState, useEffect } from 'react'
import { Save, Globe, FileText, Server, Sparkles, Check } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useSettings, useSaveSettings } from '@/hooks/use-settings'
import type { AllSettings } from '@/types/settings'

/** 设置选项卡 */
type SettingsTab = 'site' | 'post' | 'render' | 'ai'

const tabs: { key: SettingsTab; label: string; icon: React.ComponentType<{ className?: string; strokeWidth?: number }> }[] = [
  { key: 'site', label: '站点信息', icon: Globe },
  { key: 'post', label: '文章设置', icon: FileText },
  { key: 'render', label: '渲染模式', icon: Server },
  { key: 'ai', label: 'AI 集成', icon: Sparkles },
]

/**
 * 系统设置页面
 * 分 Tab 展示，表单编辑 + 保存
 */
export function SettingsPage() {
  const [activeTab, setActiveTab] = useState<SettingsTab>('site')
  const [form, setForm] = useState<AllSettings | null>(null)
  const [saved, setSaved] = useState(false)

  const { data: settings, isLoading } = useSettings()
  const saveMutation = useSaveSettings()

  // 数据加载后初始化表单
  useEffect(() => {
    if (settings && !form) {
      setForm(structuredClone(settings))
    }
  }, [settings, form])

  /** 保存设置 */
  function handleSave() {
    if (!form) return
    saveMutation.mutate(form, {
      onSuccess: () => {
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      },
    })
  }

  if (isLoading || !form) {
    return (
      <div className="flex items-center justify-center py-16 text-sm text-[var(--kite-text-muted)]">
        加载中…
      </div>
    )
  }

  return (
    <div>
      {/* 页面标题区 */}
      <div className="mb-6 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--kite-text-heading)]">
            系统设置
          </h1>
          <p className="mt-1 text-sm text-[var(--kite-text-muted)]">
            配置博客系统的核心参数
          </p>
        </div>
        <button
          onClick={handleSave}
          disabled={saveMutation.isPending}
          className="flex h-9 items-center gap-2 border border-[var(--kite-accent)] bg-[var(--kite-accent)] px-4 text-sm font-medium text-white transition-colors duration-100 hover:bg-[#333] disabled:opacity-50 cursor-pointer"
        >
          {saved ? (
            <>
              <Check className="h-4 w-4" strokeWidth={1.5} />
              已保存
            </>
          ) : saveMutation.isPending ? (
            '保存中…'
          ) : (
            <>
              <Save className="h-4 w-4" strokeWidth={1.5} />
              保存设置
            </>
          )}
        </button>
      </div>

      <div className="flex gap-6">
        {/* 左侧 Tab 导航 */}
        <nav className="w-44 flex-shrink-0">
          <ul className="space-y-0.5">
            {tabs.map((tab) => {
              const Icon = tab.icon
              const isActive = activeTab === tab.key
              return (
                <li key={tab.key}>
                  <button
                    onClick={() => setActiveTab(tab.key)}
                    className={cn(
                      'flex w-full items-center gap-2.5 border-l-2 px-3 py-2 text-sm transition-colors duration-100 cursor-pointer',
                      isActive
                        ? 'border-l-[var(--kite-accent)] bg-[var(--kite-bg-hover)] font-medium text-[var(--kite-text-heading)]'
                        : 'border-l-transparent text-[var(--kite-text-muted)] hover:bg-[var(--kite-bg-hover)] hover:text-[var(--kite-text-heading)]'
                    )}
                  >
                    <Icon className="h-4 w-4" strokeWidth={1.5} />
                    {tab.label}
                  </button>
                </li>
              )
            })}
          </ul>
        </nav>

        {/* 右侧表单区 */}
        <div className="flex-1 border border-[var(--kite-border)] bg-[var(--kite-bg)] p-6">
          {activeTab === 'site' && <SiteForm form={form} setForm={setForm} />}
          {activeTab === 'post' && <PostForm form={form} setForm={setForm} />}
          {activeTab === 'render' && <RenderForm form={form} setForm={setForm} />}
          {activeTab === 'ai' && <AiForm form={form} setForm={setForm} />}
        </div>
      </div>
    </div>
  )
}

/* ========== 通用表单组件 ========== */

interface FormProps {
  form: AllSettings
  setForm: React.Dispatch<React.SetStateAction<AllSettings | null>>
}

/** 表单项标签 */
function Label({ children }: { children: React.ReactNode }) {
  return <label className="mb-1.5 block text-xs font-medium text-[var(--kite-text-muted)]">{children}</label>
}

/** 文本输入 */
function TextInput({ value, onChange, placeholder }: { value: string; onChange: (v: string) => void; placeholder?: string }) {
  return (
    <input
      type="text"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
    />
  )
}

/** 文本域 */
function TextArea({ value, onChange, placeholder, rows = 3 }: { value: string; onChange: (v: string) => void; placeholder?: string; rows?: number }) {
  return (
    <textarea
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      rows={rows}
      className="w-full resize-none border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 py-2 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
    />
  )
}

/** 数字输入 */
function NumberInput({ value, onChange, min, max }: { value: number; onChange: (v: number) => void; min?: number; max?: number }) {
  return (
    <input
      type="number"
      value={value}
      onChange={(e) => onChange(Number(e.target.value))}
      min={min}
      max={max}
      className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none focus:border-[var(--kite-accent)]"
    />
  )
}

/** 开关组件 */
function Toggle({ checked, onChange, label, description }: { checked: boolean; onChange: (v: boolean) => void; label: string; description?: string }) {
  return (
    <button
      type="button"
      onClick={() => onChange(!checked)}
      className="flex w-full items-center justify-between border border-[var(--kite-border)] px-4 py-3 text-left transition-colors duration-100 hover:border-[var(--kite-border-hover)] cursor-pointer"
    >
      <div>
        <p className="text-sm font-medium text-[var(--kite-text-heading)]">{label}</p>
        {description && <p className="mt-0.5 text-xs text-[var(--kite-text-muted)]">{description}</p>}
      </div>
      <div className={cn(
        'flex h-5 w-9 items-center border p-0.5 transition-colors duration-150',
        checked ? 'border-[var(--kite-accent)] bg-[var(--kite-accent)]' : 'border-[var(--kite-border)] bg-[var(--kite-bg)]'
      )}>
        <div className={cn(
          'h-3.5 w-3.5 transition-transform duration-150',
          checked ? 'translate-x-3.5 bg-white' : 'translate-x-0 bg-[var(--kite-text-muted)]'
        )} />
      </div>
    </button>
  )
}

/** 下拉选择 */
function SelectInput({ value, onChange, options }: { value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }) {
  return (
    <select
      value={value}
      onChange={(e) => onChange(e.target.value)}
      className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none focus:border-[var(--kite-accent)] cursor-pointer"
    >
      {options.map((o) => (
        <option key={o.value} value={o.value}>{o.label}</option>
      ))}
    </select>
  )
}

/** 分组标题 */
function SectionTitle({ title, description }: { title: string; description?: string }) {
  return (
    <div className="mb-4">
      <h3 className="text-sm font-semibold text-[var(--kite-text-heading)]">{title}</h3>
      {description && <p className="mt-0.5 text-xs text-[var(--kite-text-muted)]">{description}</p>}
    </div>
  )
}

/* ========== 各模块表单 ========== */

/** 站点信息表单 */
function SiteForm({ form, setForm }: FormProps) {
  function update(key: keyof AllSettings['site'], value: string) {
    setForm((prev) => prev ? { ...prev, site: { ...prev.site, [key]: value } } : prev)
  }
  return (
    <div className="space-y-5">
      <SectionTitle title="站点信息" description="配置博客的基本信息，这些内容将显示在前台页面" />
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>站点名称</Label>
          <TextInput value={form.site.siteName} onChange={(v) => update('siteName', v)} placeholder="Kite Blog" />
        </div>
        <div>
          <Label>站点 URL</Label>
          <TextInput value={form.site.siteUrl} onChange={(v) => update('siteUrl', v)} placeholder="https://blog.example.com" />
        </div>
      </div>
      <div>
        <Label>站点描述</Label>
        <TextArea value={form.site.description} onChange={(v) => update('description', v)} placeholder="简要描述你的博客…" rows={2} />
      </div>
      <div>
        <Label>SEO 关键词</Label>
        <TextInput value={form.site.keywords} onChange={(v) => update('keywords', v)} placeholder="博客,技术,Go,React" />
      </div>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>ICP 备案号</Label>
          <TextInput value={form.site.icp} onChange={(v) => update('icp', v)} placeholder="京ICP备XXXXXXXX号" />
        </div>
        <div>
          <Label>Favicon 路径</Label>
          <TextInput value={form.site.favicon} onChange={(v) => update('favicon', v)} placeholder="/favicon.svg" />
        </div>
      </div>
      <div>
        <Label>页脚文本</Label>
        <TextInput value={form.site.footer} onChange={(v) => update('footer', v)} placeholder="© 2026 Blog" />
      </div>
    </div>
  )
}

/** 文章设置表单 */
function PostForm({ form, setForm }: FormProps) {
  function updateNum(key: keyof AllSettings['post'], value: number) {
    setForm((prev) => prev ? { ...prev, post: { ...prev.post, [key]: value } } : prev)
  }
  function updateBool(key: keyof AllSettings['post'], value: boolean) {
    setForm((prev) => prev ? { ...prev, post: { ...prev.post, [key]: value } } : prev)
  }
  return (
    <div className="space-y-5">
      <SectionTitle title="文章设置" description="控制文章的默认展示行为" />
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label>每页文章数</Label>
          <NumberInput value={form.post.postsPerPage} onChange={(v) => updateNum('postsPerPage', v)} min={1} max={50} />
        </div>
        <div>
          <Label>摘要截取长度</Label>
          <NumberInput value={form.post.summaryLength} onChange={(v) => updateNum('summaryLength', v)} min={50} max={500} />
        </div>
      </div>
      <div className="space-y-2">
        <Toggle
          checked={form.post.enableComment as boolean}
          onChange={(v) => updateBool('enableComment', v)}
          label="启用评论"
          description="允许读者在文章下方发表评论"
        />
        <Toggle
          checked={form.post.enableToc as boolean}
          onChange={(v) => updateBool('enableToc', v)}
          label="启用目录"
          description="自动生成文章的目录导航"
        />
      </div>
    </div>
  )
}

/** 渲染模式表单 */
function RenderForm({ form, setForm }: FormProps) {
  function updateStr(key: keyof AllSettings['render'], value: string) {
    setForm((prev) => prev ? { ...prev, render: { ...prev.render, [key]: value } } : prev)
  }
  function updateBool(key: keyof AllSettings['render'], value: boolean) {
    setForm((prev) => prev ? { ...prev, render: { ...prev.render, [key]: value } } : prev)
  }
  return (
    <div className="space-y-5">
      <SectionTitle title="渲染模式" description="控制系统的内容输出方式" />
      <div>
        <Label>渲染模式</Label>
        <SelectInput
          value={form.render.renderMode}
          onChange={(v) => updateStr('renderMode', v)}
          options={[
            { value: 'classic', label: 'Classic — Go Template 服务端渲染' },
            { value: 'headless', label: 'Headless — 纯 JSON API 输出' },
          ]}
        />
        <p className="mt-1.5 text-xs text-[var(--kite-text-muted)]">
          Classic 模式使用 Go Template 渲染 HTML；Headless 模式仅输出 JSON，适配前后端分离架构。
        </p>
      </div>
      <div>
        <Label>API 前缀</Label>
        <TextInput value={form.render.apiPrefix} onChange={(v) => updateStr('apiPrefix', v)} placeholder="/api/v1" />
      </div>
      <Toggle
        checked={form.render.enableCors as boolean}
        onChange={(v) => updateBool('enableCors', v)}
        label="启用 CORS"
        description="允许来自其他域名的 API 跨域请求"
      />
    </div>
  )
}

/** AI 集成表单 */
function AiForm({ form, setForm }: FormProps) {
  function updateStr(key: keyof AllSettings['ai'], value: string) {
    setForm((prev) => prev ? { ...prev, ai: { ...prev.ai, [key]: value } } : prev)
  }
  function updateBool(key: keyof AllSettings['ai'], value: boolean) {
    setForm((prev) => prev ? { ...prev, ai: { ...prev.ai, [key]: value } } : prev)
  }
  return (
    <div className="space-y-5">
      <SectionTitle title="AI 集成" description="接入大语言模型，自动生成文章摘要和标签" />
      <Toggle
        checked={form.ai.enabled as boolean}
        onChange={(v) => updateBool('enabled', v)}
        label="启用 AI 功能"
        description="开启后将在文章编辑时提供 AI 辅助功能"
      />

      {form.ai.enabled && (
        <div className="space-y-4 border-l-2 border-[var(--kite-border)] pl-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <Label>AI 服务商</Label>
              <SelectInput
                value={form.ai.provider}
                onChange={(v) => updateStr('provider', v)}
                options={[
                  { value: 'deepseek', label: 'DeepSeek' },
                  { value: 'openai', label: 'OpenAI' },
                ]}
              />
            </div>
            <div>
              <Label>模型名称</Label>
              <TextInput value={form.ai.model} onChange={(v) => updateStr('model', v)} placeholder="deepseek-chat" />
            </div>
          </div>
          <div>
            <Label>API Key</Label>
            <input
              type="password"
              value={form.ai.apiKey}
              onChange={(e) => updateStr('apiKey', e.target.value)}
              placeholder="sk-xxxxxxxxxxxx"
              className="h-9 w-full border border-[var(--kite-border)] bg-[var(--kite-bg)] px-3 text-sm text-[var(--kite-text)] outline-none placeholder:text-[var(--kite-text-muted)] focus:border-[var(--kite-accent)]"
            />
          </div>
          <div className="space-y-2">
            <Toggle
              checked={form.ai.autoSummary as boolean}
              onChange={(v) => updateBool('autoSummary', v)}
              label="自动生成摘要"
              description="发布文章时自动调用 AI 生成摘要"
            />
            <Toggle
              checked={form.ai.autoTag as boolean}
              onChange={(v) => updateBool('autoTag', v)}
              label="自动推荐标签"
              description="基于文章内容自动推荐相关标签"
            />
          </div>
        </div>
      )}
    </div>
  )
}
