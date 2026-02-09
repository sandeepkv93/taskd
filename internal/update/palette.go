package update

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/commands"
)

func (m Model) handlePaletteKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.Palette.Active = false
		m.Palette.Input = ""
		m.commandInput.SetValue("")
		m.commandInput.Blur()
		m.Status = StatusBar{Text: "command palette closed", IsError: false}
	case "enter":
		m.Palette.Input = m.commandInput.Value()
		m = m.executePaletteCommand()
	default:
		if msg.Type == tea.KeyRunes {
			m.commandInput.SetValue(m.commandInput.Value() + string(msg.Runes))
			m.Palette.Input = m.commandInput.Value()
			return m
		}
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
		_ = cmd
		m.Palette.Input = m.commandInput.Value()
	}
	return m
}

func (m Model) executePaletteCommand() Model {
	raw := strings.TrimSpace(m.Palette.Input)
	cmd, err := commands.Parse(raw)
	if err != nil {
		m.Status = StatusBar{Text: err.Error(), IsError: true}
		m.Palette.Active = false
		m.Palette.Input = ""
		return m
	}

	res, err := commands.Execute(cmd, commands.Handlers{
		Add: func(a commands.AddArgs) (commands.Result, error) {
			m.CurrentView = ViewInbox
			m.addInboxItem(a.Title)
			return commands.Result{Message: fmt.Sprintf("added inbox task: %s", a.Title)}, nil
		},
		Snooze: func(s commands.SnoozeArgs) (commands.Result, error) {
			applied := 0
			for i := range m.Today.Items {
				if strings.EqualFold(s.Target, "overdue") && m.Today.Items[i].Bucket == TodayBucketOverdue {
					m.Today.Items[i].Bucket = TodayBucketAnytime
					m.Today.Items[i].Notes = strings.TrimSpace(m.Today.Items[i].Notes + " | snoozed " + s.For)
					applied++
				}
			}
			if applied == 0 {
				return commands.Result{}, &commands.CommandError{Code: commands.ErrCodeInvalidArgument, Message: "no matching items for snooze target"}
			}
			return commands.Result{Message: fmt.Sprintf("snoozed %d task(s) for %s", applied, s.For)}, nil
		},
		Show: func(s commands.ShowArgs) (commands.Result, error) {
			if s.Tag != "" {
				m.Filter.Tag = s.Tag
				return commands.Result{Message: fmt.Sprintf("show filter applied: tag=%s", s.Tag)}, nil
			}
			return commands.Result{Message: fmt.Sprintf("show %s", s.Subject)}, nil
		},
		Reschedule: func(r commands.RescheduleArgs) (commands.Result, error) {
			if r.Target != "selected" {
				return commands.Result{}, &commands.CommandError{Code: commands.ErrCodeInvalidArgument, Message: "reschedule currently supports target: selected"}
			}
			applied := 0
			for i := range m.Inbox.Items {
				item := m.Inbox.Items[i]
				if m.Inbox.Selected[item.ID] {
					m.Inbox.Items[i].ScheduledFor = r.When
					applied++
				}
			}
			if applied == 0 {
				return commands.Result{}, &commands.CommandError{Code: commands.ErrCodeInvalidArgument, Message: "no selected inbox items to reschedule"}
			}
			return commands.Result{Message: fmt.Sprintf("rescheduled %d selected item(s) to %s", applied, r.When)}, nil
		},
	})
	if err != nil {
		m.Status = StatusBar{Text: err.Error(), IsError: true}
		m.notify("Command Failed", err.Error(), "error")
	} else {
		m.Status = StatusBar{Text: res.Message, IsError: false}
		m.notify("Command", res.Message, "info")
		m.refreshProductivitySignals()
	}

	m.Palette.Active = false
	m.Palette.Input = ""
	m.commandInput.SetValue("")
	return m
}
