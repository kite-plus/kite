import { cn } from "@/lib/utils";

// Brand logos for storage backends.
// Each mark is drawn on a 32x32 viewBox so it scales cleanly at any size.
// These are stylized marks — evocative of each brand, not pixel-perfect trademarks.
type LogoVendor =
  | "aws"
  | "cloudflare"
  | "aliyun"
  | "tencent"
  | "huawei"
  | "baidu"
  | "gcp"
  | "azure"
  | "backblaze"
  | "minio"
  | "wasabi"
  | "do"
  | "scaleway"
  | "ftp"
  | "local";

const LogoMarks: Record<LogoVendor, (p: React.SVGProps<SVGSVGElement>) => React.ReactElement> = {
  // AWS — orange cubes mark
  aws: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <path d="M16 3 4 9v14l12 6 12-6V9L16 3Z" fill="#FF9900" />
      <path d="M16 3 4 9l12 6 12-6-12-6Z" fill="#FFB13C" />
      <path d="M16 15v14l12-6V9l-12 6Z" fill="#D97706" />
      <path d="M9 13.5v4l6 3v-4l-6-3Zm0 5.5v3l6 3v-3l-6-3Z" fill="#fff" opacity=".95" />
      <path d="M23 13.5v4l-6 3v-4l6-3Zm0 5.5v3l-6 3v-3l6-3Z" fill="#fff" opacity=".55" />
    </svg>
  ),

  // Cloudflare — signature orange cloud
  cloudflare: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#fff" />
      <path
        d="M25 18.5 22.2 13c-.4-1-1.4-1.7-2.5-1.7h-8c-1.1 0-2.1.7-2.5 1.7l-.6 1.4c-1.7.1-3 1.5-3 3.2 0 1.8 1.4 3.2 3.2 3.2h14.8c1.4 0 2.5-1.1 2.5-2.5-.1-.6-.4-1.2-.9-1.6"
        fill="#F48120"
      />
      <path
        d="M23.7 21c-1.4 0-2.6-1-2.9-2.4l-.8-3.4c-.4-1.6-1.9-2.7-3.6-2.7h-4.3c.4-.8 1.2-1.3 2.1-1.3h8c1.1 0 2.1.7 2.5 1.7l2.8 5.5c.5.4.8 1 .9 1.6 0 1.4-1.1 2.5-2.5 2.5h-1.9c.2-.6.3-1.1-.3-1.5"
        fill="#FAAD3F"
      />
    </svg>
  ),

  // Alibaba Cloud — orange "[]" brackets with the "a" dot
  aliyun: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#FF6A00" />
      <rect x="5" y="10" width="7" height="12" rx="2" fill="#fff" />
      <rect x="20" y="10" width="7" height="12" rx="2" fill="#fff" />
      <rect x="13" y="15" width="6" height="2" rx="1" fill="#fff" />
    </svg>
  ),

  // Tencent Cloud — blue ribbon
  tencent: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#00A4FF" />
      <path
        d="M22 13.8c0 .5-.3 1-.7 1.3l-7 4.7c-.4.3-1 .3-1.4 0l-2.3-1.5a.8.8 0 0 1 .9-1.3l1.9 1.3c.1.1.2.1.3 0l7-4.7c.3-.2.4-.7.2-1l-2.8-4.2c-.2-.2-.4-.4-.7-.4h-6.6c-.3 0-.5.2-.7.4L6.3 12.6c-.2.3-.1.7.2 1l1.3.8a.8.8 0 0 1-.8 1.3l-1.3-.8c-1-.7-1.3-2-.7-3l2.8-4.2c.4-.7 1.2-1.1 2-1.1h6.6c.8 0 1.6.4 2 1.1l2.8 4.2c.1.2.2.5.2.7"
        fill="#fff"
      />
      <path
        d="M9 19.5a.8.8 0 1 1 1.5-.5l.6 1.7a.2.2 0 0 0 .3.1l8.9-6a.8.8 0 0 1 .9 1.3l-8.9 6c-.7.5-1.6.2-1.9-.5L9 19.5Z"
        fill="#fff"
      />
    </svg>
  ),

  // Huawei Cloud — red diamond mark
  huawei: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#CF0A2C" />
      <path
        d="M16 6 8 12v8l8 6 8-6v-8L16 6Zm0 3.2 5.2 3.9v5.8L16 22.8l-5.2-3.9v-5.8L16 9.2Z"
        fill="#fff"
      />
      <path d="M16 13a3 3 0 1 0 0 6 3 3 0 0 0 0-6Z" fill="#fff" />
    </svg>
  ),

  // Baidu Cloud — blue cloud mark
  baidu: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#2932E1" />
      <path
        d="M9.5 13a4 4 0 0 0-1.5 7.7V21h16v-.3A4 4 0 0 0 22.5 13a5.5 5.5 0 0 0-10.5-.5A3.5 3.5 0 0 0 9.5 13Z"
        fill="#fff"
      />
    </svg>
  ),

  // Google Cloud — 4-color petals
  gcp: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#fff" />
      <path d="M16 6a10 10 0 0 1 8.7 5l-3.6 2.1A5.8 5.8 0 0 0 16 10.2V6Z" fill="#EA4335" />
      <path d="M16 6v4.2A5.8 5.8 0 0 0 10.2 16H6A10 10 0 0 1 16 6Z" fill="#4285F4" />
      <path d="M6 16h4.2a5.8 5.8 0 0 0 5.8 5.8V26A10 10 0 0 1 6 16Z" fill="#FBBC04" />
      <path d="M16 26v-4.2a5.8 5.8 0 0 0 5.1-3.1l3.6 2.1A10 10 0 0 1 16 26Z" fill="#34A853" />
    </svg>
  ),

  // Azure — blue triangle pair
  azure: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#fff" />
      <path d="M13.5 7h5L28 26h-9l-1.5-4h-5l4-3.5L13.5 7Z" fill="#0078D4" />
      <path d="m13.5 7-9 19h7.2l8.8-2.5L13.5 7Z" fill="#50E6FF" />
    </svg>
  ),

  // Backblaze — red "B"
  backblaze: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#E8372C" />
      <path
        d="M11 9h5c2.8 0 4.5 1.2 4.5 3.3 0 1.3-.7 2.2-1.9 2.7 1.6.4 2.6 1.5 2.6 3.1 0 2.2-1.8 3.7-4.7 3.7H11V9Zm2.6 2.1v3.6h2.3c1.5 0 2.3-.7 2.3-1.8s-.8-1.8-2.3-1.8h-2.3Zm0 5.5v3.8h2.7c1.6 0 2.5-.7 2.5-1.9s-.9-1.9-2.5-1.9h-2.7Z"
        fill="#fff"
      />
    </svg>
  ),

  // MinIO — crimson "M"
  minio: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#C72E49" />
      <path d="M7 23V9h3.5L16 19.5 21.5 9H25v14h-3V14l-4.5 9h-3L10 14v9H7Z" fill="#fff" />
    </svg>
  ),

  // Wasabi — green droplet
  wasabi: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#01CD3E" />
      <path
        d="M16 5c-3.2 3.6-6 7-6 10.5a6 6 0 1 0 12 0C22 12 19.2 8.6 16 5Z"
        fill="#fff"
      />
    </svg>
  ),

  // DigitalOcean Spaces — blue circles
  do: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#0080FF" />
      <circle cx="16" cy="16" r="7.5" fill="none" stroke="#fff" strokeWidth="2.5" />
      <rect x="6" y="22" width="4" height="4" rx="0.8" fill="#fff" />
    </svg>
  ),

  // Scaleway — purple brackets
  scaleway: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#4F0599" />
      <path
        d="M12 7v14a2 2 0 0 1-2 2H7v-2h3V7h2Zm8 18V11a2 2 0 0 1 2-2h3v2h-3v14h-2Z"
        fill="#fff"
      />
    </svg>
  ),

  // FTP — server rack
  ftp: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#52525B" />
      <rect x="7" y="9" width="18" height="5" rx="1.2" fill="#fff" />
      <rect x="7" y="18" width="18" height="5" rx="1.2" fill="#fff" />
      <circle cx="10" cy="11.5" r="0.8" fill="#22C55E" />
      <circle cx="10" cy="20.5" r="0.8" fill="#22C55E" />
    </svg>
  ),

  // Local disk — a tidy HDD
  local: (p) => (
    <svg viewBox="0 0 32 32" fill="none" {...p}>
      <rect width="32" height="32" rx="7" fill="#F4F4F5" />
      <rect x="5" y="11" width="22" height="10" rx="2" fill="#fff" stroke="#D4D4D8" />
      <circle cx="23" cy="16" r="1.3" fill="#22C55E" />
      <path d="M8 14h8v.8H8zM8 16.6h6v.8H8zM8 19h4v.8H8z" fill="#A1A1AA" />
    </svg>
  ),
};

