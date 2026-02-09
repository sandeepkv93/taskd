package update

import (
	"os"
	"strconv"
	"strings"
)

type RuntimeConfig struct {
	DesktopNotifications      bool
	FocusWorkMinutes          int
	FocusBreakMinutes         int
	ProductivityAvailableMins int
	SchedulerBuffer           int
}

func DefaultRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		DesktopNotifications:      false,
		FocusWorkMinutes:          25,
		FocusBreakMinutes:         5,
		ProductivityAvailableMins: 60,
		SchedulerBuffer:           64,
	}
}

func RuntimeConfigFromEnv(base RuntimeConfig) RuntimeConfig {
	cfg := base
	if v, ok := getEnvBool("TASKD_DESKTOP_NOTIFICATIONS"); ok {
		cfg.DesktopNotifications = v
	}
	if v, ok := getEnvInt("TASKD_FOCUS_WORK_MINUTES"); ok && v > 0 {
		cfg.FocusWorkMinutes = v
	}
	if v, ok := getEnvInt("TASKD_FOCUS_BREAK_MINUTES"); ok && v > 0 {
		cfg.FocusBreakMinutes = v
	}
	if v, ok := getEnvInt("TASKD_PRODUCTIVITY_AVAILABLE_MINUTES"); ok && v > 0 {
		cfg.ProductivityAvailableMins = v
	}
	if v, ok := getEnvInt("TASKD_SCHEDULER_BUFFER"); ok && v > 0 {
		cfg.SchedulerBuffer = v
	}
	return cfg
}

func getEnvInt(name string) (int, bool) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, false
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}

func getEnvBool(name string) (bool, bool) {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	if raw == "" {
		return false, false
	}
	switch raw {
	case "1", "true", "yes", "y", "on":
		return true, true
	case "0", "false", "no", "n", "off":
		return false, true
	default:
		return false, false
	}
}
