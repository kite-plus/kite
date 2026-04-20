package storage

import (
	"encoding/json"
	"strings"
)

// DetectProvider returns a stable vendor identifier that the frontend uses to render a brand logo.
// For local/FTP drivers it returns the driver name directly; for S3-compatible drivers it infers the
// provider from the endpoint.
func DetectProvider(driver string, rawConfig string) string {
	switch driver {
	case DriverLocal, DriverFTP:
		return driver
	case DriverOSS:
		return "aliyun-oss"
	case DriverCOS:
		return "tencent-cos"
	case DriverOBS:
		return "huawei-obs"
	case DriverBOS:
		return "baidu-bos"
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
	case strings.Contains(ep, "myhuaweicloud.com"):
		return "huawei-obs"
	case strings.Contains(ep, "bcebos.com"):
		return "baidu-bos"
	case strings.Contains(ep, "googleapis.com"):
		return "google-gcs"
	case strings.Contains(ep, "ufileos.com") || strings.Contains(ep, "us3.ucloud"):
		return "ucloud-us3"
	case strings.Contains(ep, "jdcloud-oss.com"):
		return "jdcloud-oss"
	case strings.Contains(ep, "min.io") || strings.Contains(ep, "minio"):
		return "minio"
	}
	return "s3"
}
