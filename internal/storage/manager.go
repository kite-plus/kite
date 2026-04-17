package storage

import (
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// ConfigMeta 单个存储配置在 manager 中的完整元数据视图。
// 既包含驱动需要的 StorageConfig，也包含上层路由策略需要的 priority / 容量 / 默认标记。
type ConfigMeta struct {
	ID                 string
	Name               string
	Driver             string
	DriverConfig       StorageConfig
	Priority           int
	CapacityLimitBytes int64
	IsDefault          bool
	IsActive           bool
}

// Manager 管理多个存储配置的驱动实例。
// 状态在 Reload 时整体替换，避免 CRUD 后默认存储信息与数据库不一致。
type Manager struct {
	mu        sync.RWMutex
	drivers   map[string]StorageDriver
	metas     map[string]ConfigMeta
	defaultID string
}

// NewManager 创建存储管理器。
func NewManager() *Manager {
	return &Manager{
		drivers: make(map[string]StorageDriver),
		metas:   make(map[string]ConfigMeta),
	}
}

// RawConfig 用于 Reload 的原始配置输入。
// 由调用方从数据库加载后传入，避免 manager 依赖 repo 包形成循环引用。
type RawConfig struct {
	ID                 string
	Name               string
	Driver             string
	ConfigJSON         string
	Priority           int
	CapacityLimitBytes int64
	IsDefault          bool
	IsActive           bool
}

// Reload 根据最新的数据库配置整体重建驱动与元数据。
// 原子替换，任何解析/构造失败都会被跳过并返回聚合错误，但不影响其他驱动加载。
func (m *Manager) Reload(rawConfigs []RawConfig) error {
	newDrivers := make(map[string]StorageDriver, len(rawConfigs))
	newMetas := make(map[string]ConfigMeta, len(rawConfigs))
	var defaultID string
	var errs []string

	for _, raw := range rawConfigs {
		if !raw.IsActive {
			continue
		}

		scfg, err := ParseConfig(raw.Driver, json.RawMessage(raw.ConfigJSON))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s(%s): parse: %v", raw.Name, raw.ID, err))
			continue
		}

		driver, err := NewDriver(scfg)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s(%s): build: %v", raw.Name, raw.ID, err))
			continue
		}

		newDrivers[raw.ID] = driver
		newMetas[raw.ID] = ConfigMeta{
			ID:                 raw.ID,
			Name:               raw.Name,
			Driver:             raw.Driver,
			DriverConfig:       scfg,
			Priority:           raw.Priority,
			CapacityLimitBytes: raw.CapacityLimitBytes,
			IsDefault:          raw.IsDefault,
			IsActive:           raw.IsActive,
		}

		if raw.IsDefault {
			defaultID = raw.ID
		}
	}

	// 如果数据库里没有任何 is_default 但有活跃配置，退化为 priority 最小的一个，避免"无默认"导致上传失败。
	if defaultID == "" && len(newMetas) > 0 {
		sorted := sortMetas(newMetas)
		defaultID = sorted[0].ID
	}

	m.mu.Lock()
	m.drivers = newDrivers
	m.metas = newMetas
	m.defaultID = defaultID
	m.mu.Unlock()

	if len(errs) > 0 {
		return fmt.Errorf("reload storage: %d errors: %v", len(errs), errs)
	}
	return nil
}

// Get 获取指定配置 ID 的存储驱动。
func (m *Manager) Get(configID string) (StorageDriver, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	driver, ok := m.drivers[configID]
	if !ok {
		return nil, fmt.Errorf("storage driver not found for config %q", configID)
	}
	return driver, nil
}

// Default 获取默认存储驱动。
func (m *Manager) Default() (StorageDriver, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.defaultID == "" {
		return nil, "", fmt.Errorf("no default storage configured")
	}

	driver, ok := m.drivers[m.defaultID]
	if !ok {
		return nil, "", fmt.Errorf("default storage driver %q not loaded", m.defaultID)
	}
	return driver, m.defaultID, nil
}

// DefaultID 返回当前默认存储配置 ID（无默认时返回空字符串）。
func (m *Manager) DefaultID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultID
}

// Meta 返回指定配置的元数据快照。
func (m *Manager) Meta(configID string) (ConfigMeta, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	meta, ok := m.metas[configID]
	return meta, ok
}

// ActiveMetas 返回所有活跃配置的元数据快照，按 priority 升序、再按 created-like 顺序（ID）稳定排序。
func (m *Manager) ActiveMetas() []ConfigMeta {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return sortMetas(m.metas)
}

// List 列出所有已注册的存储配置 ID。
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.drivers))
	for id := range m.drivers {
		ids = append(ids, id)
	}
	return ids
}

func sortMetas(metas map[string]ConfigMeta) []ConfigMeta {
	out := make([]ConfigMeta, 0, len(metas))
	for _, m := range metas {
		out = append(out, m)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Priority != out[j].Priority {
			return out[i].Priority < out[j].Priority
		}
		return out[i].ID < out[j].ID
	})
	return out
}
