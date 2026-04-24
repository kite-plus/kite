import {
  FileText,
  FileCode,
  FileJson,
  FileType,
  FileArchive,
  FileSpreadsheet,
  FileImage,
  Music,
  Image,
  Video,
  FileKey,
  Database,
  Terminal,
  Globe,
  Cpu,
  BookOpen,
  Palette,
  Disc,
  type LucideIcon,
} from 'lucide-react'

interface FileIconInfo {
  icon: LucideIcon
  color: string // Tailwind text color class
  bg: string // Tailwind bg color class
  label: string // Friendly type name in Chinese
}

// Extension → icon/color/label mapping
const extMap: Record<string, FileIconInfo> = {
  // PDF
  pdf: {
    icon: FileText,
    color: 'text-red-600',
    bg: 'bg-red-50',
    label: 'PDF 文档',
  },
  // Word
  doc: {
    icon: FileText,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'Word 文档',
  },
  docx: {
    icon: FileText,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'Word 文档',
  },
  odt: {
    icon: FileText,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'ODT 文档',
  },
  rtf: {
    icon: FileText,
    color: 'text-blue-500',
    bg: 'bg-blue-50',
    label: 'RTF 文档',
  },
  pages: {
    icon: FileText,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'Pages 文档',
  },
  // Excel / Spreadsheet
  xls: {
    icon: FileSpreadsheet,
    color: 'text-green-700',
    bg: 'bg-green-50',
    label: 'Excel 表格',
  },
  xlsx: {
    icon: FileSpreadsheet,
    color: 'text-green-700',
    bg: 'bg-green-50',
    label: 'Excel 表格',
  },
  ods: {
    icon: FileSpreadsheet,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: 'ODS 表格',
  },
  numbers: {
    icon: FileSpreadsheet,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: 'Numbers 表格',
  },
  csv: {
    icon: FileSpreadsheet,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: 'CSV 表格',
  },
  // PPT / Presentation
  ppt: {
    icon: FileImage,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'PPT 演示',
  },
  pptx: {
    icon: FileImage,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'PPT 演示',
  },
  odp: {
    icon: FileImage,
    color: 'text-orange-500',
    bg: 'bg-orange-50',
    label: 'ODP 演示',
  },
  key: {
    icon: FileImage,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'Keynote 演示',
  },
  // Plain text & config
  txt: {
    icon: FileText,
    color: 'text-gray-600',
    bg: 'bg-gray-100',
    label: '纯文本',
  },
  log: {
    icon: FileText,
    color: 'text-gray-500',
    bg: 'bg-gray-100',
    label: '日志文件',
  },
  ini: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: 'INI 配置',
  },
  cfg: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: '配置文件',
  },
  conf: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: '配置文件',
  },
  env: {
    icon: FileText,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    label: '环境变量',
  },
  properties: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: '属性文件',
  },
  // Markdown & data formats
  md: {
    icon: FileText,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    label: 'Markdown',
  },
  mdx: {
    icon: FileText,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    label: 'MDX',
  },
  yaml: {
    icon: FileCode,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    label: 'YAML',
  },
  yml: {
    icon: FileCode,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    label: 'YAML',
  },
  json: {
    icon: FileJson,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'JSON',
  },
  jsonc: {
    icon: FileJson,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'JSONC',
  },
  json5: {
    icon: FileJson,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    label: 'JSON5',
  },
  xml: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'XML',
  },
  xsl: {
    icon: FileCode,
    color: 'text-orange-500',
    bg: 'bg-orange-50',
    label: 'XSL',
  },
  toml: {
    icon: FileCode,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    label: 'TOML',
  },
  svg: {
    icon: FileImage,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    label: 'SVG',
  },
  graphql: {
    icon: FileCode,
    color: 'text-pink-700',
    bg: 'bg-pink-50',
    label: 'GraphQL',
  },
  proto: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'Protobuf',
  },
  // Web frontend
  html: {
    icon: Globe,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    label: 'HTML',
  },
  htm: {
    icon: Globe,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    label: 'HTML',
  },
  css: { icon: FileCode, color: 'text-sky-600', bg: 'bg-sky-50', label: 'CSS' },
  scss: {
    icon: FileCode,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    label: 'SCSS',
  },
  sass: {
    icon: FileCode,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    label: 'Sass',
  },
  less: {
    icon: FileCode,
    color: 'text-indigo-500',
    bg: 'bg-indigo-50',
    label: 'Less',
  },
  // JavaScript family
  js: {
    icon: FileCode,
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    label: 'JavaScript',
  },
  jsx: {
    icon: FileCode,
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    label: 'JSX',
  },
  mjs: {
    icon: FileCode,
    color: 'text-yellow-600',
    bg: 'bg-yellow-50',
    label: 'JavaScript',
  },
  cjs: {
    icon: FileCode,
    color: 'text-yellow-600',
    bg: 'bg-yellow-50',
    label: 'JavaScript',
  },
  ts: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'TypeScript',
  },
  tsx: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'TSX',
  },
  vue: {
    icon: FileCode,
    color: 'text-emerald-600',
    bg: 'bg-emerald-50',
    label: 'Vue',
  },
  svelte: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'Svelte',
  },
  // Backend languages
  py: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'Python',
  },
  go: { icon: FileCode, color: 'text-cyan-700', bg: 'bg-cyan-50', label: 'Go' },
  rs: {
    icon: FileCode,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    label: 'Rust',
  },
  java: {
    icon: FileCode,
    color: 'text-red-600',
    bg: 'bg-red-50',
    label: 'Java',
  },
  kt: {
    icon: FileCode,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    label: 'Kotlin',
  },
  scala: {
    icon: FileCode,
    color: 'text-red-500',
    bg: 'bg-red-50',
    label: 'Scala',
  },
  c: { icon: FileCode, color: 'text-blue-700', bg: 'bg-blue-50', label: 'C' },
  cpp: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'C++',
  },
  cc: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'C++',
  },
  h: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'C 头文件',
  },
  hpp: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'C++ 头文件',
  },
  cs: {
    icon: FileCode,
    color: 'text-purple-700',
    bg: 'bg-purple-50',
    label: 'C#',
  },
  rb: { icon: FileCode, color: 'text-red-500', bg: 'bg-red-50', label: 'Ruby' },
  php: {
    icon: FileCode,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    label: 'PHP',
  },
  swift: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'Swift',
  },
  dart: {
    icon: FileCode,
    color: 'text-cyan-600',
    bg: 'bg-cyan-50',
    label: 'Dart',
  },
  lua: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'Lua',
  },
  r: { icon: FileCode, color: 'text-blue-600', bg: 'bg-blue-50', label: 'R' },
  zig: {
    icon: FileCode,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    label: 'Zig',
  },
  nim: {
    icon: FileCode,
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    label: 'Nim',
  },
  hs: {
    icon: FileCode,
    color: 'text-purple-700',
    bg: 'bg-purple-50',
    label: 'Haskell',
  },
  ml: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    label: 'OCaml',
  },
  clj: {
    icon: FileCode,
    color: 'text-green-700',
    bg: 'bg-green-50',
    label: 'Clojure',
  },
  wasm: {
    icon: FileCode,
    color: 'text-violet-700',
    bg: 'bg-violet-50',
    label: 'WebAssembly',
  },
  // Shell & script
  sh: {
    icon: Terminal,
    color: 'text-emerald-700',
    bg: 'bg-emerald-50',
    label: 'Shell',
  },
  bash: {
    icon: Terminal,
    color: 'text-emerald-700',
    bg: 'bg-emerald-50',
    label: 'Bash',
  },
  zsh: {
    icon: Terminal,
    color: 'text-emerald-700',
    bg: 'bg-emerald-50',
    label: 'Zsh',
  },
  fish: {
    icon: Terminal,
    color: 'text-emerald-600',
    bg: 'bg-emerald-50',
    label: 'Fish',
  },
  ps1: {
    icon: Terminal,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'PowerShell',
  },
  bat: {
    icon: Terminal,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    label: '批处理',
  },
  cmd: {
    icon: Terminal,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    label: '批处理',
  },
  // Database & SQL
  sql: {
    icon: Database,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'SQL',
  },
  db: {
    icon: Database,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: '数据库',
  },
  sqlite: {
    icon: Database,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'SQLite',
  },
  sqlite3: {
    icon: Database,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'SQLite',
  },
  // Archives
  zip: {
    icon: FileArchive,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    label: 'ZIP 压缩包',
  },
  rar: {
    icon: FileArchive,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    label: 'RAR 压缩包',
  },
  '7z': {
    icon: FileArchive,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    label: '7z 压缩包',
  },
  tar: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'TAR 归档',
  },
  gz: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'GZ 压缩包',
  },
  tgz: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'TGZ 压缩包',
  },
  bz2: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'BZ2 压缩包',
  },
  xz: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    label: 'XZ 压缩包',
  },
  // Fonts
  ttf: {
    icon: FileType,
    color: 'text-fuchsia-600',
    bg: 'bg-fuchsia-50',
    label: 'TrueType 字体',
  },
  otf: {
    icon: FileType,
    color: 'text-fuchsia-600',
    bg: 'bg-fuchsia-50',
    label: 'OpenType 字体',
  },
  woff: {
    icon: FileType,
    color: 'text-fuchsia-500',
    bg: 'bg-fuchsia-50',
    label: 'WOFF 字体',
  },
  woff2: {
    icon: FileType,
    color: 'text-fuchsia-500',
    bg: 'bg-fuchsia-50',
    label: 'WOFF2 字体',
  },
  // Executables / installers
  exe: {
    icon: Cpu,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    label: '可执行文件',
  },
  msi: {
    icon: Cpu,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    label: '安装包',
  },
  dmg: {
    icon: Cpu,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    label: 'DMG 镜像',
  },
  app: {
    icon: Cpu,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: '应用程序',
  },
  deb: {
    icon: Cpu,
    color: 'text-red-500',
    bg: 'bg-red-50',
    label: 'DEB 安装包',
  },
  rpm: {
    icon: Cpu,
    color: 'text-red-500',
    bg: 'bg-red-50',
    label: 'RPM 安装包',
  },
  apk: {
    icon: Cpu,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: 'APK 安装包',
  },
  ipa: {
    icon: Cpu,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: 'IPA 安装包',
  },
  jar: { icon: Cpu, color: 'text-red-500', bg: 'bg-red-50', label: 'JAR 包' },
  war: { icon: Cpu, color: 'text-red-500', bg: 'bg-red-50', label: 'WAR 包' },
  // Design
  psd: {
    icon: Palette,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    label: 'Photoshop',
  },
  ai: {
    icon: Palette,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    label: 'Illustrator',
  },
  sketch: {
    icon: Palette,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    label: 'Sketch',
  },
  fig: {
    icon: Palette,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    label: 'Figma',
  },
  xd: {
    icon: Palette,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    label: 'Adobe XD',
  },
  // eBooks
  epub: {
    icon: BookOpen,
    color: 'text-teal-600',
    bg: 'bg-teal-50',
    label: 'EPUB 电子书',
  },
  mobi: {
    icon: BookOpen,
    color: 'text-teal-600',
    bg: 'bg-teal-50',
    label: 'Mobi 电子书',
  },
  azw: {
    icon: BookOpen,
    color: 'text-teal-700',
    bg: 'bg-teal-50',
    label: 'Kindle 电子书',
  },
  azw3: {
    icon: BookOpen,
    color: 'text-teal-700',
    bg: 'bg-teal-50',
    label: 'Kindle 电子书',
  },
  // Disk images
  iso: {
    icon: Disc,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    label: 'ISO 镜像',
  },
  img: {
    icon: Disc,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: '磁盘映像',
  },
  vmdk: {
    icon: Disc,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    label: 'VMDK 虚拟盘',
  },
  // Certificates
  pem: {
    icon: FileKey,
    color: 'text-green-700',
    bg: 'bg-green-50',
    label: 'PEM 证书',
  },
  crt: {
    icon: FileKey,
    color: 'text-green-700',
    bg: 'bg-green-50',
    label: '证书文件',
  },
  cer: {
    icon: FileKey,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: '证书文件',
  },
  p12: {
    icon: FileKey,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: 'PKCS12 证书',
  },
  pfx: {
    icon: FileKey,
    color: 'text-green-600',
    bg: 'bg-green-50',
    label: 'PFX 证书',
  },
}

