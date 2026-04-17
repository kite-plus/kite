package storage

import (
	"context"
	"fmt"
	"sync/atomic"
)

// 上传策略常量。存储在 settings 表的 `storage.upload_policy` 键下。
const (
	PolicySingle          = "single"           // 只用默认存储，失败即失败
	PolicyPrimaryFallback = "primary_fallback" // 按 priority 遍历，容量满/写失败时降级下一个
	PolicyRoundRobin      = "round_robin"      // 在所有活跃存储间轮询
	PolicyMirror          = "mirror"           // 主存储同步写入，副本后台并发
)

// UsageFn 返回指定存储配置当前已用字节数（由 repo 层实现注入）。
type UsageFn func(ctx context.Context, configID string) (int64, error)

// PolicyFn 返回当前上传策略字符串（从 settings 表读取）。
type PolicyFn func(ctx context.Context) (string, error)

// Router 基于 Manager 的驱动缓存和运行时策略，决定一次上传应该写入哪些存储。
type Router struct {
	mgr      *Manager
	usageFn  UsageFn
	policyFn PolicyFn
	rrCursor atomic.Uint64
}

// NewRouter 创建一个路由器。
// usageFn 用于容量过滤；policyFn 用于读取策略；两者都必填。
func NewRouter(mgr *Manager, usageFn UsageFn, policyFn PolicyFn) *Router {
	return &Router{mgr: mgr, usageFn: usageFn, policyFn: policyFn}
}

// Target 一个上传目标：包含元数据和对应的驱动实例。
type Target struct {
	Meta   ConfigMeta
	Driver StorageDriver
}

// Plan 单次上传的路由计划。
// Mode 决定上层如何消费 Targets：
//   - single / round_robin: 写 Targets[0]，失败即失败
//   - primary_fallback: 依次尝试 Targets[0..n]，首个成功即视为成功
//   - mirror: Targets[0] 为主节点必须成功；Targets[1..] 为副本，后台并发
type Plan struct {
	Mode    string
	Targets []Target
}

// Plan 根据当前策略、优先级和容量，为一次 size 字节的上传挑选目标。
// 若没有任何符合容量要求的活跃存储，返回错误。
func (r *Router) Plan(ctx context.Context, size int64) (*Plan, error) {
	mode, err := r.policyFn(ctx)
	if err != nil || mode == "" {
		mode = PolicySingle
	}

	metas := r.mgr.ActiveMetas()
	if len(metas) == 0 {
		return nil, fmt.Errorf("no active storage configured")
	}

	switch mode {
	case PolicyMirror:
		// 所有活跃存储都是目标，主节点优先选容量够的，其他按 priority 排列。
		all := r.buildTargets(metas)
		eligible := r.filterByCapacity(ctx, all, size)
		if len(eligible) == 0 {
			return nil, fmt.Errorf("no storage has enough capacity for %d bytes", size)
		}
		// 保证主节点（Targets[0]）是容量够的那一个；副本可以包含容量够的其他节点。
		primary := eligible[0]
		var replicas []Target
		for _, t := range eligible[1:] {
			replicas = append(replicas, t)
		}
		return &Plan{Mode: PolicyMirror, Targets: append([]Target{primary}, replicas...)}, nil

	case PolicyPrimaryFallback:
		all := r.buildTargets(metas)
		eligible := r.filterByCapacity(ctx, all, size)
		if len(eligible) == 0 {
			return nil, fmt.Errorf("no storage has enough capacity for %d bytes", size)
		}
		return &Plan{Mode: PolicyPrimaryFallback, Targets: eligible}, nil

	case PolicyRoundRobin:
		all := r.buildTargets(metas)
		eligible := r.filterByCapacity(ctx, all, size)
		if len(eligible) == 0 {
			return nil, fmt.Errorf("no storage has enough capacity for %d bytes", size)
		}
		idx := int(r.rrCursor.Add(1)-1) % len(eligible)
		return &Plan{Mode: PolicyRoundRobin, Targets: []Target{eligible[idx]}}, nil

	default: // PolicySingle
		defID := r.mgr.DefaultID()
		if defID == "" {
			return nil, fmt.Errorf("no default storage configured")
		}
		driver, err := r.mgr.Get(defID)
		if err != nil {
			return nil, err
		}
		meta, _ := r.mgr.Meta(defID)
		// single 策略仍然尊重容量上限：超过时直接失败，不静默换别的存储，避免行为不可预期。
		if meta.CapacityLimitBytes > 0 {
			used, _ := r.usageFn(ctx, defID)
			if used+size > meta.CapacityLimitBytes {
				return nil, fmt.Errorf("default storage %q is full", meta.Name)
			}
		}
		return &Plan{Mode: PolicySingle, Targets: []Target{{Meta: meta, Driver: driver}}}, nil
	}
}

// buildTargets 将 metas 解析为 (meta, driver) 对；已反映 priority 排序。
func (r *Router) buildTargets(metas []ConfigMeta) []Target {
	out := make([]Target, 0, len(metas))
	for _, m := range metas {
		driver, err := r.mgr.Get(m.ID)
		if err != nil {
			continue
		}
		out = append(out, Target{Meta: m, Driver: driver})
	}
	return out
}

// filterByCapacity 过滤掉已满的存储；CapacityLimitBytes==0 视为无上限。
// 用量查询失败时保守放行（不因为临时查询错误阻塞上传）。
func (r *Router) filterByCapacity(ctx context.Context, targets []Target, size int64) []Target {
	out := make([]Target, 0, len(targets))
	for _, t := range targets {
		if t.Meta.CapacityLimitBytes <= 0 {
			out = append(out, t)
			continue
		}
		used, err := r.usageFn(ctx, t.Meta.ID)
		if err != nil {
			out = append(out, t)
			continue
		}
		if used+size <= t.Meta.CapacityLimitBytes {
			out = append(out, t)
		}
	}
	return out
}
