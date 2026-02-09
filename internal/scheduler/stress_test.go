package scheduler

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEngineStressConcurrentSchedule(t *testing.T) {
	engine := NewEngine(4096)
	engine.Start()
	defer engine.Stop()

	const workers = 8
	const perWorker = 200
	total := workers * perWorker

	now := time.Now().UTC()
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		w := w
		go func() {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				delay := time.Duration((w+i)%50+10) * time.Millisecond
				ev := ReminderEvent{
					ID:        fmt.Sprintf("w%d-%d", w, i),
					TaskID:    fmt.Sprintf("task-%d", i),
					Type:      "Soft",
					TriggerAt: now.Add(delay),
				}
				if err := engine.Schedule(ev); err != nil {
					t.Errorf("schedule failed: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	deadline := time.After(5 * time.Second)
	var received int64
	for atomic.LoadInt64(&received) < int64(total) {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting events: received=%d total=%d dropped=%d", received, total, engine.Dropped())
		case <-engine.C():
			atomic.AddInt64(&received, 1)
		}
	}

	if got := int(received); got != total {
		t.Fatalf("unexpected received count: got=%d want=%d", got, total)
	}
	if engine.Dropped() != 0 {
		t.Fatalf("expected zero drops with active consumer, got=%d", engine.Dropped())
	}
}
