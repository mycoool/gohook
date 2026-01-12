package nodeclient

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

type runtimeStatus struct {
	Type            string  `json:"type"`
	NodeID          uint    `json:"nodeId"`
	UpdatedAt       string  `json:"updatedAt"`
	Hostname        string  `json:"hostname,omitempty"`
	UptimeSec       uint64  `json:"uptimeSec,omitempty"`
	CPUPercent      float64 `json:"cpuPercent,omitempty"`
	MemUsedPercent  float64 `json:"memUsedPercent,omitempty"`
	Load1           float64 `json:"load1,omitempty"`
	Load1Percent    float64 `json:"load1Percent,omitempty"`
	DiskUsedPercent float64 `json:"diskUsedPercent,omitempty"`
	CPUCores        int     `json:"cpuCores,omitempty"`
}

func collectRuntimeStatus(ctx context.Context, nodeID uint) runtimeStatus {
	out := runtimeStatus{
		Type:      "node_status",
		NodeID:    nodeID,
		UpdatedAt: time.Now().Format(time.RFC3339),
	}

	if hi, err := host.InfoWithContext(ctx); err == nil && hi != nil {
		out.Hostname = hi.Hostname
		out.UptimeSec = hi.Uptime
	}
	if avg, err := load.AvgWithContext(ctx); err == nil && avg != nil {
		out.Load1 = avg.Load1
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil && vm != nil {
		out.MemUsedPercent = vm.UsedPercent
	}
	if du, err := disk.UsageWithContext(ctx, "/"); err == nil && du != nil {
		out.DiskUsedPercent = du.UsedPercent
	}
	if perc, err := cpu.PercentWithContext(ctx, 0, false); err == nil && len(perc) > 0 {
		out.CPUPercent = perc[0]
	}

	cores := 0
	if counts, err := cpu.CountsWithContext(ctx, true); err == nil && counts > 0 {
		cores = counts
	} else {
		cores = runtime.NumCPU()
	}
	if cores > 0 {
		out.CPUCores = cores
		out.Load1Percent = (out.Load1 / float64(cores)) * 100
	}

	return out
}