// MIME → ext fallback
const mimeExtMap: Record<string, string> = {
  'application/pdf': 'pdf',
  'application/msword': 'doc',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document':
    'docx',
  'application/vnd.ms-excel': 'xls',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet': 'xlsx',
  'application/vnd.ms-powerpoint': 'ppt',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation':
    'pptx',
  'text/csv': 'csv',
  'text/markdown': 'md',
  'text/html': 'html',
  'text/css': 'css',
  'application/json': 'json',
  'application/javascript': 'js',
  'text/javascript': 'js',
  'application/typescript': 'ts',
  'application/xml': 'xml',
  'text/xml': 'xml',
  'text/yaml': 'yaml',
  'application/x-yaml': 'yaml',
  'application/zip': 'zip',
  'application/x-rar-compressed': 'rar',
  'application/gzip': 'gz',
  'application/x-7z-compressed': '7z',
  'application/x-tar': 'tar',
  'application/epub+zip': 'epub',
  'application/java-archive': 'jar',
  'font/ttf': 'ttf',
  'font/otf': 'otf',
  'font/woff': 'woff',
  'font/woff2': 'woff2',
}

// Media type defaults (for image/video/audio file_type)
const mediaDefaults: Record<string, FileIconInfo> = {
  image: {
    icon: Image,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    label: '图片',
  },
  video: {
    icon: Video,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    label: '视频',
  },
  audio: {
    icon: Music,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    label: '音频',
  },
}

