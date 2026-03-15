import { useCallback, useState, useRef, useEffect } from 'react'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import Placeholder from '@tiptap/extension-placeholder'
import { ResizableImage } from '@/extensions/resizable-image/resizable-image-extension'
import Link from '@tiptap/extension-link'
import CodeBlockLowlight from '@tiptap/extension-code-block-lowlight'
import Underline from '@tiptap/extension-underline'
import Highlight from '@tiptap/extension-highlight'
import Superscript from '@tiptap/extension-superscript'
import Subscript from '@tiptap/extension-subscript'
import TextAlign from '@tiptap/extension-text-align'
import TaskList from '@tiptap/extension-task-list'
import TaskItem from '@tiptap/extension-task-item'
import { Table } from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableHeader from '@tiptap/extension-table-header'
import TableCell from '@tiptap/extension-table-cell'
import Color from '@tiptap/extension-color'
import { TextStyle } from '@tiptap/extension-text-style'
import { common, createLowlight } from 'lowlight'
import { Divider, Modal, Input } from '@douyinfe/semi-ui'
import TurndownService from 'turndown'
import { marked } from 'marked'
import {
  Bold, Italic, Strikethrough, Code, List, ListOrdered,
  Quote, Minus, Heading1, Heading2, Heading3,
  Link as LinkIcon, Image as ImageIcon, Undo2, Redo2, FileCode,
  CodeXml, UnderlineIcon, Highlighter, SuperscriptIcon, SubscriptIcon,
  AlignLeft, AlignCenter, AlignRight, AlignJustify,
  ListTodo, TableIcon, Upload, Lightbulb,
} from 'lucide-react'
import { Callout, CALLOUT_TYPES, type CalloutType } from '@/extensions/callout/callout-extension'
import { SlashCommand } from '@/extensions/slash-command/slash-command-extension'
import { slashCommandSuggestion } from '@/extensions/slash-command/suggestion'
import '@/styles/tiptap.css'
import '@/styles/callout.css'

/** 代码高亮引擎 */
const lowlight = createLowlight(common)

/** HTML → Markdown 转换器 */
const turndown = new TurndownService({
  headingStyle: 'atx',
  codeBlockStyle: 'fenced',
  bulletListMarker: '-',
  emDelimiter: '*',
  strongDelimiter: '**',
  hr: '---',
})

turndown.addRule('strikethrough', {
  filter: ['del', 's'],
  replacement: (content) => `~~${content}~~`,
})

/** Callout HTML → Markdown ::: 容器语法（支持自定义标题） */
turndown.addRule('callout', {
  filter: (node) => {
    return node.nodeName === 'DIV' && node.hasAttribute('data-callout')
  },
  replacement: (content, node) => {
    const el = node as HTMLElement
    const type = el.getAttribute('data-callout') || 'info'
    const title = el.getAttribute('data-callout-title') || ''
    const header = title ? `${type} ${title}` : type
    const trimmed = content.trim()
    return `\n\n::: ${header}\n${trimmed}\n:::\n\n`
  },
})

/** Markdown ::: 容器语法 → Callout HTML（支持自定义标题） */
const calloutExtension = {
  name: 'calloutContainer',
  level: 'block' as const,
  start(src: string) { return src.match(/^:::\s/)?.index },
  tokenizer(src: string) {
    // 匹配 ::: type 可选标题\n内容\n:::
    const match = src.match(/^:::\s*(\w+)(?: ([^\n]+))?\n([\s\S]*?)\n:::\s*(?:\n|$)/)
    if (match) {
      return {
        type: 'calloutContainer',
        raw: match[0],
        calloutType: match[1],
        calloutTitle: match[2]?.trim() || '',
        text: match[3].trim(),
      }
    }
    return undefined
  },
  renderer(token: { calloutType: string; calloutTitle: string; text: string }) {
    const inner = marked.parse(token.text, { async: false }) as string
    const titleAttr = token.calloutTitle ? ` data-callout-title="${token.calloutTitle}"` : ''
    return `<div data-callout="${token.calloutType}"${titleAttr} class="callout callout-${token.calloutType}">${inner}</div>`
  },
}
marked.use({ extensions: [calloutExtension] })

interface TiptapEditorProps {
  content?: string
  onChange?: (html: string) => void
  placeholder?: string
}

