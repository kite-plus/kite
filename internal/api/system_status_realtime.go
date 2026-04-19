package api

import (
	"net/http"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
)

const latencyRingCapacity = 1024

type runtimeTotals struct {
	reqCount       uint64
	errCount       uint64
	cacheableReqs  uint64
	cacheHits      uint64
	totalLatencyNS uint64
	lastLatencyNS  uint64
	bytesIn        uint64
	bytesOut       uint64
	inFlight       int64
	wsClients      int64
}

type latencyRing struct {
	mu     sync.Mutex
	buf    [latencyRingCapacity]uint64
	head   int
	filled int
}

func (r *latencyRing) push(sample uint64) {
	r.mu.Lock()
	r.buf[r.head] = sample
	r.head = (r.head + 1) % latencyRingCapacity
	if r.filled < latencyRingCapacity {
		r.filled++
	}
	r.mu.Unlock()
}

func (r *latencyRing) percentileMS(p float64) float64 {
	r.mu.Lock()
	n := r.filled
	if n == 0 {
		r.mu.Unlock()
		return 0
	}
	samples := make([]uint64, n)
	copy(samples, r.buf[:n])
	r.mu.Unlock()

	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	idx := int(float64(n-1) * p)
	if idx < 0 {
		idx = 0
	}
	if idx >= n {
		idx = n - 1
	}
	return float64(samples[idx]) / 1_000_000
}

type systemStatusSnapshot struct {
	ReqCount       uint64
	ErrCount       uint64
	CacheableReqs  uint64
	CacheHits      uint64
	TotalLatencyNS uint64
	BytesIn        uint64
	BytesOut       uint64
	NetBytesSent   uint64
	NetBytesRecv   uint64
	CPUMicros      uint64
	At             time.Time
}

// RealtimeSystemStatusCollector collects lightweight process and HTTP runtime
// telemetry for live dashboard status updates.
type RealtimeSystemStatusCollector struct {
	start time.Time

	totals  runtimeTotals
	latency latencyRing

	mu   sync.Mutex
	last systemStatusSnapshot
}

type RealtimeSystemStatus struct {
	CPUPercent          float64 `json:"cpu_percent"`
	ProcessCPUPercent   float64 `json:"process_cpu_percent"`
	CPUCores            int     `json:"cpu_cores"`
	MemoryUsedBytes     uint64  `json:"memory_used_bytes"`
	MemoryTotalBytes    uint64  `json:"memory_total_bytes"`
	UploadMbps          float64 `json:"upload_mbps"`
	DownloadMbps        float64 `json:"download_mbps"`
	UploadPercent       float64 `json:"upload_percent"`
	DownloadPercent     float64 `json:"download_percent"`
	APILatencyMS        float64 `json:"api_latency_ms"`
	P95LatencyMS        float64 `json:"p95_latency_ms"`
	CacheHitRatePercent float64 `json:"cache_hit_rate_percent"`
	DiskIOMBps          float64 `json:"disk_io_mbps"`
	ActiveConnections   int64   `json:"active_connections"`
	ErrorRatePercent    float64 `json:"error_rate_percent"`
	UptimeDays          int64   `json:"uptime_days"`
	AllOperational      bool    `json:"all_operational"`
	GeneratedAtUnixSec  int64   `json:"generated_at_unix_sec"`
}

func NewRealtimeSystemStatusCollector() *RealtimeSystemStatusCollector {
	now := time.Now()
	curCPU := readProcessCPUMicros()
	netSent, netRecv := readSystemNetBytes()
	return &RealtimeSystemStatusCollector{
		start: now,
		last: systemStatusSnapshot{
			NetBytesSent: netSent,
			NetBytesRecv: netRecv,
			CPUMicros:    curCPU,
			At:           now,
		},
	}
}

func (c *RealtimeSystemStatusCollector) Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if strings.HasPrefix(ctx.Request.URL.Path, "/api/") {
			atomic.AddInt64(&c.totals.inFlight, 1)
			start := time.Now()
			ctx.Next()

			dur := time.Since(start)
			latencyNS := uint64(dur.Nanoseconds())
			atomic.AddInt64(&c.totals.inFlight, -1)
			atomic.AddUint64(&c.totals.reqCount, 1)
			atomic.AddUint64(&c.totals.totalLatencyNS, latencyNS)
			atomic.StoreUint64(&c.totals.lastLatencyNS, latencyNS)
			c.latency.push(latencyNS)

			status := ctx.Writer.Status()
			if status >= http.StatusBadRequest {
				atomic.AddUint64(&c.totals.errCount, 1)
			}

			// Cache hit rate tracks conditional-GET effectiveness: a GET
			// that returned 304 is a hit; a GET that returned 200 is a miss.
			if ctx.Request.Method == http.MethodGet {
				if status == http.StatusNotModified {
					atomic.AddUint64(&c.totals.cacheableReqs, 1)
					atomic.AddUint64(&c.totals.cacheHits, 1)
				} else if status == http.StatusOK {
					atomic.AddUint64(&c.totals.cacheableReqs, 1)
				}
			}

			if in := ctx.Request.ContentLength; in > 0 {
				atomic.AddUint64(&c.totals.bytesIn, uint64(in))
			}
			if out := ctx.Writer.Size(); out > 0 {
				atomic.AddUint64(&c.totals.bytesOut, uint64(out))
			}
			return
		}

		ctx.Next()
	}
}

