import { useCallback } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import { common, createLowlight } from 'lowlight'
import { Button, Tooltip, Divider } from '@douyinfe/semi-ui'
import {
  Bold, Italic, Strikethrough, Code, List, ListOrdered,
  Quote, Minus, Heading1, Heading2, Heading3,
  Link as LinkIcon, Image as ImageIcon, Undo2, Redo2, FileCode,
} from 'lucide-react'
import '@/styles/tiptap.css'

/** 代码高亮引擎 */
const lowlight = createLowlight(common)

interface TiptapEditorProps {
  content?: string
  onChange?: (html: string) => void
  placeholder?: string
}

/**
 * Tiptap 富文本编辑器
 * 工具栏使用 Semi Button + Tooltip
 */
export function TiptapEditor({ content = '', onChange, placeholder = '开始写作…' }: TiptapEditorProps) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false }),
      Placeholder.configure({ placeholder }),
      Image.configure({ inline: false, allowBase64: true }),
      Link.configure({ openOnClick: false, autolink: true }),
      CodeBlockLowlight.configure({ lowlight }),
    ],
    content,
    onUpdate: ({ editor }) => { onChange?.(editor.getHTML()) },
  })

  const setLink = useCallback(() => {
    if (!editor) return
    const previousUrl = editor.getAttributes('link').href
    const url = window.prompt('输入链接地址', previousUrl)
    if (url === null) return
    if (url === '') { editor.chain().focus().extendMarkRange('link').unsetLink().run(); return }
    editor.chain().focus().extendMarkRange('link').setLink({ href: url }).run()
  }, [editor])

  const addImage = useCallback(() => {
    if (!editor) return
    const url = window.prompt('输入图片地址')
    if (url) editor.chain().focus().setImage({ src: url }).run()
  }, [editor])

  if (!editor) return null

  return (
    <div className="tiptap-wrapper">
      {/* 工具栏 */}
      <div className="tiptap-toolbar">
        <ToolBtn icon={Undo2} tooltip="撤销" onClick={() => editor.chain().focus().undo().run()} disabled={!editor.can().undo()} />
        <ToolBtn icon={Redo2} tooltip="重做" onClick={() => editor.chain().focus().redo().run()} disabled={!editor.can().redo()} />

        <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

        <ToolBtn icon={Heading1} tooltip="标题 1" onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()} active={editor.isActive('heading', { level: 1 })} />
        <ToolBtn icon={Heading2} tooltip="标题 2" onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} active={editor.isActive('heading', { level: 2 })} />
        <ToolBtn icon={Heading3} tooltip="标题 3" onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()} active={editor.isActive('heading', { level: 3 })} />

        <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

        <ToolBtn icon={Bold} tooltip="加粗" onClick={() => editor.chain().focus().toggleBold().run()} active={editor.isActive('bold')} />
        <ToolBtn icon={Italic} tooltip="斜体" onClick={() => editor.chain().focus().toggleItalic().run()} active={editor.isActive('italic')} />
        <ToolBtn icon={Strikethrough} tooltip="删除线" onClick={() => editor.chain().focus().toggleStrike().run()} active={editor.isActive('strike')} />
        <ToolBtn icon={Code} tooltip="行内代码" onClick={() => editor.chain().focus().toggleCode().run()} active={editor.isActive('code')} />

        <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

        <ToolBtn icon={List} tooltip="无序列表" onClick={() => editor.chain().focus().toggleBulletList().run()} active={editor.isActive('bulletList')} />
        <ToolBtn icon={ListOrdered} tooltip="有序列表" onClick={() => editor.chain().focus().toggleOrderedList().run()} active={editor.isActive('orderedList')} />
        <ToolBtn icon={Quote} tooltip="引用" onClick={() => editor.chain().focus().toggleBlockquote().run()} active={editor.isActive('blockquote')} />
        <ToolBtn icon={FileCode} tooltip="代码块" onClick={() => editor.chain().focus().toggleCodeBlock().run()} active={editor.isActive('codeBlock')} />

        <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

        <ToolBtn icon={LinkIcon} tooltip="链接" onClick={setLink} active={editor.isActive('link')} />
        <ToolBtn icon={ImageIcon} tooltip="图片" onClick={addImage} />
        <ToolBtn icon={Minus} tooltip="分隔线" onClick={() => editor.chain().focus().setHorizontalRule().run()} />
      </div>

      {/* 编辑区域 */}
      <EditorContent editor={editor} />
    </div>
  )
}

/* ========== 工具栏按钮 ========== */
interface ToolBtnProps {
  icon: React.ComponentType<{ className?: string }>
  tooltip: string
  onClick: () => void
  active?: boolean
  disabled?: boolean
}

function ToolBtn({ icon: Icon, tooltip, onClick, active, disabled }: ToolBtnProps) {
  return (
    <Tooltip content={tooltip} position="bottom">
      <Button
        type="tertiary"
        theme={active ? 'solid' : 'borderless'}
        size="small"
        onClick={onClick}
        disabled={disabled}
        icon={<Icon className="h-4 w-4" />}
        style={{ width: 32, height: 32, padding: 0 }}
      />
    </Tooltip>
  )
}
