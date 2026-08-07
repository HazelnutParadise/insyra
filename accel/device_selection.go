package accel

import (
	"os"
	"strconv"
	"strings"
)

const accelDevicesEnv = "INSYRA_ACCEL_DEVICES"

type deviceSelection struct {
	unmatched  []UnmatchedDeviceSelector
	emptyBound string
}

// applyDeviceBounds resolves both hard bounds against the same discovery list
// and only then intersects them. This keeps discovery indices stable: a config
// index never changes meaning just because the environment masked another
// device.
func applyDeviceBounds(devices []Device, cfg Config) ([]Device, deviceSelection) {
	envSelectors := parseDeviceSelectors(os.Getenv(accelDevicesEnv))
	configSelectors := normalizeDeviceSelectors(cfg.Devices)
	envBound := len(envSelectors) > 0
	configBound := len(configSelectors) > 0
	if !envBound && !configBound {
		return devices, deviceSelection{}
	}

	envMatches, envUnmatched := resolveDeviceSelectors(envSelectors, devices, accelDevicesEnv)
	configMatches, configUnmatched := resolveDeviceSelectors(configSelectors, devices, "Config.Devices")
	selection := deviceSelection{
		unmatched: append(envUnmatched, configUnmatched...),
	}

	eligible := make([]Device, 0, len(devices))
	for idx, device := range devices {
		if envBound {
			if _, ok := envMatches[idx]; !ok {
				continue
			}
		}
		if configBound {
			if _, ok := configMatches[idx]; !ok {
				continue
			}
		}
		eligible = append(eligible, device)
	}

	if len(eligible) == 0 {
		switch {
		case envBound && len(envMatches) == 0:
			selection.emptyBound = accelDevicesEnv
		case configBound && len(configMatches) == 0:
			selection.emptyBound = "Config.Devices"
		case envBound && configBound:
			selection.emptyBound = accelDevicesEnv + " and Config.Devices"
		case envBound:
			selection.emptyBound = accelDevicesEnv
		case configBound:
			selection.emptyBound = "Config.Devices"
		}
	}
	return eligible, selection
}

func parseDeviceSelectors(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return normalizeDeviceSelectors(strings.Split(value, ","))
}

func normalizeDeviceSelectors(selectors []string) []string {
	normalized := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		selector = strings.TrimSpace(selector)
		if selector != "" {
			normalized = append(normalized, selector)
		}
	}
	return normalized
}

func resolveDeviceSelectors(selectors []string, devices []Device, bound string) (map[int]struct{}, []UnmatchedDeviceSelector) {
	matched := make(map[int]struct{}, len(selectors))
	var unmatched []UnmatchedDeviceSelector
	for _, selector := range selectors {
		found := false
		for idx, device := range devices {
			if device.ID == selector {
				matched[idx] = struct{}{}
				found = true
			}
		}
		if !found {
			if idx, err := strconv.Atoi(selector); err == nil && idx >= 0 && idx < len(devices) {
				matched[idx] = struct{}{}
				found = true
			}
		}
		if !found {
			unmatched = append(unmatched, UnmatchedDeviceSelector{Bound: bound, Selector: selector})
		}
	}
	return matched, unmatched
}
