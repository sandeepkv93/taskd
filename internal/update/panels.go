package update

import (
	"strings"
	"time"

	"github.com/sandeepkv93/taskd/internal/views"
)

func (m Model) renderCommandPalette() string {
	return views.RenderCommandPalette(m.Palette.Active, m.Palette.Input)
}

func (m Model) renderNotificationsView() string {
	if len(m.Notifications) == 0 {
		return ""
	}
	n := m.Notifications[len(m.Notifications)-1]
	return views.RenderNotification(n.Level, n.Body)
}

func (m Model) renderProductivityView() string {
	s := m.Productivity.Signals
	suggestions := make([]views.SuggestionData, 0, len(s.Suggestions))
	if len(s.Suggestions) > 0 {
		for _, sg := range s.Suggestions {
			suggestions = append(suggestions, views.SuggestionData{
				Title:   sg.Title,
				Energy:  sg.Energy,
				Minutes: sg.Minutes,
				Reason:  sg.Reason,
			})
		}
	}
	return views.RenderProductivityPanel(views.ProductivityPanelData{
		TemporalDebtScore: s.TemporalDebtScore,
		TemporalDebtLabel: s.TemporalDebtLabel,
		Suggestions:       suggestions,
	})
}

func (m Model) renderTodayMetadataPane() string {
	selected, ok := m.currentTodayItem()
	if !ok {
		return "metadata:\n(no selection)"
	}
	return views.RenderTodayMetadataPane(views.TodayMetadataData{
		SelectedID:       selected.ID,
		Priority:         selected.Priority,
		Tags:             selected.Tags,
		NotesEditorView:  m.notesArea.View(),
		MarkdownMetaView: m.metaViewport.View(),
	})
}

func (m Model) renderRecurrenceEditorIfVisible() string {
	return views.RenderRecurrenceEditor(views.RecurrenceEditorData{
		Active:       m.recurrenceEditor.Active,
		RuleType:     m.recurrenceEditor.RuleType,
		IntervalText: m.recurrenceEditor.IntervalText,
		ErrorText:    m.recurrenceEditor.Err,
		Preview:      m.recurrenceEditor.Preview,
	})
}

func (m *Model) notify(title, body, level string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	n := Notification{
		Title: title,
		Body:  body,
		Level: level,
		At:    time.Now().UTC(),
	}
	m.Notifications = append(m.Notifications, n)
	if len(m.Notifications) > 40 {
		m.Notifications = m.Notifications[len(m.Notifications)-40:]
	}
	if m.DesktopEnabled && m.notifier != nil {
		_ = m.notifier.Send(n)
	}
}
