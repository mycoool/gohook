package syncnode

import (
	"context"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
)

func collectLocalRuntime(ctx context.Context) NodeRuntimeStatus {
	out := NodeRuntimeStatus{
		UpdatedAt: time.Now(),
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

	return out
}