/**
 * Tiptap 富文本编辑器
 * 对标 Tiptap 官方演示的完整工具栏
 */
export function TiptapEditor({ content = '', onChange, placeholder = '开始写作…' }: TiptapEditorProps) {
  const [imageModalVisible, setImageModalVisible] = useState(false)
  const [imageUrl, setImageUrl] = useState('')
  const [imageTab, setImageTab] = useState<'url' | 'upload'>('url')
  const [uploadPreview, setUploadPreview] = useState('')
  const [dragOver, setDragOver] = useState(false)
  const [linkModalVisible, setLinkModalVisible] = useState(false)
  const [linkUrl, setLinkUrl] = useState('')
  const [sourceMode, setSourceMode] = useState(false)
  const [sourceCode, setSourceCode] = useState('')
  const sourceRef = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [calloutMenuOpen, setCalloutMenuOpen] = useState(false)
  const calloutBtnRef = useRef<HTMLDivElement>(null)
  const [tableMenuOpen, setTableMenuOpen] = useState(false)
  const [tableHover, setTableHover] = useState({ row: 0, col: 0 })
  const [customRows, setCustomRows] = useState(3)
  const [customCols, setCustomCols] = useState(3)
  const tableBtnRef = useRef<HTMLDivElement>(null)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({ codeBlock: false, link: false }),
      Placeholder.configure({ placeholder }),
      ResizableImage.configure({ inline: false, allowBase64: true }),
      Link.configure({ openOnClick: false, autolink: true }),
      CodeBlockLowlight.configure({ lowlight }),
      Underline,
      Highlight.configure({ multicolor: false }),
      Superscript,
      Subscript,
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      TaskList,
      TaskItem.configure({ nested: true }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
      TextStyle,
      Color,
      Callout,
      SlashCommand.configure({ suggestion: slashCommandSuggestion() }),
    ],
    content,
    onUpdate: ({ editor }) => { onChange?.(editor.getHTML()) },
  })

  /** 切换 Markdown 源码模式 */
  const toggleSourceMode = useCallback(() => {
    if (!editor) return
    if (!sourceMode) {
      const html = editor.getHTML()
      const markdown = turndown.turndown(html)
      setSourceCode(markdown)
    } else {
      const html = marked.parse(sourceCode, { async: false }) as string
      editor.commands.setContent(html, { emitUpdate: true })
      onChange?.(html)
    }
    setSourceMode(!sourceMode)
  }, [editor, sourceMode, sourceCode, onChange])

  useEffect(() => {
    if (sourceMode && sourceRef.current) sourceRef.current.focus()
  }, [sourceMode])

  /** 打开链接对话框 */
  const openLinkModal = useCallback(() => {
    if (!editor) return
    const previousUrl = editor.getAttributes('link').href || ''
    setLinkUrl(previousUrl)
    setLinkModalVisible(true)
  }, [editor])

  const confirmLink = useCallback(() => {
    if (!editor) return
    if (linkUrl === '') {
      editor.chain().focus().extendMarkRange('link').unsetLink().run()
    } else {
      editor.chain().focus().extendMarkRange('link').setLink({ href: linkUrl }).run()
    }
    setLinkModalVisible(false)
    setLinkUrl('')
  }, [editor, linkUrl])

  const openImageModal = useCallback(() => {
    setImageUrl('')
    setUploadPreview('')
    setImageTab('url')
    setImageModalVisible(true)
  }, [])

  /** 确认插入图片（URL 或上传） */
  const confirmImage = useCallback(() => {
    if (!editor) return
    const src = imageTab === 'url' ? imageUrl.trim() : uploadPreview
    if (!src) return
    editor.chain().focus().setImage({ src }).run()
    setImageModalVisible(false)
    setImageUrl('')
    setUploadPreview('')
  }, [editor, imageUrl, uploadPreview, imageTab])

  /** 处理文件上传（转 base64） */
  const handleFileUpload = useCallback((file: File) => {
    if (!file.type.startsWith('image/')) return
    const reader = new FileReader()
    reader.onload = (e) => {
      const result = e.target?.result as string
      if (result) setUploadPreview(result)
    }
    reader.readAsDataURL(file)
  }, [])

  /** 拖拽处理 */
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFileUpload(file)
  }, [handleFileUpload])

  const handleFileChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) handleFileUpload(file)
    e.target.value = '' // 重置 input 允许重复选同一文件
  }, [handleFileUpload])

  /** 插入指定尺寸的表格 */
  const insertTable = useCallback((rows: number, cols: number) => {
    if (!editor) return
    editor.chain().focus().insertTable({ rows, cols, withHeaderRow: true }).run()
    setTableMenuOpen(false)
    setTableHover({ row: 0, col: 0 })
  }, [editor])

  // 点击外部关闭表格选择器
  useEffect(() => {
    if (!tableMenuOpen) return
    const handler = (e: MouseEvent) => {
      if (tableBtnRef.current && !tableBtnRef.current.contains(e.target as Node)) {
        setTableMenuOpen(false)
        setTableHover({ row: 0, col: 0 })
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [tableMenuOpen])

  if (!editor) return null

  return (
    <div className="tiptap-wrapper">
      {/* 工具栏 */}
      <div className="tiptap-toolbar">
        {!sourceMode && (
          <>
            {/* 历史操作 */}
            <ToolBtn icon={Undo2} tooltip="撤销" onClick={() => editor.chain().focus().undo().run()} disabled={!editor.can().undo()} />
            <ToolBtn icon={Redo2} tooltip="重做" onClick={() => editor.chain().focus().redo().run()} disabled={!editor.can().redo()} />

            <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

            {/* 标题 */}
            <ToolBtn icon={Heading1} tooltip="标题 1" onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()} active={editor.isActive('heading', { level: 1 })} />
            <ToolBtn icon={Heading2} tooltip="标题 2" onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()} active={editor.isActive('heading', { level: 2 })} />
            <ToolBtn icon={Heading3} tooltip="标题 3" onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()} active={editor.isActive('heading', { level: 3 })} />

            <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

            {/* 文本格式 */}
            <ToolBtn icon={Bold} tooltip="加粗" onClick={() => editor.chain().focus().toggleBold().run()} active={editor.isActive('bold')} />
            <ToolBtn icon={Italic} tooltip="斜体" onClick={() => editor.chain().focus().toggleItalic().run()} active={editor.isActive('italic')} />
            <ToolBtn icon={UnderlineIcon} tooltip="下划线" onClick={() => editor.chain().focus().toggleUnderline().run()} active={editor.isActive('underline')} />
            <ToolBtn icon={Strikethrough} tooltip="删除线" onClick={() => editor.chain().focus().toggleStrike().run()} active={editor.isActive('strike')} />
            <ToolBtn icon={Code} tooltip="行内代码" onClick={() => editor.chain().focus().toggleCode().run()} active={editor.isActive('code')} />
            <ToolBtn icon={Highlighter} tooltip="高亮" onClick={() => editor.chain().focus().toggleHighlight().run()} active={editor.isActive('highlight')} />

            <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

            {/* 上标 / 下标 */}
            <ToolBtn icon={SuperscriptIcon} tooltip="上标" onClick={() => editor.chain().focus().toggleSuperscript().run()} active={editor.isActive('superscript')} />
            <ToolBtn icon={SubscriptIcon} tooltip="下标" onClick={() => editor.chain().focus().toggleSubscript().run()} active={editor.isActive('subscript')} />

            <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

            {/* 对齐 */}
            <ToolBtn icon={AlignLeft} tooltip="左对齐" onClick={() => editor.chain().focus().setTextAlign('left').run()} active={editor.isActive({ textAlign: 'left' })} />
            <ToolBtn icon={AlignCenter} tooltip="居中" onClick={() => editor.chain().focus().setTextAlign('center').run()} active={editor.isActive({ textAlign: 'center' })} />
            <ToolBtn icon={AlignRight} tooltip="右对齐" onClick={() => editor.chain().focus().setTextAlign('right').run()} active={editor.isActive({ textAlign: 'right' })} />
            <ToolBtn icon={AlignJustify} tooltip="两端对齐" onClick={() => editor.chain().focus().setTextAlign('justify').run()} active={editor.isActive({ textAlign: 'justify' })} />

            <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

            {/* 列表 */}
            <ToolBtn icon={List} tooltip="无序列表" onClick={() => editor.chain().focus().toggleBulletList().run()} active={editor.isActive('bulletList')} />
            <ToolBtn icon={ListOrdered} tooltip="有序列表" onClick={() => editor.chain().focus().toggleOrderedList().run()} active={editor.isActive('orderedList')} />
            <ToolBtn icon={ListTodo} tooltip="任务列表" onClick={() => editor.chain().focus().toggleTaskList().run()} active={editor.isActive('taskList')} />

            <Divider layout="vertical" style={{ margin: '0 4px', height: 20 }} />

            {/* 块级元素 */}
            <ToolBtn icon={Quote} tooltip="引用" onClick={() => editor.chain().focus().toggleBlockquote().run()} active={editor.isActive('blockquote')} />
            <ToolBtn icon={FileCode} tooltip="代码块" onClick={() => editor.chain().focus().toggleCodeBlock().run()} active={editor.isActive('codeBlock')} />
            {/* 表格矩阵选择器 */}
            <div style={{ position: 'relative' }} ref={tableBtnRef}>
              <ToolBtn icon={TableIcon} tooltip="表格" onClick={() => setTableMenuOpen(!tableMenuOpen)} />
              {tableMenuOpen && (
                <div className="table-grid-picker">
                  <div className="table-grid-label">
                    {tableHover.row > 0 ? `${tableHover.row} × ${tableHover.col}` : '选择表格尺寸'}
                  </div>
                  <div className="table-grid">
                    {Array.from({ length: 8 }, (_, r) =>
                      Array.from({ length: 6 }, (_, c) => {
                        const row = r + 1
                        const col = c + 1
                        const active = row <= tableHover.row && col <= tableHover.col
                        return (
                          <div
                            key={`${r}-${c}`}
                            className={`table-grid-cell${active ? ' active' : ''}`}
                            onMouseEnter={() => setTableHover({ row, col })}
                            onClick={() => insertTable(row, col)}
                          />
                        )
                      })
                    )}
                  </div>
                  {/* 自定义行列输入 */}
                  <div className="table-grid-custom">
                    <div className="table-grid-custom-row">
                      <input
                        type="number"
                        min={1}
                        max={99}
                        value={customRows}
                        onChange={(e) => setCustomRows(Math.max(1, +e.target.value))}
                        className="table-grid-input"
                      />
                      <span className="table-grid-x">×</span>
                      <input
                        type="number"
                        min={1}
                        max={99}
                        value={customCols}
                        onChange={(e) => setCustomCols(Math.max(1, +e.target.value))}
                        className="table-grid-input"
                      />
                      <button
                        className="table-grid-confirm"
                        onClick={() => insertTable(customRows, customCols)}
                        type="button"
                      >
                        插入
                      </button>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Callout 下拉按钮 */}
            <div style={{ position: 'relative' }} ref={calloutBtnRef}>
              <button
                className={`tiptap-tool-btn${editor.isActive('callout') ? ' active' : ''}`}
                onClick={() => setCalloutMenuOpen(!calloutMenuOpen)}
                data-tooltip="引用块"
                type="button"
              >
                <Lightbulb className="h-4 w-4" />
              </button>
              {calloutMenuOpen && (
                <div className="callout-toolbar-menu">
                  {(Object.entries(CALLOUT_TYPES) as [CalloutType, (typeof CALLOUT_TYPES)[CalloutType]][]).map(([key, val]) => {
                    const ItemIcon = val.icon
                    return (
                      <button
                        key={key}
                        className={`callout-toolbar-option${editor.isActive('callout', { type: key }) ? ' active' : ''}`}
                        onClick={() => {
                          (editor.commands as any).toggleCallout(key)
                          setCalloutMenuOpen(false)
                        }}
                        type="button"
                      >
                        <ItemIcon size={15} style={{ color: val.color }} />
                        <span>{val.label}</span>
                      </button>
                    )
                  })}
                </div>
              )}
            </div>

            <ToolBtn icon={LinkIcon} tooltip="链接" onClick={openLinkModal} active={editor.isActive('link')} />
            <ToolBtn icon={ImageIcon} tooltip="图片" onClick={openImageModal} />
            <ToolBtn icon={Minus} tooltip="分隔线" onClick={() => editor.chain().focus().setHorizontalRule().run()} />
          </>
        )}

        {sourceMode && (
          <span className="tiptap-source-label">Markdown 源码模式</span>
        )}

        <div style={{ marginLeft: 'auto' }}>
          <ToolBtn icon={CodeXml} tooltip={sourceMode ? '退出源码' : '源码模式'} onClick={toggleSourceMode} active={sourceMode} />
        </div>
      </div>

      {/* 编辑区域 */}
      {sourceMode ? (
        <textarea
          ref={sourceRef}
          className="tiptap-source-editor"
          value={sourceCode}
          onChange={(e) => setSourceCode(e.target.value)}
          spellCheck={false}
        />
      ) : (
        <EditorContent editor={editor} />
      )}

      {/* 底部状态栏 */}
      {!sourceMode && editor && (
        <div className="tiptap-statusbar">
          <span>{editor.storage.characterCount?.words?.() ?? editor.getText().split(/\s+/).filter(Boolean).length} 字</span>
          <span>{editor.storage.characterCount?.characters?.() ?? editor.getText().length} 字符</span>
        </div>
      )}

      {/* 链接对话框 */}
      <Modal
        title="插入链接"
        visible={linkModalVisible}
        onOk={confirmLink}
        onCancel={() => { setLinkModalVisible(false); setLinkUrl('') }}
        okText="确认"
        cancelText="取消"
        width={480}
        centered
        maskClosable={false}
      >
        <Input
          value={linkUrl}
          onChange={setLinkUrl}
          placeholder="https://example.com"
          prefix="🔗"
          size="large"
          autoFocus
          onEnterPress={confirmLink}
        />
      </Modal>

      {/* 图片对话框 */}
      <Modal
        title="插入图片"
        visible={imageModalVisible}
        onOk={confirmImage}
        onCancel={() => { setImageModalVisible(false); setImageUrl(''); setUploadPreview('') }}
        okText="插入"
        cancelText="取消"
        width={520}
        centered
        maskClosable={false}
        okButtonProps={{ disabled: imageTab === 'url' ? !imageUrl.trim() : !uploadPreview }}
      >
        {/* Tab 切换 */}
        <div className="img-upload-tabs">
          <button className={`img-upload-tab${imageTab === 'url' ? ' active' : ''}`} onClick={() => setImageTab('url')} type="button">
            🔗 外部链接
          </button>
          <button className={`img-upload-tab${imageTab === 'upload' ? ' active' : ''}`} onClick={() => setImageTab('upload')} type="button">
            <Upload className="h-3.5 w-3.5" style={{ marginRight: 4 }} /> 本地上传
          </button>
        </div>

        {/* URL 模式 */}
        {imageTab === 'url' && (
          <>
            <Input
              value={imageUrl}
              onChange={setImageUrl}
              placeholder="https://example.com/image.jpg"
              prefix="🖼️"
              size="large"
              autoFocus
              onEnterPress={confirmImage}
            />
            {imageUrl.trim() && (
              <div className="img-preview-box">
                <img
                  src={imageUrl.trim()}
                  alt="预览"
                  onError={(e) => { (e.target as HTMLImageElement).style.display = 'none' }}
                  onLoad={(e) => { (e.target as HTMLImageElement).style.display = 'block' }}
                />
              </div>
            )}
          </>
        )}

        {/* 上传模式 */}
        {imageTab === 'upload' && (
          <>
            <input ref={fileInputRef} type="file" accept="image/*" onChange={handleFileChange} style={{ display: 'none' }} />
            {!uploadPreview ? (
              <div
                className={`img-dropzone${dragOver ? ' drag-over' : ''}`}
                onClick={() => fileInputRef.current?.click()}
                onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
                onDragLeave={() => setDragOver(false)}
                onDrop={handleDrop}
              >
                <Upload className="img-dropzone-icon" />
                <p className="img-dropzone-title">点击选择图片或拖拽到此处</p>
                <p className="img-dropzone-hint">支持 JPG、PNG、GIF、WebP 格式</p>
              </div>
            ) : (
              <div className="img-preview-box">
                <img src={uploadPreview} alt="上传预览" />
                <button
                  className="img-preview-replace"
                  onClick={() => { setUploadPreview(''); fileInputRef.current?.click() }}
                  type="button"
                >
                  重新选择
                </button>
              </div>
            )}
          </>
        )}
      </Modal>
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
    <button
      className={`tiptap-tool-btn${active ? ' active' : ''}${disabled ? ' disabled' : ''}`}
      onClick={onClick}
      disabled={disabled}
      data-tooltip={tooltip}
      type="button"
    >
      <Icon className="h-4 w-4" />
    </button>
  )
}
