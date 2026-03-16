import { Node, mergeAttributes } from '@tiptap/core'
import { ReactNodeViewRenderer } from '@tiptap/react'
import { ProtectedBlockView } from './ProtectedBlockView'

// 声明自定义命令类型
declare module '@tiptap/core' {
  interface Commands<ReturnType> {
    protectedBlock: {
      setProtectedBlock: () => ReturnType
      toggleProtectedBlock: () => ReturnType
    }
  }
}

/**
 * 自定义 ProtectedBlock 块级节点
 * 用于标记需要密码保护的内容区域
 * HTML 输出：<div data-protected="true" data-hint="...">内容</div>
 */
export const ProtectedBlock = Node.create({
  name: 'protectedBlock',

  group: 'block',

  content: 'block+',

  defining: true,

  addAttributes() {
    return {
      hint: {
        default: '输入密码查看隐藏内容',
        parseHTML: (element: HTMLElement) => element.getAttribute('data-hint') || '输入密码查看隐藏内容',
        renderHTML: (attributes: Record<string, string>) => ({
          'data-hint': attributes.hint,
        }),
      },
    }
  },

  parseHTML() {
    return [{ tag: 'div[data-protected="true"]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['div', mergeAttributes(HTMLAttributes, { 'data-protected': 'true' }), 0]
  },

  addNodeView() {
    return ReactNodeViewRenderer(ProtectedBlockView)
  },

  addCommands() {
    return {
      setProtectedBlock: () => ({ commands }) => {
        return commands.wrapIn(this.name)
      },
      toggleProtectedBlock: () => ({ commands, state }) => {
        const { selection } = state
        const node = selection.$head.node(-1)
        if (node?.type.name === this.name) {
          return commands.lift(this.name)
        }
        return commands.wrapIn(this.name)
      },
    }
  },
})
