import { createRoot } from 'react-dom/client'
import type { SuggestionProps, SuggestionKeyDownProps } from '@tiptap/suggestion'
import { SLASH_COMMANDS } from './slash-command-extension'
import type { SlashCommandItem } from './slash-command-extension'
import { SlashCommandMenu } from './SlashCommandMenu'

/**
 * 斜杠命令 suggestion 渲染配置
 * 将 React 组件挂载到 DOM，处理搜索过滤和生命周期
 */
export function slashCommandSuggestion() {
  return {
    items: ({ query }: { query: string }): SlashCommandItem[] => {
      return SLASH_COMMANDS.filter((item) =>
        item.title.toLowerCase().includes(query.toLowerCase()) ||
        item.description.toLowerCase().includes(query.toLowerCase())
      )
    },

    render: () => {
      let container: HTMLDivElement | null = null
      let root: ReturnType<typeof createRoot> | null = null
      let commandFn: ((item: SlashCommandItem) => void) | null = null

      return {
        onStart: (props: SuggestionProps) => {
          container = document.createElement('div')
          document.body.appendChild(container)
          root = createRoot(container)
          commandFn = (item: SlashCommandItem) => {
            props.command(item)
          }
          root.render(
            <SlashCommandMenu
              items={props.items as SlashCommandItem[]}
              command={commandFn}
              clientRect={props.clientRect as (() => DOMRect | null) | null}
            />
          )
        },

        onUpdate: (props: SuggestionProps) => {
          commandFn = (item: SlashCommandItem) => {
            props.command(item)
          }
          root?.render(
            <SlashCommandMenu
              items={props.items as SlashCommandItem[]}
              command={commandFn}
              clientRect={props.clientRect as (() => DOMRect | null) | null}
            />
          )
        },

        onKeyDown: (props: SuggestionKeyDownProps) => {
          if (props.event.key === 'Escape') {
            container?.remove()
            root?.unmount()
            container = null
            root = null
            return true
          }
          // 让菜单内部处理上下和 Enter
          if (['ArrowDown', 'ArrowUp', 'Enter'].includes(props.event.key)) {
            return true
          }
          return false
        },

        onExit: () => {
          container?.remove()
          root?.unmount()
          container = null
          root = null
        },
      }
    },
  }
}
