import { NodeViewWrapper, NodeViewContent } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { Lock } from 'lucide-react'

/**
 * ProtectedBlock React NodeView
 * 显示为橙色虚线边框 + 🔒 标识的加密区域
 */
export function ProtectedBlockView({ node, updateAttributes }: NodeViewProps) {
  const hint = node.attrs.hint || '输入密码查看隐藏内容'

  return (
    <NodeViewWrapper
      className="protected-block-wrapper"
      data-protected="true"
    >
      {/* 顶部标识栏 */}
      <div className="protected-block-header" contentEditable={false}>
        <Lock size={14} />
        <input
          className="protected-block-hint-input"
          value={hint}
          placeholder="输入密码查看隐藏内容"
          onChange={(e) => updateAttributes({ hint: e.target.value })}
        />
      </div>

      {/* 内容区域 —— 可编辑 */}
      <NodeViewContent className="protected-block-body" />
    </NodeViewWrapper>
  )
}
