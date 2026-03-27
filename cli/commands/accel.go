package commands

import (
	"fmt"
	"io"
	"sort"
	"strings"

	accelpkg "github.com/HazelnutParadise/insyra/accel"
	clienv "github.com/HazelnutParadise/insyra/cli/env"
	insyra "github.com/HazelnutParadise/insyra"
)

func init() {
	_ = Register(&CommandHandler{
		Name:        "accel",
		Usage:       "accel <devices|cache|run> [--mode auto|cpu|gpu|strict-gpu]",
		Description: "Inspect acceleration backends and session reports",
		Run:         runAccelCommand,
	})
}

func runAccelCommand(ctx *ExecContext, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: accel <devices|cache|run> [--mode auto|cpu|gpu|strict-gpu]")
	}

	action := strings.ToLower(args[0])
	cfg, err := accelConfigFromArgs(args[1:])
	if err != nil {
		return err
	}

	switch action {
	case "devices":
		session, err := accelpkg.Open(cfg)
		if err != nil {
			if session != nil {
				defer func() { _ = session.Close() }()
				renderAccelDevices(ctx.Output, session)
			}
			return err
		}
		defer func() { _ = session.Close() }()
		renderAccelDevices(ctx.Output, session)
		return nil
	case "cache":
		session, err := accelpkg.Open(cfg)
		if session != nil {
			hydrateAccelCacheFromContext(session, ctx)
		}
		if err != nil {
			if session != nil {
				defer func() { _ = session.Close() }()
				renderAccelCache(ctx.Output, session.Report(), session.CacheSnapshot())
			}
			return err
		}
		defer func() { _ = session.Close() }()
		renderAccelCache(ctx.Output, session.Report(), session.CacheSnapshot())
		return nil
	case "run":
		session, err := accelpkg.Open(cfg)
		if session != nil {
			defer func() { _ = session.Close() }()
			renderAccelRun(ctx.Output, session.Report(), session.PlanShardable())
		}
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown accel action: %s", action)
	}
}

func accelConfigFromArgs(args []string) (accelpkg.Config, error) {
	cfg := accelpkg.Config{}
	explicitMode := ""
	for idx := 0; idx < len(args); idx++ {
		switch args[idx] {
		case "--mode":
			if idx+1 >= len(args) {
				return cfg, fmt.Errorf("usage: --mode auto|cpu|gpu|strict-gpu")
			}
			explicitMode = args[idx+1]
			idx++
		default:
			return cfg, fmt.Errorf("unknown accel argument: %s", args[idx])
		}
	}

	mode, err := resolveAccelMode(explicitMode)
	if err != nil {
		return cfg, err
	}
	cfg.Mode = mode
	return cfg, nil
}

func resolveAccelMode(explicit string) (accelpkg.Mode, error) {
	raw := strings.TrimSpace(strings.ToLower(explicit))
	if raw == "" {
		cfg, err := clienv.LoadGlobalConfig()
		if err != nil {
			return "", err
		}
		raw = strings.TrimSpace(strings.ToLower(cfg.AccelMode))
	}
	if raw == "" {
		raw = string(accelpkg.ModeAuto)
	}

	switch accelpkg.Mode(raw) {
	case accelpkg.ModeAuto, accelpkg.ModeCPU, accelpkg.ModeGPU, accelpkg.ModeStrictGPU:
		return accelpkg.Mode(raw), nil
	default:
		return "", fmt.Errorf("invalid accel mode: %s", raw)
	}
}

func renderAccelDevices(out io.Writer, session *accelpkg.Session) {
	devices := session.Devices()
	report := session.Report()
	if len(devices) == 0 {
		_, _ = fmt.Fprintf(out, "no accel devices detected backend=%s fallback=%s\n", report.SelectedBackend, report.FallbackReason)
		return
	}
	for _, device := range devices {
		_, _ = fmt.Fprintf(
			out,
			"id=%s backend=%s probe=%s vendor=%s type=%s memory=%s budget=%d accelerated=%t caps=%s\n",
			device.ID,
			device.Backend,
			device.ProbeSource,
			device.Vendor,
			device.Type,
			device.MemoryClass,
			device.BudgetBytes,
			report.Accelerated,
			formatCapabilities(device.CapabilitySummary),
		)
	}
}

func renderAccelCache(out io.Writer, report accelpkg.Report, snapshot accelpkg.CacheSnapshot) {
	_, _ = fmt.Fprintf(
		out,
		"backend=%s fallback=%s resident_buffers=%d resident_bytes=%d budget_bytes=%d evicted_buffers=%d evicted_bytes=%d\n",
		report.SelectedBackend,
		report.FallbackReason,
		snapshot.ResidentBuffers,
		snapshot.ResidentBytes,
		snapshot.BudgetBytes,
		snapshot.EvictedBuffers,
		snapshot.EvictedBytes,
	)
	for _, usage := range snapshot.DeviceUsage {
		_, _ = fmt.Fprintf(
			out,
			"device %s resident_buffers=%d resident_bytes=%d budget_bytes=%d\n",
			usage.DeviceID,
			usage.ResidentBuffers,
			usage.ResidentBytes,
			usage.BudgetBytes,
		)
	}
	for _, entry := range snapshot.Entries {
		deviceIDs := "none"
		if len(entry.DeviceIDs) > 0 {
			deviceIDs = strings.Join(entry.DeviceIDs, ",")
		}
		_, _ = fmt.Fprintf(
			out,
			"entry dataset=%s buffer=%s type=%s len=%d bytes=%d devices=%s\n",
			entry.DatasetName,
			entry.BufferName,
			entry.Type,
			entry.Len,
			entry.ResidentBytes,
			deviceIDs,
		)
	}
}

func renderAccelRun(out io.Writer, report accelpkg.Report, plan accelpkg.ShardPlan) {
	selected := "none"
	if len(report.SelectedDeviceIDs) > 0 {
		selected = strings.Join(report.SelectedDeviceIDs, ",")
	}
	shardDevices := "none"
	if len(plan.DeviceIDs) > 0 {
		shardDevices = strings.Join(plan.DeviceIDs, ",")
	}
	_, _ = fmt.Fprintf(
		out,
		"mode=%s accelerated=%t backend=%s devices=%s discovered=%.0f selected=%.0f planned=%d shard_devices=%s shard_budget=%d reason=%s\n",
		report.Mode,
		report.Accelerated,
		report.SelectedBackend,
		selected,
		report.Metrics["devices.discovered"],
		report.Metrics["devices.selected"],
		len(plan.DeviceIDs),
		shardDevices,
		plan.TotalBudgetBytes,
		report.FallbackReason,
	)
}

func hydrateAccelCacheFromContext(session *accelpkg.Session, ctx *ExecContext) {
	if session == nil || ctx == nil {
		return
	}
	for _, value := range ctx.Vars {
		switch typed := value.(type) {
		case *insyra.DataList:
			_, _ = session.ProjectDataList(typed)
		case *insyra.DataTable:
			_, _ = session.ProjectDataTable(typed)
		}
	}
}

func formatCapabilities(caps map[string]bool) string {
	if len(caps) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(caps))
	for key, enabled := range caps {
		if enabled {
			keys = append(keys, key)
		}
	}
	if len(keys) == 0 {
		return "none"
	}
	sort.Strings(keys)
	return strings.Join(keys, ",")
}
