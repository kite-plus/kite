package service

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	UploadMaxFileSizeMBSettingKey       = "upload.max_file_size_mb"
	uploadSizeMBBytes             int64 = 1024 * 1024
)

// DefaultUploadMaxFileSizeMB converts the compiled default byte limit into the
// integer MB value stored in runtime settings.
func DefaultUploadMaxFileSizeMB(maxBytes int64) string {
	mb := maxBytes / uploadSizeMBBytes
	if maxBytes%uploadSizeMBBytes != 0 {
		mb++
	}
	if mb < 1 {
		mb = 1
	}
	return strconv.FormatInt(mb, 10)
}

// NormalizeUploadMaxFileSizeMB validates and normalizes the admin-facing upload
// limit value. The persisted form is always a trimmed positive integer in MB.
func NormalizeUploadMaxFileSizeMB(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("cannot be empty")
	}

	mb, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return "", fmt.Errorf("must be a positive integer in MB")
	}
	if mb < 1 {
		return "", fmt.Errorf("must be at least 1 MB")
	}
	if mb > (1<<63-1)/uploadSizeMBBytes {
		return "", fmt.Errorf("value is too large")
	}

	return strconv.FormatInt(mb, 10), nil
}

// ParseUploadMaxFileSizeBytes parses the stored MB setting into bytes.
func ParseUploadMaxFileSizeBytes(raw string) (int64, error) {
	normalized, err := NormalizeUploadMaxFileSizeMB(raw)
	if err != nil {
		return 0, err
	}
	mb, _ := strconv.ParseInt(normalized, 10, 64)
	return mb * uploadSizeMBBytes, nil
}