func (c *RealtimeSystemStatusCollector) AddWSClient() {
	atomic.AddInt64(&c.totals.wsClients, 1)
}

func (c *RealtimeSystemStatusCollector) RemoveWSClient() {
	atomic.AddInt64(&c.totals.wsClients, -1)
}

func (c *RealtimeSystemStatusCollector) Snapshot() RealtimeSystemStatus {
	now := time.Now()
	netSent, netRecv := readSystemNetBytes()
	cur := systemStatusSnapshot{
		ReqCount:       atomic.LoadUint64(&c.totals.reqCount),
		ErrCount:       atomic.LoadUint64(&c.totals.errCount),
		CacheableReqs:  atomic.LoadUint64(&c.totals.cacheableReqs),
		CacheHits:      atomic.LoadUint64(&c.totals.cacheHits),
		TotalLatencyNS: atomic.LoadUint64(&c.totals.totalLatencyNS),
		BytesIn:        atomic.LoadUint64(&c.totals.bytesIn),
		BytesOut:       atomic.LoadUint64(&c.totals.bytesOut),
		NetBytesSent:   netSent,
		NetBytesRecv:   netRecv,
		CPUMicros:      readProcessCPUMicros(),
		At:             now,
	}

	c.mu.Lock()
	prev := c.last
	c.last = cur
	c.mu.Unlock()

	deltaSecs := cur.At.Sub(prev.At).Seconds()
	if deltaSecs <= 0 {
		deltaSecs = 1
	}

	deltaReq := safeDelta(cur.ReqCount, prev.ReqCount)
	deltaErr := safeDelta(cur.ErrCount, prev.ErrCount)
	deltaLatencyNS := safeDelta(cur.TotalLatencyNS, prev.TotalLatencyNS)
	deltaIn := safeDelta(cur.BytesIn, prev.BytesIn)
	deltaOut := safeDelta(cur.BytesOut, prev.BytesOut)
	deltaNetSent := safeDelta(cur.NetBytesSent, prev.NetBytesSent)
	deltaNetRecv := safeDelta(cur.NetBytesRecv, prev.NetBytesRecv)
	deltaCPU := safeDelta(cur.CPUMicros, prev.CPUMicros)
	totalReq := cur.ReqCount
	totalLatencyNS := cur.TotalLatencyNS

	var apiLatencyMS float64
	if deltaReq > 0 {
		apiLatencyMS = float64(deltaLatencyNS) / float64(deltaReq) / 1_000_000
	} else if totalReq > 0 {
		apiLatencyMS = float64(totalLatencyNS) / float64(totalReq) / 1_000_000
	} else {
		apiLatencyMS = float64(atomic.LoadUint64(&c.totals.lastLatencyNS)) / 1_000_000
	}

	uploadBytes := deltaNetSent
	downloadBytes := deltaNetRecv
	if uploadBytes == 0 {
		uploadBytes = deltaIn
	}
	if downloadBytes == 0 {
		downloadBytes = deltaOut
	}

	uploadMbps := float64(uploadBytes*8) / deltaSecs / 1_000_000
	downloadMbps := float64(downloadBytes*8) / deltaSecs / 1_000_000
	diskIOMBps := float64(deltaIn+deltaOut) / deltaSecs / (1024 * 1024)

	cpuPercent := readSystemCPUPercent()
	processCPUPercent := 0.0
	if deltaCPU > 0 {
		processCPUPercent = (float64(deltaCPU) / 1_000_000) / (deltaSecs * float64(runtime.NumCPU())) * 100
		if processCPUPercent < 0 {
			processCPUPercent = 0
		}
	}
	if cpuPercent <= 0 {
		cpuPercent = processCPUPercent
		if cpuPercent < 0 {
			cpuPercent = 0
		}
	}

	memoryUsed, memoryTotal := readSystemMemoryBytes()

	var errRate float64
	if deltaReq > 0 {
		errRate = (float64(deltaErr) / float64(deltaReq)) * 100
	}

	uploadPct := uploadMbps / 100 * 100
	if uploadPct > 100 {
		uploadPct = 100
	}
	if uploadPct < 0 {
		uploadPct = 0
	}

	downloadPct := downloadMbps / 100 * 100
	if downloadPct > 100 {
		downloadPct = 100
	}
	if downloadPct < 0 {
		downloadPct = 0
	}

	allOperational := errRate < 5.0

	p95MS := c.latency.percentileMS(0.95)

	var cacheHitRate float64
	if cur.CacheableReqs > 0 {
		cacheHitRate = float64(cur.CacheHits) / float64(cur.CacheableReqs) * 100
	}

	return RealtimeSystemStatus{
		CPUPercent:          cpuPercent,
		ProcessCPUPercent:   processCPUPercent,
		CPUCores:            runtime.NumCPU(),
		MemoryUsedBytes:     memoryUsed,
		MemoryTotalBytes:    memoryTotal,
		UploadMbps:          uploadMbps,
		DownloadMbps:        downloadMbps,
		UploadPercent:       uploadPct,
		DownloadPercent:     downloadPct,
		APILatencyMS:        apiLatencyMS,
		P95LatencyMS:        p95MS,
		CacheHitRatePercent: cacheHitRate,
		DiskIOMBps:          diskIOMBps,
		ActiveConnections:   atomic.LoadInt64(&c.totals.inFlight) + atomic.LoadInt64(&c.totals.wsClients),
		ErrorRatePercent:    errRate,
		UptimeDays:          int64(time.Since(c.start).Hours() / 24),
		AllOperational:      allOperational,
		GeneratedAtUnixSec:  now.Unix(),
	}
}

