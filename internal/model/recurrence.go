package model

import (
	"errors"
	"fmt"
	"sort"
	"time"
)

type RecurrenceType string

const (
	RecurrenceEveryWeekday   RecurrenceType = "every_weekday"
	RecurrenceEveryNDays     RecurrenceType = "every_n_days"
	RecurrenceEveryNWeeks    RecurrenceType = "every_n_weeks"
	RecurrenceLastDayOfMonth RecurrenceType = "last_day_of_month"
	RecurrenceAfterComplete  RecurrenceType = "after_completion"
)

var (
	ErrInvalidRecurrenceType = errors.New("model: invalid recurrence type")
	ErrInvalidInterval       = errors.New("model: invalid recurrence interval")
	ErrCompletionRequired    = errors.New("model: completion time required for after_completion recurrence")
)

type RecurrenceRule struct {
	Type            RecurrenceType
	Interval        int
	Anchor          time.Time
	Weekdays        []time.Weekday
	AfterCompleteIn time.Duration
}

func (r RecurrenceRule) Validate() error {
	switch r.Type {
	case RecurrenceEveryWeekday, RecurrenceEveryNDays, RecurrenceEveryNWeeks, RecurrenceLastDayOfMonth, RecurrenceAfterComplete:
	default:
		return fmt.Errorf("%w: %q", ErrInvalidRecurrenceType, r.Type)
	}
	if r.Anchor.IsZero() {
		return errors.New("model: recurrence anchor is required")
	}
	if r.Interval <= 0 {
		return fmt.Errorf("%w: %d", ErrInvalidInterval, r.Interval)
	}
	if r.Type == RecurrenceEveryWeekday && len(r.Weekdays) > 0 {
		s := make([]int, 0, len(r.Weekdays))
		for _, d := range r.Weekdays {
			s = append(s, int(d))
		}
		sort.Ints(s)
		for i := 1; i < len(s); i++ {
			if s[i] == s[i-1] {
				return errors.New("model: duplicate weekday in recurrence")
			}
		}
	}
	return nil
}

func (r RecurrenceRule) NextAfter(from time.Time, completedAt *time.Time) (time.Time, error) {
	if err := r.Validate(); err != nil {
		return time.Time{}, err
	}
	base := from.UTC()
	if base.Before(r.Anchor.UTC()) {
		base = r.Anchor.UTC().Add(-time.Nanosecond)
	}

	switch r.Type {
	case RecurrenceEveryWeekday:
		return r.nextWeekday(base), nil
	case RecurrenceEveryNDays:
		return r.nextEveryNDays(base), nil
	case RecurrenceEveryNWeeks:
		return r.nextEveryNWeeks(base), nil
	case RecurrenceLastDayOfMonth:
		return r.nextLastDayOfMonth(base), nil
	case RecurrenceAfterComplete:
		if completedAt == nil || completedAt.IsZero() {
			return time.Time{}, ErrCompletionRequired
		}
		wait := r.AfterCompleteIn
		if wait <= 0 {
			wait = time.Duration(r.Interval) * 24 * time.Hour
		}
		return completedAt.UTC().Add(wait), nil
	default:
		return time.Time{}, fmt.Errorf("%w: %q", ErrInvalidRecurrenceType, r.Type)
	}
}

func (r RecurrenceRule) Preview(from time.Time, completedAt *time.Time, count int) ([]time.Time, error) {
	if count <= 0 {
		return []time.Time{}, nil
	}
	out := make([]time.Time, 0, count)
	cursor := from
	for i := 0; i < count; i++ {
		next, err := r.NextAfter(cursor, completedAt)
		if err != nil {
			return nil, err
		}
		out = append(out, next)
		cursor = next.Add(time.Nanosecond)
	}
	return out, nil
}

func (r RecurrenceRule) nextWeekday(from time.Time) time.Time {
	allowed := r.allowedWeekdays()
	probe := withAnchorClock(from.AddDate(0, 0, 1), r.Anchor)
	for {
		if allowed[probe.Weekday()] {
			return probe
		}
		probe = probe.AddDate(0, 0, 1)
	}
}

func (r RecurrenceRule) allowedWeekdays() map[time.Weekday]bool {
	if len(r.Weekdays) > 0 {
		m := make(map[time.Weekday]bool, len(r.Weekdays))
		for _, w := range r.Weekdays {
			m[w] = true
		}
		return m
	}
	return map[time.Weekday]bool{
		time.Monday:    true,
		time.Tuesday:   true,
		time.Wednesday: true,
		time.Thursday:  true,
		time.Friday:    true,
	}
}

func (r RecurrenceRule) nextEveryNDays(from time.Time) time.Time {
	anchor := r.Anchor.UTC()
	interval := time.Duration(r.Interval) * 24 * time.Hour
	if from.Before(anchor) {
		return anchor
	}
	elapsed := from.Sub(anchor)
	steps := int(elapsed / interval)
	next := anchor.Add(time.Duration(steps+1) * interval)
	return withAnchorClock(next, anchor)
}

func (r RecurrenceRule) nextEveryNWeeks(from time.Time) time.Time {
	anchor := r.Anchor.UTC()
	intervalDays := r.Interval * 7
	interval := time.Duration(intervalDays) * 24 * time.Hour
	if from.Before(anchor) {
		return anchor
	}
	elapsed := from.Sub(anchor)
	steps := int(elapsed / interval)
	next := anchor.Add(time.Duration(steps+1) * interval)
	return withAnchorClock(next, anchor)
}

func (r RecurrenceRule) nextLastDayOfMonth(from time.Time) time.Time {
	anchor := r.Anchor.UTC()
	y, m, _ := from.Date()
	loc := anchor.Location()

	candidate := lastDayAt(y, m, anchor, loc)
	if !candidate.After(from) {
		nextMonth := time.Date(y, m, 1, 0, 0, 0, 0, loc).AddDate(0, 1, 0)
		yy, mm, _ := nextMonth.Date()
		candidate = lastDayAt(yy, mm, anchor, loc)
	}
	return candidate
}

func lastDayAt(y int, m time.Month, anchor time.Time, loc *time.Location) time.Time {
	firstNextMonth := time.Date(y, m, 1, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), loc).AddDate(0, 1, 0)
	return firstNextMonth.AddDate(0, 0, -1)
}

func withAnchorClock(date time.Time, anchor time.Time) time.Time {
	y, m, d := date.Date()
	return time.Date(y, m, d, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), anchor.Location())
}
