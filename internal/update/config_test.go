package update

import "testing"

func TestRuntimeConfigDefaults(t *testing.T) {
	cfg := DefaultRuntimeConfig()
	if cfg.FocusWorkMinutes != 25 || cfg.FocusBreakMinutes != 5 {
		t.Fatalf("unexpected focus defaults: %+v", cfg)
	}
	if cfg.ProductivityAvailableMins != 60 || cfg.SchedulerBuffer != 64 {
		t.Fatalf("unexpected runtime defaults: %+v", cfg)
	}
	if cfg.CompletionStatePath != ".taskd_state.json" {
		t.Fatalf("unexpected completion state default: %+v", cfg)
	}
}

func TestRuntimeConfigFromEnv(t *testing.T) {
	t.Setenv("TASKD_DESKTOP_NOTIFICATIONS", "true")
	t.Setenv("TASKD_FOCUS_WORK_MINUTES", "30")
	t.Setenv("TASKD_FOCUS_BREAK_MINUTES", "7")
	t.Setenv("TASKD_PRODUCTIVITY_AVAILABLE_MINUTES", "45")
	t.Setenv("TASKD_SCHEDULER_BUFFER", "128")
	t.Setenv("TASKD_STATE_FILE", "state/custom.json")

	cfg := RuntimeConfigFromEnv(DefaultRuntimeConfig())
	if !cfg.DesktopNotifications {
		t.Fatal("expected desktop notifications true from env")
	}
	if cfg.FocusWorkMinutes != 30 || cfg.FocusBreakMinutes != 7 {
		t.Fatalf("unexpected focus config: %+v", cfg)
	}
	if cfg.ProductivityAvailableMins != 45 || cfg.SchedulerBuffer != 128 {
		t.Fatalf("unexpected config overrides: %+v", cfg)
	}
	if cfg.CompletionStatePath != "state/custom.json" {
		t.Fatalf("unexpected completion path override: %+v", cfg)
	}
}
