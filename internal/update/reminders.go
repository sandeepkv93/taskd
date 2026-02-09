package update

import (
	"fmt"
	"strings"
	"time"

	"github.com/sandeepkv93/taskd/internal/scheduler"
)

func (m *Model) applyReminderBehavior(ev scheduler.ReminderEvent, now time.Time) {
	t := strings.ToLower(strings.TrimSpace(ev.Type))
	switch t {
	case "hard":
		m.Status = StatusBar{Text: fmt.Sprintf("HARD reminder: %s", ev.ID), IsError: true}
	case "soft":
		m.Status = StatusBar{Text: fmt.Sprintf("soft reminder: %s", ev.ID), IsError: false}
		if !m.SoftFollowedUp[ev.ID] {
			m.SoftFollowedUp[ev.ID] = true
			m.rescheduleReminder(ev, now.Add(10*time.Minute))
		}
	case "nagging":
		m.Status = StatusBar{Text: fmt.Sprintf("nagging reminder: %s", ev.ID), IsError: false}
		if !m.ReminderAck[ev.ID] && !m.isTaskCompleted(ev.TaskID) {
			m.rescheduleReminder(ev, now.Add(2*time.Minute))
		}
	case "contextual":
		if inContextualWindowForRule(now, ev.RepeatRule) {
			m.Status = StatusBar{Text: fmt.Sprintf("contextual reminder: %s", ev.ID), IsError: false}
		} else {
			next := nextContextualWindowStartForRule(now, ev.RepeatRule)
			m.Status = StatusBar{Text: fmt.Sprintf("contextual deferred: %s -> %s", ev.ID, next.Format("15:04")), IsError: false}
			m.rescheduleReminder(ev, next)
		}
	default:
		m.Status = StatusBar{Text: fmt.Sprintf("reminder fired: %s", ev.ID), IsError: false}
	}
}

func (m *Model) rescheduleReminder(ev scheduler.ReminderEvent, next time.Time) {
	if m.Scheduler == nil {
		return
	}
	nextEv := ev
	nextEv.TriggerAt = next
	if err := m.Scheduler.Schedule(nextEv); err != nil {
		m.Status = StatusBar{Text: fmt.Sprintf("reminder reschedule failed: %v", err), IsError: true}
	}
}

func inContextualWindowForRule(now time.Time, rule string) bool {
	cfg := parseContextualRule(rule)
	if !cfg.allowsWeekday(now.Weekday()) {
		return false
	}
	return cfg.inWindow(now)
}

func nextContextualWindowStartForRule(now time.Time, rule string) time.Time {
	cfg := parseContextualRule(rule)
	loc := now.Location()
	for i := 0; i < 30; i++ {
		day := now.AddDate(0, 0, i)
		if !cfg.allowsWeekday(day.Weekday()) {
			continue
		}
		y, mo, d := day.Date()
		for _, w := range cfg.windows {
			candidate := time.Date(y, mo, d, w.StartHour, 0, 0, 0, loc)
			if candidate.After(now) {
				return candidate
			}
		}
	}
	return now.Add(24 * time.Hour)
}

type contextualWindow struct {
	StartHour int
	EndHour   int
}

type contextualRule struct {
	windows []contextualWindow
	days    map[time.Weekday]bool
}

func parseContextualRule(raw string) contextualRule {
	rule := contextualRule{
		windows: []contextualWindow{{StartHour: 18, EndHour: 22}}, // default: evening
	}
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return rule
	}

	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ';' || r == '|'
	})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "window=") {
			val := strings.TrimSpace(strings.TrimPrefix(part, "window="))
			if parsed := parseContextualWindows(val); len(parsed) > 0 {
				rule.windows = parsed
			}
			continue
		}
		if strings.HasPrefix(part, "days=") {
			val := strings.TrimSpace(strings.TrimPrefix(part, "days="))
			if parsed := parseContextualDays(val); len(parsed) > 0 {
				rule.days = parsed
			}
			continue
		}
	}

	// Backward-compatible keyword parsing for simpler rules.
	if strings.Contains(normalized, "morning") {
		rule.windows = []contextualWindow{{StartHour: 8, EndHour: 12}}
	}
	if strings.Contains(normalized, "afternoon") {
		rule.windows = []contextualWindow{{StartHour: 12, EndHour: 17}}
	}
	if strings.Contains(normalized, "evening") {
		rule.windows = []contextualWindow{{StartHour: 18, EndHour: 22}}
	}
	if strings.Contains(normalized, "weekend") {
		rule.days = map[time.Weekday]bool{time.Saturday: true, time.Sunday: true}
	}
	if strings.Contains(normalized, "weekday") {
		rule.days = map[time.Weekday]bool{
			time.Monday: true, time.Tuesday: true, time.Wednesday: true, time.Thursday: true, time.Friday: true,
		}
	}

	return rule
}

func parseContextualWindows(raw string) []contextualWindow {
	out := make([]contextualWindow, 0, 2)
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "morning":
			out = append(out, contextualWindow{StartHour: 8, EndHour: 12})
		case "afternoon":
			out = append(out, contextualWindow{StartHour: 12, EndHour: 17})
		case "evening":
			out = append(out, contextualWindow{StartHour: 18, EndHour: 22})
		}
	}
	return out
}

func parseContextualDays(raw string) map[time.Weekday]bool {
	parsed := make(map[time.Weekday]bool)
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "weekdays", "weekday":
			parsed[time.Monday] = true
			parsed[time.Tuesday] = true
			parsed[time.Wednesday] = true
			parsed[time.Thursday] = true
			parsed[time.Friday] = true
		case "weekends", "weekend":
			parsed[time.Saturday] = true
			parsed[time.Sunday] = true
		case "mon", "monday":
			parsed[time.Monday] = true
		case "tue", "tuesday":
			parsed[time.Tuesday] = true
		case "wed", "wednesday":
			parsed[time.Wednesday] = true
		case "thu", "thursday":
			parsed[time.Thursday] = true
		case "fri", "friday":
			parsed[time.Friday] = true
		case "sat", "saturday":
			parsed[time.Saturday] = true
		case "sun", "sunday":
			parsed[time.Sunday] = true
		}
	}
	return parsed
}

func (r contextualRule) allowsWeekday(d time.Weekday) bool {
	if len(r.days) == 0 {
		return true
	}
	return r.days[d]
}

func (r contextualRule) inWindow(now time.Time) bool {
	h := now.Hour()
	for _, w := range r.windows {
		if h >= w.StartHour && h < w.EndHour {
			return true
		}
	}
	return false
}

func (m Model) isTaskCompleted(taskID string) bool {
	if strings.TrimSpace(taskID) == "" {
		return false
	}
	return m.CompletedTasks[taskID]
}
