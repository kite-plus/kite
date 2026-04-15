package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalDriver 本地文件系统存储驱动。
// 文件存储在服务器本地磁盘，通过 HTTP 服务直接提供访问。
type LocalDriver struct {
	basePath string // 文件存储根目录（绝对路径）
	baseURL  string // 访问 URL 前缀
}

// NewLocalDriver 创建本地存储驱动实例。
func NewLocalDriver(cfg LocalConfig) (*LocalDriver, error) {
	if cfg.BasePath == "" {
		return nil, fmt.Errorf("local driver: base_path is required")
	}
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("local driver: base_url is required")
	}

	// 确保存储目录存在
	absPath, err := filepath.Abs(cfg.BasePath)
	if err != nil {
		return nil, fmt.Errorf("local driver: resolve base_path: %w", err)
	}
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return nil, fmt.Errorf("local driver: create base_path %q: %w", absPath, err)
	}

	return &LocalDriver{
		basePath: absPath,
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
	}, nil
}

// resolveKey 将外部 key 解析为位于 basePath 之内的绝对路径。
// 拒绝包含 ".." 等导致路径穿越的输入。
func (d *LocalDriver) resolveKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("local driver: empty key")
	}
	clean := filepath.Clean(filepath.FromSlash(key))
	if filepath.IsAbs(clean) || strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("local driver: invalid key %q", key)
	}
	fullPath := filepath.Join(d.basePath, clean)
	rel, err := filepath.Rel(d.basePath, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", fmt.Errorf("local driver: path traversal detected in key %q", key)
	}
	return fullPath, nil
}

// Put 将文件写入本地磁盘。
// 自动创建 key 路径中的中间目录。
func (d *LocalDriver) Put(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	fullPath, err := d.resolveKey(key)
	if err != nil {
		return err
	}

	// 创建父目录
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("local put mkdir %q: %w", dir, err)
	}

	// 先写入临时文件，再原子重命名，避免写入一半时被读取到
	tmpFile, err := os.CreateTemp(dir, ".kite-upload-*")
	if err != nil {
		return fmt.Errorf("local put create temp: %w", err)
	}
	tmpPath := tmpFile.Name()

	_, writeErr := io.Copy(tmpFile, reader)
	closeErr := tmpFile.Close()

	if writeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("local put write %q: %w", key, writeErr)
	}
	if closeErr != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("local put close %q: %w", key, closeErr)
	}

	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("local put rename %q: %w", key, err)
	}

	return nil
}

// Get 从本地磁盘读取文件。
// 返回的 ReadCloser 由调用方负责关闭。
func (d *LocalDriver) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	fullPath, err := d.resolveKey(key)
	if err != nil {
		return nil, 0, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, fmt.Errorf("local get %q: %w", key, err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, 0, fmt.Errorf("local get stat %q: %w", key, err)
	}

	return file, info.Size(), nil
}

// Delete 从本地磁盘删除文件。
// 文件不存在时返回 nil（幂等删除）。
func (d *LocalDriver) Delete(_ context.Context, key string) error {
	fullPath, err := d.resolveKey(key)
	if err != nil {
		return err
	}

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local delete %q: %w", key, err)
	}
	return nil
}

// Exists 检查本地磁盘上文件是否存在。
func (d *LocalDriver) Exists(_ context.Context, key string) (bool, error) {
	fullPath, err := d.resolveKey(key)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("local exists %q: %w", key, err)
	}
	return true, nil
}

// URL 生成文件的访问 URL。
func (d *LocalDriver) URL(key string) string {
	return d.baseURL + "/" + key
}

// SignedURL 本地存储不支持预签名 URL，直接返回普通 URL。
func (d *LocalDriver) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return d.URL(key), nil
}
