import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Save, Check, Loader2, User, Mail, MapPin, Globe, FileText, Lock } from 'lucide-react'
import { useCurrentUser, useUpdateProfile, useChangePassword } from '@/hooks/use-auth'
import type { ProfileInput } from '@/hooks/use-auth'
import { cn } from '@/lib/utils'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search as HeaderSearch } from '@/components/search'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { toast } from 'sonner'
import { PasswordInput } from '@/components/PasswordInput'
import { ImageUploader } from '@/components/ImageUploader'

const inputCls = 'border-zinc-200 dark:border-zinc-700 bg-transparent rounded-md shadow-none focus-visible:ring-1 focus-visible:ring-zinc-400'

/**
 * 个人资料页面 — Vercel 风格
 */
export function ProfilePage() {
  const { data: currentUser, isLoading } = useCurrentUser()
  const updateMutation = useUpdateProfile()
  const [form, setForm] = useState<ProfileInput | null>(null)
  const [saved, setSaved] = useState(false)
  const changePasswordMutation = useChangePassword()
  const [pwdForm, setPwdForm] = useState({ old_password: '', new_password: '', confirm_password: '' })

  useEffect(() => {
    if (currentUser?.user && !form) {
      setForm({
        display_name: currentUser.user.displayName || '',
        email: currentUser.user.email || '',
        bio: currentUser.user.bio || '',
        avatar: currentUser.user.avatar || '',
        website: currentUser.user.website || '',
        location: currentUser.user.location || '',
      })
    }
  }, [currentUser, form])

  function handleSave() {
    if (!form) return
    updateMutation.mutate(form, {
      onSuccess: () => {
        toast.success('个人资料已更新')
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      },
      onError: () => {
        toast.error('保存失败，请稍后重试')
      },
    })
  }

  function updateField(key: keyof ProfileInput, value: string) {
    setForm((prev) => prev ? { ...prev, [key]: value } : prev)
  }

  function handleChangePassword() {
    if (!pwdForm.old_password) { toast.error('请输入当前密码'); return }
    if (!pwdForm.new_password) { toast.error('请输入新密码'); return }
    if (pwdForm.new_password.length < 6) { toast.error('新密码长度不能少于 6 位'); return }
    if (pwdForm.new_password !== pwdForm.confirm_password) { toast.error('两次输入的新密码不一致'); return }
    changePasswordMutation.mutate(
      { old_password: pwdForm.old_password, new_password: pwdForm.new_password },
      {
        onSuccess: () => {
          toast.success('密码已修改')
          setPwdForm({ old_password: '', new_password: '', confirm_password: '' })
        },
        onError: (err) => {
          toast.error('修改失败', { description: err.message })
        },
      }
    )
  }

  if (isLoading || !form) {
    return <div className="flex justify-center py-16"><Loader2 className="w-4 h-4 animate-spin text-zinc-400" /></div>
  }

  const initials = (form.display_name || currentUser?.user.username || 'AD').slice(0, 2).toUpperCase()

  return (
    <>
      <Header fixed>
        <HeaderSearch />
        <div className='ml-auto' />
      </Header>
      <Main>
        <div className="flex justify-between items-center mb-8">
          <div>
            <h1 className="text-lg font-semibold text-zinc-900 dark:text-zinc-50 tracking-tight">个人资料</h1>
            <p className="text-sm text-zinc-500 mt-1">管理您的个人信息</p>
          </div>
          <Button
            className="bg-zinc-900 dark:bg-zinc-100 text-white dark:text-zinc-900 rounded-md shadow-sm hover:bg-zinc-800 dark:hover:bg-zinc-200 flex gap-2 items-center h-9 text-sm"
            onClick={handleSave}
            disabled={updateMutation.isPending}
          >
            {saved ? <><Check className="w-4 h-4" /> 已保存</> : updateMutation.isPending ? '保存中…' : <><Save className="w-4 h-4" /> 保存</>}
          </Button>
        </div>

        <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-sm">
          {/* 头像预览区 */}
          <div className="flex items-center gap-5 p-8 border-b border-zinc-100 dark:border-zinc-800">
            <Avatar className="h-16 w-16 rounded-xl">
              {form.avatar && <AvatarImage src={form.avatar} alt={form.display_name} className="rounded-xl" />}
              <AvatarFallback className="rounded-xl bg-primary/10 text-primary text-xl font-medium">
                {initials}
              </AvatarFallback>
            </Avatar>
            <div>
              <p className="text-base font-semibold text-zinc-900 dark:text-zinc-50">
                {form.display_name || currentUser?.user.username}
              </p>
              <p className="text-sm text-zinc-500">@{currentUser?.user.username}</p>
            </div>
          </div>

          {/* 表单区域 */}
          <div className="p-8 space-y-7">
            <FormRow label="显示名称" icon={User}>
              <Input
                value={form.display_name}
                onChange={(e) => updateField('display_name', e.target.value)}
                placeholder="您的显示名称"
                className={cn(inputCls, 'max-w-md')}
              />
            </FormRow>

            <FormRow label="邮箱" icon={Mail}>
              <Input
                type="email"
                value={form.email}
                onChange={(e) => updateField('email', e.target.value)}
                placeholder="you@example.com"
                className={cn(inputCls, 'max-w-md')}
              />
            </FormRow>

            <FormRow label="个人简介" icon={FileText}>
              <Textarea
                value={form.bio}
                onChange={(e) => updateField('bio', e.target.value)}
                placeholder="简单介绍一下自己…"
                rows={3}
                className={cn(inputCls, 'max-w-md resize-none')}
              />
            </FormRow>

            <FormRow label="头像" icon={User}>
              <div className="max-w-md">
                <ImageUploader
                  value={form.avatar}
                  onChange={(url) => updateField('avatar', url)}
                  placeholder="上传头像图片"
                />
              </div>
            </FormRow>

            <FormRow label="个人网站" icon={Globe}>
              <Input
                value={form.website}
                onChange={(e) => updateField('website', e.target.value)}
                placeholder="https://yoursite.com"
                className={cn(inputCls, 'max-w-md')}
              />
            </FormRow>

            <FormRow label="所在地" icon={MapPin}>
              <Input
                value={form.location}
                onChange={(e) => updateField('location', e.target.value)}
                placeholder="上海"
                className={cn(inputCls, 'max-w-md')}
              />
            </FormRow>
          </div>
        </div>

        {/* 密码修改 */}
        <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg shadow-sm mt-6">
          <div className="p-8">
            <h2 className="text-base font-semibold text-zinc-900 dark:text-zinc-50 mb-1">修改密码</h2>
            <p className="text-sm text-zinc-500 mb-6">更新您的登录密码</p>
            <div className="space-y-5">
              <FormRow label="当前密码" icon={Lock}>
                <PasswordInput
                  value={pwdForm.old_password}
                  onChange={(e) => setPwdForm(p => ({ ...p, old_password: e.target.value }))}
                  placeholder="输入当前密码"
                  className={cn(inputCls, 'max-w-md')}
                />
              </FormRow>
              <FormRow label="新密码" icon={Lock}>
                <PasswordInput
                  value={pwdForm.new_password}
                  onChange={(e) => setPwdForm(p => ({ ...p, new_password: e.target.value }))}
                  placeholder="至少 6 位"
                  className={cn(inputCls, 'max-w-md')}
                />
              </FormRow>
              <FormRow label="确认新密码" icon={Lock}>
                <PasswordInput
                  value={pwdForm.confirm_password}
                  onChange={(e) => setPwdForm(p => ({ ...p, confirm_password: e.target.value }))}
                  placeholder="再次输入新密码"
                  className={cn(inputCls, 'max-w-md')}
                />
              </FormRow>
              <div className="flex items-start gap-10">
                <div className="w-36 shrink-0" />
                <Button
                  variant="outline"
                  className="rounded-md shadow-sm border-zinc-200 dark:border-zinc-700 text-sm h-9"
                  onClick={handleChangePassword}
                  disabled={changePasswordMutation.isPending}
                >
                  {changePasswordMutation.isPending ? '修改中…' : '修改密码'}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </Main>
    </>
  )
}

function FormRow({ label, icon: Icon, children }: { label: string; icon: React.ElementType; children: React.ReactNode }) {
  return (
    <div className="flex items-start gap-10">
      <div className="w-36 shrink-0 pt-2 flex items-center gap-2">
        <Icon className="w-4 h-4 text-zinc-400" />
        <p className="text-sm font-medium text-zinc-700 dark:text-zinc-300">{label}</p>
      </div>
      <div className="flex-1">{children}</div>
    </div>
  )
}
