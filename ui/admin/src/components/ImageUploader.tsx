import { useState, useRef } from 'react'
import { Button, Spin } from '@douyinfe/semi-ui'
import { IconUpload, IconDelete } from '@douyinfe/semi-icons'
import { apiUpload } from '@/lib/api-client'

interface ImageUploaderProps {
  value?: string
  onChange?: (url: string) => void
  placeholder?: string
}

/**
 * 图片上传组件 — 支持上传和 URL 输入
 */
export function ImageUploader({ value, onChange, placeholder = '上传或粘贴图片 URL' }: ImageUploaderProps) {
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleFile(file: File) {
    if (!file.type.startsWith('image/')) return
    setUploading(true)
    try {
      const result = await apiUpload(file)
      onChange?.(result.url)
    } catch (e) {
      console.error('上传失败', e)
    } finally {
      setUploading(false)
    }
  }

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragOver(false)
    const file = e.dataTransfer.files[0]
    if (file) handleFile(file)
  }

  return (
    <div>
      {value ? (
        <div style={{ position: 'relative' }}>
          <img
            src={value}
            alt="封面预览"
            style={{ width: '100%', borderRadius: 6, border: '1px solid var(--semi-color-border)' }}
          />
          <div style={{ display: 'flex', gap: 6, marginTop: 8 }}>
            <Button size="small" icon={<IconUpload />} onClick={() => fileInputRef.current?.click()} disabled={uploading}>
              {uploading ? '上传中…' : '更换'}
            </Button>
            <Button size="small" type="danger" icon={<IconDelete />} onClick={() => onChange?.('')}>
              移除
            </Button>
          </div>
        </div>
      ) : (
        <div
          style={{
            border: `2px dashed ${dragOver ? 'var(--semi-color-primary)' : 'var(--semi-color-border)'}`,
            borderRadius: 8,
            padding: '20px 12px',
            textAlign: 'center',
            cursor: 'pointer',
            transition: 'border-color 0.2s',
            background: dragOver ? 'var(--semi-color-fill-0)' : 'transparent',
          }}
          onClick={() => fileInputRef.current?.click()}
          onDragOver={(e) => { e.preventDefault(); setDragOver(true) }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
        >
          {uploading ? (
            <Spin size="middle" />
          ) : (
            <>
              <IconUpload size="large" style={{ color: 'var(--semi-color-text-2)', marginBottom: 4 }} />
              <div style={{ fontSize: 13, color: 'var(--semi-color-text-2)' }}>{placeholder}</div>
              <div style={{ fontSize: 11, color: 'var(--semi-color-text-3)', marginTop: 2 }}>
                支持 JPG / PNG / WebP / GIF
              </div>
            </>
          )}
        </div>
      )}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        style={{ display: 'none' }}
        onChange={(e) => { const f = e.target.files?.[0]; if (f) handleFile(f); e.target.value = '' }}
      />
    </div>
  )
}
