import { useState, useEffect, useCallback, useRef, useLayoutEffect } from 'react'
import type { SlashCommandItem } from './slash-command-extension'

interface SlashCommandMenuProps {
  items: SlashCommandItem[]
  command: (item: SlashCommandItem) => void
  clientRect: (() => DOMRect | null) | null
}

/**
 * 斜杠命令下拉菜单
 * 跟随光标位置显示，支持键盘上下选择和搜索过滤
 */
export function SlashCommandMenu({ items, command, clientRect }: SlashCommandMenuProps) {
  const [selectedIndex, setSelectedIndex] = useState(0)
  const menuRef = useRef<HTMLDivElement>(null)
  const [position, setPosition] = useState({ top: 0, left: 0 })

  // 更新菜单位置
  useLayoutEffect(() => {
    if (!clientRect) return
    const rect = clientRect()
    if (!rect) return
    setPosition({
      top: rect.bottom + 4,
      left: rect.left,
    })
  }, [clientRect])

  // items 变化时重置选中
  useEffect(() => {
    setSelectedIndex(0)
  }, [items])

  // 键盘事件
  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((prev) => (prev + 1) % items.length)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((prev) => (prev - 1 + items.length) % items.length)
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (items[selectedIndex]) {
        command(items[selectedIndex])
      }
    }
  }, [items, selectedIndex, command])

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  // 滚动选中项可见
  useEffect(() => {
    const menu = menuRef.current
    if (!menu) return
    const item = menu.children[selectedIndex] as HTMLElement
    if (item) item.scrollIntoView({ block: 'nearest' })
  }, [selectedIndex])

  if (items.length === 0) return null

  return (
    <div
      ref={menuRef}
      className="slash-command-menu"
      style={{ top: position.top, left: position.left }}
    >
      {items.map((item, index) => (
        <button
          key={item.title}
          className={`slash-command-item${index === selectedIndex ? ' selected' : ''}`}
          onClick={() => command(item)}
          onMouseEnter={() => setSelectedIndex(index)}
          type="button"
        >
          <span className="slash-command-icon">{item.icon}</span>
          <div className="slash-command-text">
            <span className="slash-command-title">{item.title}</span>
            <span className="slash-command-desc">{item.description}</span>
          </div>
        </button>
      ))}
    </div>
  )
}
