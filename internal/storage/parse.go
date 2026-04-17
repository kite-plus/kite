package storage

import (
	"encoding/json"
	"fmt"
)

// ParseConfig 根据 driver 将原始 JSON 反序列化为 StorageConfig。
// main 启动加载和 api CRUD 共用此逻辑，避免 driver 分支不同步导致 oss/cos/ftp 启动时被跳过。
func ParseConfig(driver string, raw json.RawMessage) (StorageConfig, error) {
	scfg := StorageConfig{Driver: driver}
	switch driver {
	case DriverLocal:
		var lc LocalConfig
		if err := json.Unmarshal(raw, &lc); err != nil {
			return scfg, fmt.Errorf("parse local config: %w", err)
		}
		scfg.Local = &lc
	case DriverS3, DriverOSS, DriverCOS:
		var sc S3Config
		if err := json.Unmarshal(raw, &sc); err != nil {
			return scfg, fmt.Errorf("parse %s config: %w", driver, err)
		}
		scfg.S3 = &sc
	case DriverFTP:
		var fc FTPConfig
		if err := json.Unmarshal(raw, &fc); err != nil {
			return scfg, fmt.Errorf("parse ftp config: %w", err)
		}
		scfg.FTP = &fc
	default:
		return scfg, fmt.Errorf("unknown driver %q", driver)
	}
	return scfg, nil
}
