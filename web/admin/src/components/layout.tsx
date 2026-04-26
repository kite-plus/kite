import { Navigate } from 'react-router-dom'
import { FolderKanban, Images, ShieldCheck } from 'lucide-react'
import { useAuth } from '@/hooks/use-auth'
import { PageTransition } from '@/components/page-transition'
import { KiteLogo } from '@/components/kite-logo'
import { ThemeToggle } from '@/components/theme-toggle'
import { useI18n } from '@/i18n'

// The features list is built inside the component so each item picks
// up the active locale's strings from the catalogue. The icon stays
// outside the catalogue because it's a component reference, not a
// translatable string.
function useHeroFeatures() {
  const { t } = useI18n()
  return [
    {
      icon: FolderKanban,
      title: t('auth.feature1Title'),
      desc: t('auth.feature1Desc'),
    },
    {
      icon: Images,
      title: t('auth.feature2Title'),
      desc: t('auth.feature2Desc'),
    },
    {
      icon: ShieldCheck,
      title: t('auth.feature3Title'),
      desc: t('auth.feature3Desc'),
    },
  ]
}

export function AuthLayout() {
  const { user, loading } = useAuth()
  const { t } = useI18n()
  const features = useHeroFeatures()

  if (loading) {
    return (
      <div className="flex h-svh items-center justify-center bg-background">
        <KiteLogo className="size-8 animate-[splash-pulse_1.4s_ease-in-out_infinite]" />
      </div>
    )
  }

  if (user) {
    return <Navigate to="/user/dashboard" replace />
  }

  return (
    <div className="relative grid h-svh lg:grid-cols-2">
      {/* Floating theme toggle — top-right */}
      <ThemeToggle className="absolute top-4 right-4 z-20" />

      {/* Left: form */}
      <div className="flex items-center justify-center px-4 py-8 lg:p-8">
        <PageTransition />
      </div>

      {/* Right: hero — desktop only */}
      <div className="relative hidden h-full flex-col overflow-hidden border-l p-10 lg:flex">
        {/* Layered gradient backdrop */}
        <div
          aria-hidden="true"
          className="absolute inset-0 bg-gradient-to-br from-indigo-50 via-sky-50 to-rose-50 dark:from-indigo-950/40 dark:via-slate-950 dark:to-rose-950/40"
        />
        <div
          aria-hidden="true"
          className="absolute inset-0 bg-[radial-gradient(circle_at_25%_20%,rgba(129,140,248,0.28),transparent_45%),radial-gradient(circle_at_80%_75%,rgba(251,113,133,0.2),transparent_50%),radial-gradient(circle_at_60%_40%,rgba(56,189,248,0.2),transparent_50%)] dark:bg-[radial-gradient(circle_at_25%_20%,rgba(99,102,241,0.22),transparent_45%),radial-gradient(circle_at_80%_75%,rgba(244,63,94,0.18),transparent_50%),radial-gradient(circle_at_60%_40%,rgba(14,165,233,0.2),transparent_50%)]"
        />
        {/* Dot grid texture */}
        <div
          aria-hidden="true"
          className="absolute inset-0 opacity-40 [background-image:radial-gradient(circle,rgba(15,23,42,0.08)_1px,transparent_1px)] [background-size:22px_22px] [mask-image:radial-gradient(ellipse_at_center,black_40%,transparent_85%)] dark:opacity-30 dark:[background-image:radial-gradient(circle,rgba(255,255,255,0.12)_1px,transparent_1px)]"
        />

        {/* Brand */}
        <div className="relative z-10 flex items-center text-lg font-medium text-foreground">
          <KiteLogo className="me-2 size-6" />
          Kite
        </div>

        {/* Slogan + features */}
        <div className="relative z-10 mt-auto max-w-md space-y-8">
          <div>
            <h2 className="text-3xl font-semibold tracking-tight text-foreground">
              {t('auth.heroSlogan')}
            </h2>
            <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
              {t('auth.heroTagline')}
            </p>
          </div>

          <ul className="space-y-4">
            {features.map(({ icon: Icon, title, desc }) => (
              <li key={title} className="flex items-start gap-3">
                <div className="flex size-9 shrink-0 items-center justify-center rounded-lg border border-border/60 bg-background/60 text-foreground shadow-sm backdrop-blur-sm">
                  <Icon className="size-4" />
                </div>
                <div>
                  <p className="text-sm font-medium text-foreground">{title}</p>
                  <p className="mt-0.5 text-xs leading-relaxed text-muted-foreground">
                    {desc}
                  </p>
                </div>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  )
}