// Vendor → display label (short, used in badges)
export const STORAGE_VENDOR_LABELS: Record<LogoVendor, string> = {
  aws: "AWS",
  cloudflare: "Cloudflare",
  aliyun: "Aliyun",
  tencent: "Tencent",
  huawei: "Huawei",
  baidu: "Baidu",
  gcp: "Google Cloud",
  azure: "Azure",
  backblaze: "Backblaze",
  minio: "MinIO",
  wasabi: "Wasabi",
  do: "DO Spaces",
  scaleway: "Scaleway",
  ftp: "FTP",
  local: "Local",
};

// Map the backend's `provider` / `driver` fields to a logo vendor key.
// Provider is the richer signal (e.g. "aws-s3", "cloudflare-r2"); driver is a
// fallback so a bare "s3" / "oss" / "cos" / "local" / "ftp" still renders.
export function resolveLogoVendor(provider: string | undefined, driver: string): LogoVendor {
  const p = (provider ?? "").toLowerCase();
  if (p.includes("aws") || p.includes("amazon")) return "aws";
  if (p.includes("cloudflare") || p.includes("r2")) return "cloudflare";
  if (p.includes("aliyun") || p.includes("alibaba") || p.includes("aliyuncs")) return "aliyun";
  if (p.includes("tencent") || p.includes("cos") || p.includes("qcloud")) return "tencent";
  if (p.includes("huawei") || p.includes("obs") || p.includes("myhuaweicloud")) return "huawei";
  if (p.includes("baidu") || p.includes("bos") || p.includes("bcebos")) return "baidu";
  if (p.includes("gcp") || p.includes("google") || p.includes("gcs")) return "gcp";
  if (p.includes("azure")) return "azure";
  if (p.includes("backblaze") || p.includes("b2")) return "backblaze";
  if (p.includes("minio")) return "minio";
  if (p.includes("wasabi")) return "wasabi";
  if (p.includes("do-") || p.includes("digitalocean") || p.includes("spaces")) return "do";
  if (p.includes("scaleway")) return "scaleway";

  switch (driver) {
    case "oss":
      return "aliyun";
    case "cos":
      return "tencent";
    case "obs":
      return "huawei";
    case "bos":
      return "baidu";
    case "ftp":
      return "ftp";
    case "local":
      return "local";
    case "s3":
      return "aws";
    default:
      return "local";
  }
}

interface StorageLogoProps {
  vendor: LogoVendor;
  size?: number;
  rounded?: string;
  className?: string;
}

// StorageLogo — renders a brand mark at the given size inside a neutral frame.
// Each mark carries its own brand background, so we just wrap it in a rounded
// container with a subtle inset border for consistency across vendors.
export function StorageLogo({
  vendor,
  size = 40,
  rounded = "rounded-lg",
  className,
}: StorageLogoProps) {
  const Logo = LogoMarks[vendor] ?? LogoMarks.local;
  return (
    <div
      className={cn(
        "relative flex shrink-0 items-center justify-center overflow-hidden",
        rounded,
        className,
      )}
      style={{
        width: size,
        height: size,
        boxShadow: "inset 0 0 0 1px rgba(0,0,0,0.06)",
      }}
    >
      <Logo width={size} height={size} />
    </div>
  );
}

export type { LogoVendor };
