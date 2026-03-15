import { NodeViewWrapper, NodeViewContent } from '@tiptap/react'
import type { NodeViewProps } from '@tiptap/react'
import { useState, useRef, useEffect, useMemo } from 'react'
import { CALLOUT_TYPES, type CalloutType } from './callout-extension'

/**
 * Callout React NodeView
 * 紧凑 blockquote 风格，标题直接可编辑
 * 空内容时只显示标题（类似 > 引用），点击后展开内容区可编辑
 */
export function CalloutView({ node, updateAttributes, editor }: NodeViewProps) {
  const type = (node.attrs.type || 'info') as CalloutType
  const title = node.attrs.title || ''
  const config = CALLOUT_TYPES[type]
  const [menuOpen, setMenuOpen] = useState(false)
  const [expanded, setExpanded] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const titleRef = useRef<HTMLInputElement>(null)
  const wrapperRef = useRef<HTMLDivElement>(null)

  const IconComponent = config.icon

  // 检测内容是否为空（node 只含一个空段落）
  const isEmpty = useMemo(() => {
    if (node.content.size === 0) return true
    if (node.content.childCount === 1) {
      const child = node.content.firstChild
      if (child && child.type.name === 'paragraph' && child.content.size === 0) {
        return true
      }
    }
    return false
  }, [node.content])

  // 空且未展开时隐藏内容区
  const hideBody = isEmpty && !expanded

  // 点击 callout 内部 → 展开；点击外部 → 收起
  useEffect(() => {
    const handleMouseDown = (e: MouseEvent) => {
      if (wrapperRef.current?.contains(e.target as Node)) {
        setExpanded(true)
      } else {
        setExpanded(false)
      }
    }
    document.addEventListener('mousedown', handleMouseDown, true)
    return () => document.removeEventListener('mousedown', handleMouseDown, true)
  }, [])

  // 点击外部关闭类型菜单
  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  return (
    <NodeViewWrapper
      ref={wrapperRef}
      className={`callout callout-${type}${hideBody ? ' callout-empty' : ''}`}
      data-callout={type}
    >
      {/* 顶部标题行 */}
      <div className="callout-header" contentEditable={false}>
        {/* 类型图标按钮（点击切换类型） */}
        <div style={{ position: 'relative' }}>
          <button
            className="callout-type-trigger"
            onClick={() => setMenuOpen(!menuOpen)}
            type="button"
            style={{ color: config.color }}
          >
            <IconComponent size={14} />
          </button>

          {/* 类型切换下拉 */}
          {menuOpen && (
            <div className="callout-type-menu" ref={menuRef}>
              {(Object.entries(CALLOUT_TYPES) as [CalloutType, typeof config][]).map(([key, val]) => {
                const ItemIcon = val.icon
                return (
                  <button
                    key={key}
                    className={`callout-type-option${key === type ? ' active' : ''}`}
                    onClick={() => { updateAttributes({ type: key }); setMenuOpen(false) }}
                    type="button"
                  >
                    <ItemIcon size={14} style={{ color: val.color }} />
                    <span>{val.label}</span>
                  </button>
                )
              })}
            </div>
          )}
        </div>

        {/* 标题 —— 始终显示为 input，直接可编辑 */}
        <input
          ref={titleRef}
          className="callout-title-input"
          style={{ color: config.color }}
          value={title}
          placeholder={config.label}
          onChange={(e) => updateAttributes({ title: e.target.value })}
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              editor.commands.focus()
            }
          }}
        />
      </div>

      {/* 内容区域 —— 空且未展开时隐藏 */}
      <NodeViewContent className="callout-body" style={hideBody ? { display: 'none' } : undefined} />
    </NodeViewWrapper>
  )
}
