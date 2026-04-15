import { useState } from 'react'
import { useNavigate } from 'react-router'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { api } from '@/lib/api'

export default function LoginPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [rememberMe, setRememberMe] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setLoading(true)
    try {
      await api.login({ username, password, rememberMe })
      queryClient.setQueryData(['auth', 'check'], { required: true, authenticated: true })
      navigate('/dashboard', { replace: true })
    } catch {
      setError(t('auth.invalidCredentials'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex h-screen items-center justify-center overflow-hidden bg-muted/40">
      {/* Cyan ambient — top-left */}
      <div className="pointer-events-none absolute -left-[10%] -top-[10%] h-[50vw] w-[50vw] rounded-full bg-primary opacity-[0.06] blur-[150px] dark:opacity-[0.12]" />
      {/* Amber ambient — bottom-right */}
      <div className="pointer-events-none absolute -bottom-[10%] -right-[10%] h-[40vw] w-[40vw] rounded-full bg-secondary opacity-[0.04] blur-[120px] dark:opacity-[0.08]" />
      <div className="cyber-bg-dots pointer-events-none absolute inset-0" />
      <div className="relative z-10 w-full max-w-sm space-y-6 rounded-xl border bg-background/85 p-8 shadow-xl backdrop-blur-xl">
        <div className="flex flex-col items-center gap-3">
          <img src="/favicon.svg" alt="OpsTunnel" className="h-16 w-16 rounded-xl" />
          <div className="text-center">
            <h1 className="text-2xl font-bold">OpsTunnel</h1>
            <p className="mt-1 text-sm text-muted-foreground">{t('auth.loginSubtitle')}</p>
          </div>
        </div>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="username">{t('auth.username')}</Label>
            <Input
              id="username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoFocus
              autoComplete="username"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">{t('auth.password')}</Label>
            <Input
              id="password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={rememberMe}
              onChange={(e) => setRememberMe(e.target.checked)}
              className="rounded border-input"
            />
            {t('auth.rememberMe')}
          </label>
          {error && (
            <p className="text-sm text-destructive">{error}</p>
          )}
          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? t('auth.signingIn') : t('auth.signIn')}
          </Button>
        </form>
      </div>
    </div>
  )
}
