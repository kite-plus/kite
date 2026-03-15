import { Extension } from '@tiptap/core'
import Suggestion from '@tiptap/suggestion'
import type { Editor, Range } from '@tiptap/core'

/** 斜杠命令项定义 */
export interface SlashCommandItem {
  title: string
  description: string
  icon: string
  command: (editor: Editor, range: Range) => void
}

/** 所有可用的斜杠命令 */
export const SLASH_COMMANDS: SlashCommandItem[] = [
  {
    title: '标题 1',
    description: '大标题',
    icon: 'H₁',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleHeading({ level: 1 }).run()
    },
  },
  {
    title: '标题 2',
    description: '中标题',
    icon: 'H₂',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleHeading({ level: 2 }).run()
    },
  },
  {
    title: '标题 3',
    description: '小标题',
    icon: 'H₃',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleHeading({ level: 3 }).run()
    },
  },
  {
    title: '无序列表',
    description: '项目符号列表',
    icon: '•',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleBulletList().run()
    },
  },
  {
    title: '有序列表',
    description: '编号列表',
    icon: '1.',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleOrderedList().run()
    },
  },
  {
    title: '任务列表',
    description: '待办清单',
    icon: '☑',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleTaskList().run()
    },
  },
  {
    title: '引用',
    description: '引用块',
    icon: '❝',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleBlockquote().run()
    },
  },
  {
    title: '代码块',
    description: '代码片段',
    icon: '</>',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).toggleCodeBlock().run()
    },
  },
  {
    title: '表格',
    description: '插入表格',
    icon: '⊞',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
    },
  },
  {
    title: '分割线',
    description: '水平分割线',
    icon: '—',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).setHorizontalRule().run()
    },
  },
  {
    title: '提示',
    description: '蓝色提示框',
    icon: '💡',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).setCallout('info').run()
    },
  },
  {
    title: '警告',
    description: '黄色警告框',
    icon: '⚠',
    command: (editor, range) => {
      editor.chain().focus().deleteRange(range).setCallout('warning').run()
    },
  },
]

/**
 * 斜杠命令扩展
 * 输入 / 触发快速插入菜单
 */
export const SlashCommand = Extension.create({
  name: 'slashCommand',

  addOptions() {
    return {
      suggestion: {
        char: '/',
        startOfLine: false,
        command: ({ editor, range, props }: { editor: Editor; range: Range; props: SlashCommandItem }) => {
          props.command(editor, range)
        },
      },
    }
  },

  addProseMirrorPlugins() {
    return [
      Suggestion({
        editor: this.editor,
        ...this.options.suggestion,
      }),
    ]
  },
})
