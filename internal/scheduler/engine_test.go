package scheduler

import (
	"testing"
	"time"
)

func TestEngineEmitsInTriggerOrder(t *testing.T) {
	engine := NewEngine(8)
	engine.Start()
	defer engine.Stop()

	now := time.Now().UTC()
	if err := engine.Schedule(ReminderEvent{ID: "later", TriggerAt: now.Add(80 * time.Millisecond)}); err != nil {
		t.Fatalf("schedule later: %v", err)
	}
	if err := engine.Schedule(ReminderEvent{ID: "sooner", TriggerAt: now.Add(20 * time.Millisecond)}); err != nil {
		t.Fatalf("schedule sooner: %v", err)
	}

	first := waitEvent(t, engine.C(), time.Second)
	second := waitEvent(t, engine.C(), time.Second)
	if first.ID != "sooner" || second.ID != "later" {
		t.Fatalf("unexpected order: first=%s second=%s", first.ID, second.ID)
	}
}

func TestEngineNonBlockingDropsWhenConsumerIsSlow(t *testing.T) {
	engine := NewEngine(1)
	engine.Start()
	defer engine.Stop()

	now := time.Now().UTC().Add(20 * time.Millisecond)
	for i := 0; i < 25; i++ {
		if err := engine.Schedule(ReminderEvent{
			ID:        "evt",
			TriggerAt: now,
		}); err != nil {
			t.Fatalf("schedule event: %v", err)
		}
	}

	time.Sleep(120 * time.Millisecond)
	if engine.Dropped() == 0 {
		t.Fatalf("expected dropped events > 0, got %d", engine.Dropped())
	}
}

func TestScheduleValidatesTriggerTime(t *testing.T) {
	engine := NewEngine(1)
	if err := engine.Schedule(ReminderEvent{ID: "bad"}); err != ErrInvalidTriggerTime {
		t.Fatalf("expected ErrInvalidTriggerTime, got %v", err)
	}
}

func waitEvent(t *testing.T, ch <-chan ReminderEvent, timeout time.Duration) ReminderEvent {
	t.Helper()
	select {
	case ev := <-ch:
		return ev
	case <-time.After(timeout):
		t.Fatalf("timed out waiting for event")
		return ReminderEvent{}
	}
}
