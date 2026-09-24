import React, { useDeferredValue, useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import {
  ArrowRight,
  ChevronRight,
  ExternalLink,
  FileText,
  Laptop,
  Moon,
  Plus,
  Sun,
} from 'lucide-react'
import { api, unwrap } from '@/api/client'
import { useI18n } from '@/i18n'
import { useContentTypes, useSite } from '@/hooks/useContents'
import { useKindLabel } from '@/hooks/useKindLabel'
import { siteHome } from '@/lib/links'
import { setTheme } from '@/lib/theme'
import { useSearch } from '@/context/search-provider'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { StatusLabel } from './StatusLabel'
import { useSidebarData } from './layout/data/sidebar-data'
import { ScrollArea } from './ui/scroll-area'

type Entry = {
  key: string
  label: React.ReactNode
  text: string
  icon: React.ElementType
  run: () => void
}

/**
 * The palette behind the header's search: find content, go somewhere, or do
 * something. The server answers the content search, so cmdk's own filter is
 * off and the fixed entries are matched here.
 */
export function CommandMenu() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { open, setOpen } = useSearch()
  const sidebarData = useSidebarData()
  const types = useContentTypes()
  const kindLabel = useKindLabel()
  const site = useSite()

  const [query, setQuery] = useState('')
  const deferred = useDeferredValue(query.trim())

  const found = useQuery({
    queryKey: ['contents', 'palette', deferred],
    enabled: open && deferred.length > 0,
    queryFn: async () =>
      unwrap(
        await api.GET('/contents', {
          params: { query: { q: deferred, limit: 8 } },
        })
      ),
  })

  const places: Entry[] = sidebarData.navGroups.flatMap((group) =>
    group.items.flatMap((item): Entry[] =>
      item.items
        ? item.items.map((sub) => ({
            key: `go:${sub.url}`,
            label: (
              <>
                {item.title} <ChevronRight /> {sub.title}
              </>
            ),
            text: `${item.title} ${sub.title}`,
            icon: ArrowRight,
            run: () => void navigate({ to: sub.url }),
          }))
        : [
            {
              key: `go:${item.url}`,
              label: item.title,
              text: item.title,
              icon: ArrowRight,
              run: () => void navigate({ to: item.url }),
            },
          ]
    )
  )
  const kinds = types.data?.items.map((type) => type.kind) ?? []
  const actions: Entry[] = [
    ...kinds.map((kind) => {
      const label = t('list.newKind', { kind: kindLabel.one(kind) })
      return {
        key: `new:${kind}`,
        label,
        text: label,
        icon: Plus,
        run: () =>
          void navigate({
            to: '/content/$kind/$id',
            params: { kind, id: 'new' },
          }),
      }
    }),
    {
      key: 'site',
      label: t('nav.viewSite'),
      text: t('nav.viewSite'),
      icon: ExternalLink,
      run: () => void window.open(siteHome(site.data), '_blank', 'noreferrer'),
    },
  ]
  const themes: Entry[] = [
    { key: 'theme:light', text: t('settings.light'), icon: Sun, run: () => setTheme('light') },
    { key: 'theme:dark', text: t('settings.dark'), icon: Moon, run: () => setTheme('dark') },
    { key: 'theme:system', text: t('settings.system'), icon: Laptop, run: () => setTheme('system') },
  ].map((entry) => ({ ...entry, label: entry.text }))

  const matches = (entry: Entry) =>
    entry.text.toLowerCase().includes(deferred.toLowerCase())
  const groups = [
    { heading: t('palette.goTo'), entries: places.filter(matches) },
    { heading: t('palette.actions'), entries: actions.filter(matches) },
    { heading: t('palette.theme'), entries: themes.filter(matches) },
  ].filter((group) => group.entries.length > 0)
  const results = found.data?.items ?? []

  // cmdk highlights the first row only for rows it filtered itself. These
  // arrive late, so the highlight is moved to the top whenever they change.
  const first = results[0]?.id ?? groups[0]?.entries[0]?.key ?? ''
  const [active, setActive] = useState('')
  useEffect(() => setActive(first), [first])

  const runCommand = (command: () => unknown) => {
    setOpen(false)
    setQuery('')
    command()
  }

  return (
    <CommandDialog
      modal
      open={open}
      onOpenChange={setOpen}
      title={t('palette.title')}
      description={t('palette.description')}
      commandProps={{ shouldFilter: false, value: active, onValueChange: setActive }}
    >
      <CommandInput
        value={query}
        onValueChange={setQuery}
        placeholder={t('palette.placeholder')}
      />
      <CommandList>
        <ScrollArea type='hover' className='h-72 pe-1'>
          <CommandEmpty>{t('palette.empty')}</CommandEmpty>
          {results.length > 0 && (
            <CommandGroup heading={t('nav.content')}>
              {results.map((item) => (
                <CommandItem
                  key={item.id}
                  value={item.id}
                  onSelect={() =>
                    runCommand(() =>
                      navigate({
                        to: '/content/$kind/$id',
                        params: { kind: item.kind, id: item.id },
                      })
                    )
                  }
                >
                  <FileText />
                  <span className='truncate'>{item.title || item.slug}</span>
                  <StatusLabel status={item.status} className='ms-auto shrink-0 gap-1.5 text-xs' />
                </CommandItem>
              ))}
            </CommandGroup>
          )}
          {groups.map((group, i) => (
            <React.Fragment key={group.heading}>
              {(i > 0 || results.length > 0) && <CommandSeparator />}
              <CommandGroup heading={group.heading}>
                {group.entries.map((entry) => (
                  <CommandItem
                    key={entry.key}
                    value={entry.key}
                    onSelect={() => runCommand(entry.run)}
                  >
                    {entry.icon === ArrowRight ? (
                      <div className='flex size-4 items-center justify-center'>
                        <ArrowRight className='size-2 text-muted-foreground/80' />
                      </div>
                    ) : (
                      <entry.icon />
                    )}
                    {entry.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            </React.Fragment>
          ))}
        </ScrollArea>
      </CommandList>
    </CommandDialog>
  )
}
