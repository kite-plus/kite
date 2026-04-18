package storage

import (
	"encoding/json"
	"strings"
)

// DetectProvider 返回一个稳定的厂商标识，前端据此渲染品牌 logo。
// driver 为本地/FTP 时直接返回 driver；S3 兼容驱动则解析 endpoint 推断。
func DetectProvider(driver string, rawConfig string) string {
	switch driver {
	case DriverLocal, DriverFTP:
		return driver
	case DriverOSS:
		return "aliyun-oss"
	case DriverCOS:
		return "tencent-cos"
	case DriverS3:
		return detectS3Provider(rawConfig)
	}
	return driver
}

func detectS3Provider(rawConfig string) string {
	if rawConfig == "" {
		return "s3"
	}
	var c struct {
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal([]byte(rawConfig), &c); err != nil {
		return "s3"
	}
	ep := strings.ToLower(c.Endpoint)
	switch {
	case strings.Contains(ep, "r2.cloudflarestorage.com"):
		return "cloudflare-r2"
	case strings.Contains(ep, "amazonaws.com"):
		return "aws-s3"
	case strings.Contains(ep, "backblazeb2.com"):
		return "backblaze-b2"
	case strings.Contains(ep, "wasabisys.com"):
		return "wasabi"
	case strings.Contains(ep, "digitaloceanspaces.com"):
		return "do-spaces"
	case strings.Contains(ep, "scw.cloud"):
		return "scaleway"
	case strings.Contains(ep, "aliyuncs.com"):
		return "aliyun-oss"
	case strings.Contains(ep, "myqcloud.com"):
		return "tencent-cos"
	case strings.Contains(ep, "min.io") || strings.Contains(ep, "minio"):
		return "minio"
	}
	return "s3"
}