const defaultInfo: FileIconInfo = {
  icon: FileText,
  color: 'text-gray-500',
  bg: 'bg-gray-100',
  label: '文件',
}

/** Get file icon info based on file_type, original_name, and mime_type */
export function getFileIconInfo(file: {
  file_type?: string
  original_name?: string
  mime_type?: string
}): FileIconInfo {
  // For images/videos/audio, check extension first, fallback to media default
  const name = (file.original_name ?? '').toLowerCase()
  const dotIdx = name.lastIndexOf('.')
  const ext = dotIdx > 0 ? name.substring(dotIdx + 1) : ''

  // Try extension match first
  if (ext && extMap[ext]) return extMap[ext]

  // For media types, return media defaults
  if (file.file_type && mediaDefaults[file.file_type]) {
    return mediaDefaults[file.file_type]
  }

  // Try MIME → ext fallback
  const mime = (file.mime_type ?? '').split(';')[0].trim().toLowerCase()
  const mimeExt = mimeExtMap[mime]
  if (mimeExt && extMap[mimeExt]) return extMap[mimeExt]

  // Fuzzy MIME matching
  if (/pdf/.test(mime)) return extMap['pdf']
  if (/wordprocessingml|msword|opendocument\.text/.test(mime))
    return extMap['docx']
  if (/spreadsheet|excel/.test(mime)) return extMap['xlsx']
  if (/presentation|powerpoint/.test(mime)) return extMap['pptx']
  if (/zip|rar|7z|tar|gzip|compress|bzip/.test(mime)) return extMap['zip']
  if (/font/.test(mime)) return extMap['ttf']
  if (/json/.test(mime)) return extMap['json']
  if (/xml/.test(mime)) return extMap['xml']
  if (/yaml/.test(mime)) return extMap['yaml']
  if (/html/.test(mime)) return extMap['html']
  if (/css/.test(mime)) return extMap['css']
  if (/javascript/.test(mime)) return extMap['js']
  if (/python/.test(mime)) return extMap['py']
  if (mime.startsWith('text/')) return extMap['txt']
  if (/octet-stream|executable/.test(mime)) return extMap['exe']

  return defaultInfo
}

