package storage

import (
	"encoding/json"
	"fmt"
)

// ParseConfig deserialises the raw JSON payload into a StorageConfig for the given driver.
// Startup loading and the admin CRUD handlers share this logic so the driver switch cannot
// drift and silently drop oss/cos/ftp configurations at boot time.
func ParseConfig(driver string, raw json.RawMessage) (StorageConfig, error) {
	scfg := StorageConfig{Driver: driver}
	switch driver {
	case DriverLocal:
		var lc LocalConfig
		if err := json.Unmarshal(raw, &lc); err != nil {
			return scfg, fmt.Errorf("parse local config: %w", err)
		}
		scfg.Local = &lc
	case DriverS3, DriverOSS, DriverCOS, DriverOBS, DriverBOS:
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
