import {
  FileText,
  LayoutDashboard,
  Newspaper,
  Palette,
  Settings,
  SlidersHorizontal,
  Tags,
  UserCog,
} from 'lucide-react'
import { useI18n } from '@/i18n'
import { useContentTypes } from '@/hooks/useContents'
import { useKindLabel } from '@/hooks/useKindLabel'
import { type SidebarData } from '../types'

const kindIcons: Record<string, React.ElementType> = {
  post: Newspaper,
  page: FileText,
}

/**
 * useSidebarData is the navigation, built from what the project has: one
 * entry per content kind it declares, and nothing it does not.
 */
export function useSidebarData(): SidebarData {
  const { t } = useI18n()
  const types = useContentTypes()
  const kindLabel = useKindLabel()

  // Until the types arrive the two built-in kinds hold the rail's shape.
  const kinds = types.data?.items.map((type) => type.kind) ?? ['post', 'page']

  return {
    navGroups: [
      {
        title: t('nav.general'),
        items: [{ title: t('nav.dashboard'), url: '/', icon: LayoutDashboard }],
      },
      {
        title: t('nav.content'),
        items: [
          ...kinds.map((kind) => ({
            title: kindLabel.many(kind),
            url: `/content/${encodeURIComponent(kind)}`,
            icon: kindIcons[kind] ?? FileText,
          })),
          { title: t('nav.taxonomies'), url: '/taxonomies', icon: Tags },
        ],
      },
      {
        title: t('nav.system'),
        items: [
          {
            title: t('nav.settings'),
            icon: Settings,
            items: [
              { title: t('nav.site'), url: '/settings', icon: SlidersHorizontal },
              { title: t('nav.theme'), url: '/settings/theme', icon: Palette },
              { title: t('settings.interface'), url: '/settings/appearance', icon: UserCog },
            ],
          },
        ],
      },
    ],
  }
}