/** Get friendly type label */
export function getFileTypeLabel(file: {
  file_type?: string
  original_name?: string
  mime_type?: string
}): string {
  return getFileIconInfo(file).label
}

// Image formats the browser will actually render via <img>. Anything
// else under file_type="image" (PSD, AI, TIFF, HEIC, RAW…) will just
// render as a broken-image glyph, so the UI should fall back to the
// file-type icon instead of blindly mounting the <img>.
const BROWSER_IMAGE_EXTS = new Set([
  'jpg',
  'jpeg',
  'png',
  'gif',
  'webp',
  'avif',
  'bmp',
  'ico',
  'svg',
  'apng',
  'jfif',
  'pjpeg',
  'pjp',
])
const BROWSER_IMAGE_MIMES = new Set([
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'image/avif',
  'image/bmp',
  'image/x-icon',
  'image/vnd.microsoft.icon',
  'image/svg+xml',
  'image/apng',
])

export function isImagePreviewable(file: {
  file_type?: string
  original_name?: string
  mime_type?: string
}): boolean {
  if (file.file_type !== 'image') return false
  const mime = (file.mime_type ?? '').split(';')[0].trim().toLowerCase()
  if (mime && BROWSER_IMAGE_MIMES.has(mime)) return true
  const name = (file.original_name ?? '').toLowerCase()
  const dotIdx = name.lastIndexOf('.')
  const ext = dotIdx > 0 ? name.substring(dotIdx + 1) : ''
  return ext ? BROWSER_IMAGE_EXTS.has(ext) : false
}
