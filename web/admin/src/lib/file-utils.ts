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

import { translate } from '@/i18n'

interface FileIconInfo {
  icon: LucideIcon
  color: string // Tailwind text color class
  bg: string // Tailwind bg color class
  // i18n key (e.g. "fileType.pdf"). Resolve via the t() returned from
  // useI18n() so the label re-renders when the user switches locale.
  // Components without a React context can still resolve it via the
  // standalone translate() helper.
  labelKey: string
}

// Extension → icon/color/labelKey mapping
const extMap: Record<string, FileIconInfo> = {
  // PDF
  pdf: {
    icon: FileText,
    color: 'text-red-600',
    bg: 'bg-red-50',
    labelKey: 'fileType.pdf',
  },
  // Word
  doc: {
    icon: FileText,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.word',
  },
  docx: {
    icon: FileText,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.word',
  },
  odt: {
    icon: FileText,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.odt',
  },
  rtf: {
    icon: FileText,
    color: 'text-blue-500',
    bg: 'bg-blue-50',
    labelKey: 'fileType.rtf',
  },
  pages: {
    icon: FileText,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.pages',
  },
  // Excel / Spreadsheet
  xls: {
    icon: FileSpreadsheet,
    color: 'text-green-700',
    bg: 'bg-green-50',
    labelKey: 'fileType.excel',
  },
  xlsx: {
    icon: FileSpreadsheet,
    color: 'text-green-700',
    bg: 'bg-green-50',
    labelKey: 'fileType.excel',
  },
  ods: {
    icon: FileSpreadsheet,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.ods',
  },
  numbers: {
    icon: FileSpreadsheet,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.numbers',
  },
  csv: {
    icon: FileSpreadsheet,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.csv',
  },
  // PPT / Presentation
  ppt: {
    icon: FileImage,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.ppt',
  },
  pptx: {
    icon: FileImage,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.ppt',
  },
  odp: {
    icon: FileImage,
    color: 'text-orange-500',
    bg: 'bg-orange-50',
    labelKey: 'fileType.odp',
  },
  key: {
    icon: FileImage,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.keynote',
  },
  // Plain text & config
  txt: {
    icon: FileText,
    color: 'text-gray-600',
    bg: 'bg-gray-100',
    labelKey: 'fileType.plainText',
  },
  log: {
    icon: FileText,
    color: 'text-gray-500',
    bg: 'bg-gray-100',
    labelKey: 'fileType.log',
  },
  ini: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.iniConfig',
  },
  cfg: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.config',
  },
  conf: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.config',
  },
  env: {
    icon: FileText,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    labelKey: 'fileType.envFile',
  },
  properties: {
    icon: FileText,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.propertiesFile',
  },
  // Markdown & data formats
  md: {
    icon: FileText,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    labelKey: 'fileType.markdown',
  },
  mdx: {
    icon: FileText,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    labelKey: 'fileType.mdx',
  },
  yaml: {
    icon: FileCode,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    labelKey: 'fileType.yaml',
  },
  yml: {
    icon: FileCode,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    labelKey: 'fileType.yaml',
  },
  json: {
    icon: FileJson,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.json',
  },
  jsonc: {
    icon: FileJson,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.jsonc',
  },
  json5: {
    icon: FileJson,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    labelKey: 'fileType.json5',
  },
  xml: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.xml',
  },
  xsl: {
    icon: FileCode,
    color: 'text-orange-500',
    bg: 'bg-orange-50',
    labelKey: 'fileType.xsl',
  },
  toml: {
    icon: FileCode,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    labelKey: 'fileType.toml',
  },
  svg: {
    icon: FileImage,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    labelKey: 'fileType.svg',
  },
  graphql: {
    icon: FileCode,
    color: 'text-pink-700',
    bg: 'bg-pink-50',
    labelKey: 'fileType.graphql',
  },
  proto: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.protobuf',
  },
  // Web frontend
  html: {
    icon: Globe,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    labelKey: 'fileType.html',
  },
  htm: {
    icon: Globe,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    labelKey: 'fileType.html',
  },
  css: {
    icon: FileCode,
    color: 'text-sky-600',
    bg: 'bg-sky-50',
    labelKey: 'fileType.css',
  },
  scss: {
    icon: FileCode,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    labelKey: 'fileType.scss',
  },
  sass: {
    icon: FileCode,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    labelKey: 'fileType.sass',
  },
  less: {
    icon: FileCode,
    color: 'text-indigo-500',
    bg: 'bg-indigo-50',
    labelKey: 'fileType.less',
  },
  // JavaScript family
  js: {
    icon: FileCode,
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    labelKey: 'fileType.javascript',
  },
  jsx: {
    icon: FileCode,
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    labelKey: 'fileType.jsx',
  },
  mjs: {
    icon: FileCode,
    color: 'text-yellow-600',
    bg: 'bg-yellow-50',
    labelKey: 'fileType.javascript',
  },
  cjs: {
    icon: FileCode,
    color: 'text-yellow-600',
    bg: 'bg-yellow-50',
    labelKey: 'fileType.javascript',
  },
  ts: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.typescript',
  },
  tsx: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.tsx',
  },
  vue: {
    icon: FileCode,
    color: 'text-emerald-600',
    bg: 'bg-emerald-50',
    labelKey: 'fileType.vue',
  },
  svelte: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.svelte',
  },
  // Backend languages
  py: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.python',
  },
  go: {
    icon: FileCode,
    color: 'text-cyan-700',
    bg: 'bg-cyan-50',
    labelKey: 'fileType.go',
  },
  rs: {
    icon: FileCode,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    labelKey: 'fileType.rust',
  },
  java: {
    icon: FileCode,
    color: 'text-red-600',
    bg: 'bg-red-50',
    labelKey: 'fileType.java',
  },
  kt: {
    icon: FileCode,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    labelKey: 'fileType.kotlin',
  },
  scala: {
    icon: FileCode,
    color: 'text-red-500',
    bg: 'bg-red-50',
    labelKey: 'fileType.scala',
  },
  c: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.cLang',
  },
  cpp: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.cpp',
  },
  cc: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.cpp',
  },
  h: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.cHeader',
  },
  hpp: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.cppHeader',
  },
  cs: {
    icon: FileCode,
    color: 'text-purple-700',
    bg: 'bg-purple-50',
    labelKey: 'fileType.csharp',
  },
  rb: {
    icon: FileCode,
    color: 'text-red-500',
    bg: 'bg-red-50',
    labelKey: 'fileType.ruby',
  },
  php: {
    icon: FileCode,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    labelKey: 'fileType.php',
  },
  swift: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.swift',
  },
  dart: {
    icon: FileCode,
    color: 'text-cyan-600',
    bg: 'bg-cyan-50',
    labelKey: 'fileType.dart',
  },
  lua: {
    icon: FileCode,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.lua',
  },
  r: {
    icon: FileCode,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.rLang',
  },
  zig: {
    icon: FileCode,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    labelKey: 'fileType.zig',
  },
  nim: {
    icon: FileCode,
    color: 'text-yellow-700',
    bg: 'bg-yellow-50',
    labelKey: 'fileType.nim',
  },
  hs: {
    icon: FileCode,
    color: 'text-purple-700',
    bg: 'bg-purple-50',
    labelKey: 'fileType.haskell',
  },
  ml: {
    icon: FileCode,
    color: 'text-orange-600',
    bg: 'bg-orange-50',
    labelKey: 'fileType.ocaml',
  },
  clj: {
    icon: FileCode,
    color: 'text-green-700',
    bg: 'bg-green-50',
    labelKey: 'fileType.clojure',
  },
  wasm: {
    icon: FileCode,
    color: 'text-violet-700',
    bg: 'bg-violet-50',
    labelKey: 'fileType.webassembly',
  },
  // Shell & script
  sh: {
    icon: Terminal,
    color: 'text-emerald-700',
    bg: 'bg-emerald-50',
    labelKey: 'fileType.shell',
  },
  bash: {
    icon: Terminal,
    color: 'text-emerald-700',
    bg: 'bg-emerald-50',
    labelKey: 'fileType.bash',
  },
  zsh: {
    icon: Terminal,
    color: 'text-emerald-700',
    bg: 'bg-emerald-50',
    labelKey: 'fileType.zsh',
  },
  fish: {
    icon: Terminal,
    color: 'text-emerald-600',
    bg: 'bg-emerald-50',
    labelKey: 'fileType.fish',
  },
  ps1: {
    icon: Terminal,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.powershell',
  },
  bat: {
    icon: Terminal,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    labelKey: 'fileType.batch',
  },
  cmd: {
    icon: Terminal,
    color: 'text-gray-700',
    bg: 'bg-gray-100',
    labelKey: 'fileType.batch',
  },
  // Database & SQL
  sql: {
    icon: Database,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.sql',
  },
  db: {
    icon: Database,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.database',
  },
  sqlite: {
    icon: Database,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.sqlite',
  },
  sqlite3: {
    icon: Database,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.sqlite',
  },
  // Archives
  zip: {
    icon: FileArchive,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    labelKey: 'fileType.zipArchive',
  },
  rar: {
    icon: FileArchive,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    labelKey: 'fileType.rarArchive',
  },
  '7z': {
    icon: FileArchive,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    labelKey: 'fileType.sevenZArchive',
  },
  tar: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.tarArchive',
  },
  gz: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.gzArchive',
  },
  tgz: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.tgzArchive',
  },
  bz2: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.bz2Archive',
  },
  xz: {
    icon: FileArchive,
    color: 'text-amber-700',
    bg: 'bg-amber-50',
    labelKey: 'fileType.xzArchive',
  },
  // Fonts
  ttf: {
    icon: FileType,
    color: 'text-fuchsia-600',
    bg: 'bg-fuchsia-50',
    labelKey: 'fileType.truetypeFont',
  },
  otf: {
    icon: FileType,
    color: 'text-fuchsia-600',
    bg: 'bg-fuchsia-50',
    labelKey: 'fileType.opentypeFont',
  },
  woff: {
    icon: FileType,
    color: 'text-fuchsia-500',
    bg: 'bg-fuchsia-50',
    labelKey: 'fileType.woffFont',
  },
  woff2: {
    icon: FileType,
    color: 'text-fuchsia-500',
    bg: 'bg-fuchsia-50',
    labelKey: 'fileType.woff2Font',
  },
  // Executables / installers
  exe: {
    icon: Cpu,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    labelKey: 'fileType.executable',
  },
  msi: {
    icon: Cpu,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    labelKey: 'fileType.installer',
  },
  dmg: {
    icon: Cpu,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    labelKey: 'fileType.dmgImage',
  },
  app: {
    icon: Cpu,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.appBundle',
  },
  deb: {
    icon: Cpu,
    color: 'text-red-500',
    bg: 'bg-red-50',
    labelKey: 'fileType.debInstaller',
  },
  rpm: {
    icon: Cpu,
    color: 'text-red-500',
    bg: 'bg-red-50',
    labelKey: 'fileType.rpmInstaller',
  },
  apk: {
    icon: Cpu,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.apkInstaller',
  },
  ipa: {
    icon: Cpu,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.ipaInstaller',
  },
  jar: {
    icon: Cpu,
    color: 'text-red-500',
    bg: 'bg-red-50',
    labelKey: 'fileType.jarPackage',
  },
  war: {
    icon: Cpu,
    color: 'text-red-500',
    bg: 'bg-red-50',
    labelKey: 'fileType.warPackage',
  },
  // Design
  psd: {
    icon: Palette,
    color: 'text-blue-700',
    bg: 'bg-blue-50',
    labelKey: 'fileType.photoshop',
  },
  ai: {
    icon: Palette,
    color: 'text-orange-700',
    bg: 'bg-orange-50',
    labelKey: 'fileType.illustrator',
  },
  sketch: {
    icon: Palette,
    color: 'text-amber-600',
    bg: 'bg-amber-50',
    labelKey: 'fileType.sketch',
  },
  fig: {
    icon: Palette,
    color: 'text-purple-600',
    bg: 'bg-purple-50',
    labelKey: 'fileType.figma',
  },
  xd: {
    icon: Palette,
    color: 'text-pink-600',
    bg: 'bg-pink-50',
    labelKey: 'fileType.adobeXD',
  },
  // eBooks
  epub: {
    icon: BookOpen,
    color: 'text-teal-600',
    bg: 'bg-teal-50',
    labelKey: 'fileType.epub',
  },
  mobi: {
    icon: BookOpen,
    color: 'text-teal-600',
    bg: 'bg-teal-50',
    labelKey: 'fileType.mobi',
  },
  azw: {
    icon: BookOpen,
    color: 'text-teal-700',
    bg: 'bg-teal-50',
    labelKey: 'fileType.kindle',
  },
  azw3: {
    icon: BookOpen,
    color: 'text-teal-700',
    bg: 'bg-teal-50',
    labelKey: 'fileType.kindle',
  },
  // Disk images
  iso: {
    icon: Disc,
    color: 'text-slate-700',
    bg: 'bg-slate-100',
    labelKey: 'fileType.iso',
  },
  img: {
    icon: Disc,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.diskImage',
  },
  vmdk: {
    icon: Disc,
    color: 'text-slate-600',
    bg: 'bg-slate-100',
    labelKey: 'fileType.vmdk',
  },
  // Certificates
  pem: {
    icon: FileKey,
    color: 'text-green-700',
    bg: 'bg-green-50',
    labelKey: 'fileType.pemCert',
  },
  crt: {
    icon: FileKey,
    color: 'text-green-700',
    bg: 'bg-green-50',
    labelKey: 'fileType.certificate',
  },
  cer: {
    icon: FileKey,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.certificate',
  },
  p12: {
    icon: FileKey,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.pkcs12',
  },
  pfx: {
    icon: FileKey,
    color: 'text-green-600',
    bg: 'bg-green-50',
    labelKey: 'fileType.pfx',
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
    labelKey: 'fileType.image',
  },
  video: {
    icon: Video,
    color: 'text-violet-600',
    bg: 'bg-violet-50',
    labelKey: 'fileType.video',
  },
  audio: {
    icon: Music,
    color: 'text-blue-600',
    bg: 'bg-blue-50',
    labelKey: 'fileType.audio',
  },
}

const defaultInfo: FileIconInfo = {
  icon: FileText,
  color: 'text-gray-500',
  bg: 'bg-gray-100',
  labelKey: 'fileType.file',
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

/**
 * Resolve a friendly type label. Pass `t` from `useI18n()` to keep the
 * label reactive to locale changes; omit it in non-React contexts and
 * the standalone `translate()` helper will read the active locale from
 * localStorage.
 */
export function getFileTypeLabel(
  file: {
    file_type?: string
    original_name?: string
    mime_type?: string
  },
  t?: (key: string) => string
): string {
  const key = getFileIconInfo(file).labelKey
  return t ? t(key) : translate(key)
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
