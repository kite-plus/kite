import {
  FileText,
  Folder,
  LayoutDashboard,
  MonitorCog,
  Newspaper,
  Palette,
  Puzzle,
  Rocket,
  Settings,
  SlidersHorizontal,
  Tag,
  Tags,
  UserRound,
} from 'lucide-react'
import { useI18n } from '@/i18n'
import { useContentTypes, useTaxonomies } from '@/hooks/useContents'
import { useKindLabel, useTaxonomyLabel } from '@/hooks/useKindLabel'
import { type SidebarData } from '../types'

const kindIcons: Record<string, React.ElementType> = {
  post: Newspaper,
  page: FileText,
}

const taxonomyIcons: Record<string, React.ElementType> = {
  categories: Folder,
  tags: Tags,
}

/**
 * useSidebarData is the navigation, built from what the project has: one
 * entry per content kind it declares and per taxonomy, as a blog's posts,
 * pages, categories and tags, and nothing it does not.
 */
export function useSidebarData(): SidebarData {
  const { t } = useI18n()
  const types = useContentTypes()
  const kindLabel = useKindLabel()
  const taxonomies = useTaxonomies()
  const taxonomyLabel = useTaxonomyLabel()

  // Until the answers arrive the built-in kinds and taxonomies hold the rail's shape.
  const kinds = types.data?.items.map((type) => type.kind) ?? ['post', 'page']
  const names = taxonomies.data?.items.map((item) => item.name) ?? ['categories', 'tags']

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
          ...names.map((name) => ({
            title: taxonomyLabel(name),
            url: `/taxonomies/${encodeURIComponent(name)}`,
            icon: taxonomyIcons[name] ?? Tag,
          })),
        ],
      },
      {
        title: t('nav.system'),
        items: [
          { title: t('nav.deploy'), url: '/deploy', icon: Rocket },
          { title: t('nav.plugins'), url: '/plugins', icon: Puzzle },
          {
            title: t('nav.settings'),
            icon: Settings,
            items: [
              { title: t('nav.site'), url: '/settings', icon: SlidersHorizontal },
              { title: t('nav.theme'), url: '/settings/theme', icon: Palette },
              { title: t('settings.interface'), url: '/settings/appearance', icon: MonitorCog },
              { title: t('nav.account'), url: '/settings/account', icon: UserRound },
            ],
          },
        ],
      },
    ],
  }
}