func safeDelta(cur uint64, prev uint64) int64 {
	if cur >= prev {
		return int64(cur - prev)
	}
	return 0
}

type SystemStatusRealtimeHandler struct {
	collector *RealtimeSystemStatusCollector

	ticketMu sync.Mutex
	tickets  map[string]time.Time

	upgrader websocket.Upgrader
}

func NewSystemStatusRealtimeHandler(collector *RealtimeSystemStatusCollector) *SystemStatusRealtimeHandler {
	return &SystemStatusRealtimeHandler{
		collector: collector,
		tickets:   make(map[string]time.Time),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *SystemStatusRealtimeHandler) IssueWSTicket(c *gin.Context) {
	ticket := uuid.NewString()
	expiresAt := time.Now().Add(30 * time.Second)

	h.ticketMu.Lock()
	h.tickets[ticket] = expiresAt
	h.cleanupExpiredLocked()
	h.ticketMu.Unlock()

	success(c, gin.H{
		"ticket":      ticket,
		"expires_at":  expiresAt.Unix(),
		"expires_in":  30,
		"ws_endpoint": "/api/v1/admin/system-status/ws",
	})
}

func (h *SystemStatusRealtimeHandler) Stream(c *gin.Context) {
	ticket := strings.TrimSpace(c.Query("ticket"))
	if ticket == "" || !h.consumeValidTicket(ticket) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    40100,
			"message": "invalid or expired ws ticket",
			"data":    nil,
		})
		return
	}

	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	h.collector.AddWSClient()
	defer h.collector.RemoveWSClient()

	_ = conn.WriteJSON(h.collector.Snapshot())

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteJSON(h.collector.Snapshot()); err != nil {
			return
		}
	}
}

func (h *SystemStatusRealtimeHandler) consumeValidTicket(ticket string) bool {
	now := time.Now()
	h.ticketMu.Lock()
	defer h.ticketMu.Unlock()

	expiresAt, ok := h.tickets[ticket]
	if !ok {
		h.cleanupExpiredLocked()
		return false
	}
	delete(h.tickets, ticket)
	h.cleanupExpiredLocked()
	return expiresAt.After(now)
}

func (h *SystemStatusRealtimeHandler) cleanupExpiredLocked() {
	now := time.Now()
	for k, exp := range h.tickets {
		if !exp.After(now) {
			delete(h.tickets, k)
		}
	}
}

func readProcessCPUMicros() uint64 {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0
	}
	userMicros := uint64(ru.Utime.Sec)*1_000_000 + uint64(ru.Utime.Usec)
	sysMicros := uint64(ru.Stime.Sec)*1_000_000 + uint64(ru.Stime.Usec)
	return userMicros + sysMicros
}

func readSystemCPUPercent() float64 {
	vals, err := cpu.Percent(0, false)
	if err != nil || len(vals) == 0 {
		return 0
	}
	if vals[0] < 0 {
		return 0
	}
	return vals[0]
}

func readSystemMemoryBytes() (used uint64, total uint64) {
	vm, err := mem.VirtualMemory()
	if err == nil && vm != nil && vm.Total > 0 {
		used = vm.Used
		total = vm.Total
		if used > total {
			used = total
		}
		if used == 0 && vm.Available < vm.Total {
			used = vm.Total - vm.Available
		}
		if used > 0 || total > 0 {
			return used, total
		}
	}

	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	if ms.Sys > 0 {
		used = ms.Alloc
		total = ms.Sys
		if used > total {
			used = total
		}
		return used, total
	}

	// Keep dashboard formatting stable even on very restricted environments.
	return 0, 1
}

func readSystemNetBytes() (sent uint64, recv uint64) {
	stats, err := gnet.IOCounters(true)
	if err != nil || len(stats) == 0 {
		return 0, 0
	}

	for _, item := range stats {
		sent += item.BytesSent
		recv += item.BytesRecv
	}

	return sent, recv
}
