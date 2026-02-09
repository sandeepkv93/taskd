package update

import (
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	domainmodel "github.com/sandeepkv93/taskd/internal/model"
)

func (m Model) handleRecurrenceEditorKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.recurrenceEditor.Active = false
		return m
	case "tab":
		// Toggle between known presets to keep editor predictable in first pass.
		switch m.recurrenceEditor.RuleType {
		case "every_weekday":
			m.recurrenceEditor.RuleType = "every_n_days"
		case "every_n_days":
			m.recurrenceEditor.RuleType = "every_n_weeks"
		case "every_n_weeks":
			m.recurrenceEditor.RuleType = "last_day_of_month"
		case "last_day_of_month":
			m.recurrenceEditor.RuleType = "after_completion"
		default:
			m.recurrenceEditor.RuleType = "every_weekday"
		}
	case "enter":
		m.computeRecurrencePreview()
	case "backspace":
		if len(m.recurrenceEditor.IntervalText) > 0 {
			m.recurrenceEditor.IntervalText = m.recurrenceEditor.IntervalText[:len(m.recurrenceEditor.IntervalText)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.recurrenceEditor.IntervalText += string(msg.Runes)
		}
	}
	return m
}

func (m *Model) computeRecurrencePreview() {
	interval := 1
	if v := strings.TrimSpace(m.recurrenceEditor.IntervalText); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			interval = parsed
		}
	}
	rule := domainmodel.RecurrenceRule{
		Type:     domainmodel.RecurrenceType(m.recurrenceEditor.RuleType),
		Interval: interval,
		Anchor:   time.Now().UTC(),
	}
	preview, err := rule.Preview(time.Now().UTC(), nil, 5)
	if err != nil {
		m.recurrenceEditor.Err = err.Error()
		m.recurrenceEditor.Preview = nil
		return
	}
	m.recurrenceEditor.Err = ""
	m.recurrenceEditor.Preview = make([]string, 0, len(preview))
	for _, item := range preview {
		m.recurrenceEditor.Preview = append(m.recurrenceEditor.Preview, item.Format("2006-01-02 15:04"))
	}
}
