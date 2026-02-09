package scheduler

import (
	"container/heap"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var ErrInvalidTriggerTime = errors.New("scheduler: invalid trigger time")

type ReminderEvent struct {
	ID        string
	TaskID    string
	Type      string
	TriggerAt time.Time
}

type queueItem struct {
	event ReminderEvent
}

type priorityQueue []queueItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].event.TriggerAt.Before(pq[j].event.TriggerAt)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(x any) {
	*pq = append(*pq, x.(queueItem))
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}

type Engine struct {
	mu      sync.Mutex
	queue   priorityQueue
	out     chan ReminderEvent
	wakeup  chan struct{}
	stopCh  chan struct{}
	doneCh  chan struct{}
	started bool
	stopped bool
	dropped uint64
}

func NewEngine(bufferSize int) *Engine {
	if bufferSize <= 0 {
		bufferSize = 1
	}
	return &Engine{
		queue:  make(priorityQueue, 0),
		out:    make(chan ReminderEvent, bufferSize),
		wakeup: make(chan struct{}, 1),
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}
}

func (e *Engine) C() <-chan ReminderEvent {
	return e.out
}

func (e *Engine) Start() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.started {
		return
	}
	e.started = true
	heap.Init(&e.queue)
	go e.loop()
}

func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.started || e.stopped {
		e.mu.Unlock()
		return
	}
	e.stopped = true
	close(e.stopCh)
	e.mu.Unlock()
	<-e.doneCh
}

func (e *Engine) Schedule(ev ReminderEvent) error {
	if ev.TriggerAt.IsZero() {
		return ErrInvalidTriggerTime
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.stopped {
		return errors.New("scheduler: engine stopped")
	}

	heap.Push(&e.queue, queueItem{event: ev})
	e.signalWakeup()
	return nil
}

func (e *Engine) Dropped() uint64 {
	return atomic.LoadUint64(&e.dropped)
}

func (e *Engine) loop() {
	defer close(e.doneCh)
	defer close(e.out)

	var timer *time.Timer
	for {
		next, hasNext := e.peek()
		if !hasNext {
			select {
			case <-e.wakeup:
				continue
			case <-e.stopCh:
				return
			}
		}

		wait := time.Until(next.TriggerAt)
		if wait < 0 {
			wait = 0
		}
		timer = resetTimer(timer, wait)

		select {
		case <-timer.C:
			due := e.popDue(time.Now().UTC())
			for _, ev := range due {
				select {
				case e.out <- ev:
				default:
					atomic.AddUint64(&e.dropped, 1)
				}
			}
		case <-e.wakeup:
			continue
		case <-e.stopCh:
			if timer != nil {
				stopTimer(timer)
			}
			return
		}
	}
}

func (e *Engine) signalWakeup() {
	select {
	case e.wakeup <- struct{}{}:
	default:
	}
}

func (e *Engine) peek() (ReminderEvent, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.queue) == 0 {
		return ReminderEvent{}, false
	}
	return e.queue[0].event, true
}

func (e *Engine) popDue(now time.Time) []ReminderEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	out := make([]ReminderEvent, 0)
	for len(e.queue) > 0 {
		next := e.queue[0].event
		if next.TriggerAt.After(now) {
			break
		}
		item := heap.Pop(&e.queue).(queueItem)
		out = append(out, item.event)
	}
	return out
}

func resetTimer(timer *time.Timer, d time.Duration) *time.Timer {
	if timer == nil {
		return time.NewTimer(d)
	}
	stopTimer(timer)
	timer.Reset(d)
	return timer
}

func stopTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}
