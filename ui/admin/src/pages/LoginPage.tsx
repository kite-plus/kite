import { useState } from 'react'
import { Card, Button, Input, Typography, Toast } from '@douyinfe/semi-ui'
import { IconUser, IconLock } from '@douyinfe/semi-icons'
import { useLogin } from '@/hooks/use-auth'

const { Title, Text } = Typography

/**
 * 管理员登录页面
 */
export function LoginPage() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const loginMutation = useLogin()

  function handleSubmit() {
    if (!username.trim() || !password) return
    loginMutation.mutate({ username: username.trim(), password }, {
      onError: (err) => {
        Toast.error(err.message === 'invalid username or password' ? '用户名或密码错误' : '登录失败，请重试')
      },
    })
  }

  return (
    <div style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'var(--semi-color-bg-0)',
    }}>
      <Card
        style={{
          width: 400,
          boxShadow: 'var(--semi-shadow-elevated)',
          borderRadius: 12,
        }}
        bodyStyle={{ padding: '40px 32px' }}
      >
        {/* Logo & 标题 */}
        <div style={{ textAlign: 'center', marginBottom: 32 }}>
          <div style={{
            width: 56, height: 56, borderRadius: '50%',
            background: 'var(--semi-color-primary-light-default)',
            display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
            marginBottom: 16,
          }}>
            <span style={{ fontSize: 28 }}>🪁</span>
          </div>
          <Title heading={4} style={{ marginBottom: 4 }}>Kite 管理后台</Title>
          <Text type="tertiary">请输入管理员账号登录</Text>
        </div>

        {/* 表单 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
          <div>
            <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>用户名</Text>
            <Input
              prefix={<IconUser />}
              value={username}
              onChange={(v) => setUsername(v)}
              placeholder="admin"
              size="large"
              onEnterPress={handleSubmit}
              autoFocus
            />
          </div>
          <div>
            <Text size="small" type="tertiary" style={{ display: 'block', marginBottom: 6 }}>密码</Text>
            <Input
              prefix={<IconLock />}
              type="password"
              value={password}
              onChange={(v) => setPassword(v)}
              placeholder="••••••••"
              size="large"
              onEnterPress={handleSubmit}
            />
          </div>
          <Button
            theme="solid"
            size="large"
            block
            onClick={handleSubmit}
            loading={loginMutation.isPending}
            disabled={!username.trim() || !password}
            style={{ marginTop: 8 }}
          >
            登录
          </Button>
        </div>

        {/* 底部 */}
        <div style={{ textAlign: 'center', marginTop: 24 }}>
          <Text type="quaternary" size="small">Kite Blog — 轻量级 AI 原生博客引擎</Text>
        </div>
      </Card>
    </div>
  )
}
