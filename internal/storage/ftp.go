package storage

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

// FTPDriver 基于 FTP 协议的存储驱动。
// 每次操作建立独立连接，避免连接保持期间的超时和中断问题。
type FTPDriver struct {
	addr     string
	username string
	password string
	basePath string
	baseURL  string
}

// NewFTPDriver 根据 FTPConfig 创建一个 FTP 存储驱动实例。
// 创建时会尝试连接并登录一次，验证配置有效。
func NewFTPDriver(cfg FTPConfig) (*FTPDriver, error) {
	if cfg.Host == "" {
		return nil, fmt.Errorf("ftp driver: host is required")
	}
	if cfg.Username == "" {
		return nil, fmt.Errorf("ftp driver: username is required")
	}

	port := cfg.Port
	if port == 0 {
		port = 21
	}

	d := &FTPDriver{
		addr:     fmt.Sprintf("%s:%d", cfg.Host, port),
		username: cfg.Username,
		password: cfg.Password,
		basePath: strings.TrimRight(strings.TrimSpace(cfg.BasePath), "/"),
		baseURL:  strings.TrimRight(cfg.BaseURL, "/"),
	}

	conn, err := d.dial()
	if err != nil {
		return nil, err
	}
	_ = conn.Quit()

	return d, nil
}

// dial 建立 FTP 连接并完成登录。
func (d *FTPDriver) dial() (*ftp.ServerConn, error) {
	conn, err := ftp.Dial(d.addr, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return nil, fmt.Errorf("ftp dial %s: %w", d.addr, err)
	}
	if err := conn.Login(d.username, d.password); err != nil {
		_ = conn.Quit()
		return nil, fmt.Errorf("ftp login: %w", err)
	}
	return conn, nil
}

// remotePath 将相对 key 拼接成远程绝对路径。
func (d *FTPDriver) remotePath(key string) string {
	clean := path.Clean("/" + strings.TrimLeft(key, "/"))
	if d.basePath == "" {
		return clean
	}
	return d.basePath + clean
}

// ensureDir 递归创建远程目录。
func (d *FTPDriver) ensureDir(conn *ftp.ServerConn, dir string) error {
	if dir == "" || dir == "/" || dir == "." {
		return nil
	}
	parts := strings.Split(strings.Trim(dir, "/"), "/")
	current := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		current = current + "/" + p
		if err := conn.MakeDir(current); err != nil {
			// 目录已存在时 FTP 返回错误，忽略这种情况
			msg := strings.ToLower(err.Error())
			if !strings.Contains(msg, "exists") && !strings.Contains(msg, "file exists") && !strings.Contains(msg, "already") {
				// 再次探测目录，如果能切进去说明已经存在
				if changeErr := conn.ChangeDir(current); changeErr == nil {
					continue
				}
				return fmt.Errorf("ftp mkdir %q: %w", current, err)
			}
		}
	}
	return nil
}

// Put 上传文件到 FTP 服务器。
func (d *FTPDriver) Put(_ context.Context, key string, reader io.Reader, _ int64, _ string) error {
	conn, err := d.dial()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Quit() }()

	full := d.remotePath(key)
	if err := d.ensureDir(conn, path.Dir(full)); err != nil {
		return err
	}
	if err := conn.Stor(full, reader); err != nil {
		return fmt.Errorf("ftp put %q: %w", key, err)
	}
	return nil
}

// Get 从 FTP 服务器读取文件。
// FTP 连接需要与数据流生命周期绑定，因此返回自定义的 ReadCloser。
func (d *FTPDriver) Get(_ context.Context, key string) (io.ReadCloser, int64, error) {
	conn, err := d.dial()
	if err != nil {
		return nil, 0, err
	}

	full := d.remotePath(key)
	size, err := conn.FileSize(full)
	if err != nil {
		_ = conn.Quit()
		return nil, 0, fmt.Errorf("ftp size %q: %w", key, err)
	}

	resp, err := conn.Retr(full)
	if err != nil {
		_ = conn.Quit()
		return nil, 0, fmt.Errorf("ftp get %q: %w", key, err)
	}

	return &ftpReadCloser{resp: resp, conn: conn}, size, nil
}

// ftpReadCloser 将 FTP 响应的读取与连接生命周期绑定。
type ftpReadCloser struct {
	resp *ftp.Response
	conn *ftp.ServerConn
}

func (r *ftpReadCloser) Read(p []byte) (int, error) { return r.resp.Read(p) }

func (r *ftpReadCloser) Close() error {
	respErr := r.resp.Close()
	quitErr := r.conn.Quit()
	if respErr != nil {
		return respErr
	}
	return quitErr
}

// Delete 从 FTP 服务器删除文件，不存在时返回 nil。
func (d *FTPDriver) Delete(_ context.Context, key string) error {
	conn, err := d.dial()
	if err != nil {
		return err
	}
	defer func() { _ = conn.Quit() }()

	full := d.remotePath(key)
	if err := conn.Delete(full); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") || strings.Contains(msg, "550") {
			return nil
		}
		return fmt.Errorf("ftp delete %q: %w", key, err)
	}
	return nil
}

// Exists 检查 FTP 服务器上文件是否存在。
func (d *FTPDriver) Exists(_ context.Context, key string) (bool, error) {
	conn, err := d.dial()
	if err != nil {
		return false, err
	}
	defer func() { _ = conn.Quit() }()

	full := d.remotePath(key)
	if _, err := conn.FileSize(full); err != nil {
		msg := strings.ToLower(err.Error())
		if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") || strings.Contains(msg, "550") {
			return false, nil
		}
		return false, fmt.Errorf("ftp exists %q: %w", key, err)
	}
	return true, nil
}

// URL 返回文件的公开访问 URL。
// FTP 本身无公网 HTTP URL，依赖 baseURL 配置（通常为反向代理地址）。
func (d *FTPDriver) URL(key string) string {
	if d.baseURL == "" {
		return "/" + strings.TrimLeft(key, "/")
	}
	return d.baseURL + "/" + strings.TrimLeft(key, "/")
}

// SignedURL FTP 不支持预签名 URL，直接返回 URL。
func (d *FTPDriver) SignedURL(_ context.Context, key string, _ time.Duration) (string, error) {
	return d.URL(key), nil
}
