package syncnode

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
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

func StartLocalRuntimeBroadcaster(ctx context.Context) {
	interval := 5 * time.Second
	if raw := strings.TrimSpace(os.Getenv("SYNC_LOCAL_RUNTIME_INTERVAL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			interval = d
		} else if sec, err := strconv.Atoi(raw); err == nil && sec > 0 {
			interval = time.Duration(sec) * time.Second
		}
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				runtime := collectLocalRuntime(ctx)
				broadcastWS(wsTypeSyncNodeStatus, map[string]any{"nodeId": 0, "runtime": runtime})
			}
		}
	}()
}
